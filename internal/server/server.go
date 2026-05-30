package server

import (
	"context"
	// "fmt"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"

	// "time"
	"gohub/internal/webhook" // Adjust this import path if your go.mod module name is different
)

// Job represents a deployment task passed from the webhook handler to our worker
type Job struct {
	ProjectName string
	Ref         string
}

type Server struct {
	port         string
	gitlabSecret string
	githubSecret string
	jobQueue     chan Job
}

func NewServer(port, gitlabSecret, githubSecret string) *Server {
	return &Server{
		port:         port,
		gitlabSecret: gitlabSecret,
		githubSecret: githubSecret,
		jobQueue:     make(chan Job, 100),
	}
}

// Start boots up the background worker and the HTTP server
func (s *Server) Start(ctx context.Context) error {
	// 1. Spin up the background worker goroutine
	go s.worker(ctx)

	// 2. Setup standard library routing (Go 1.22+ syntax)
	mux := http.NewServeMux()

	// Wrap our webhook handler so it has access to the job queue
	mux.HandleFunc("POST /webhook/gitlab", s.handleGitLabWebhook())
	mux.HandleFunc("POST /webhook/github", s.handleGitHubWebhook())

	srv := &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}

	log.Printf("Gohub server listening on port %s...", s.port)
	return srv.ListenAndServe()
}

// worker runs concurrently, processing one deployment job at a time
func (s *Server) worker(ctx context.Context) {
	log.Println("Background deployment worker started successfully")
	for {
		select {
		case <-ctx.Done():
			log.Println("Worker shutting down...")
			return
		case job := <-s.jobQueue:
			log.Printf("Starting deployment for project: %s (%s)", job.ProjectName, job.Ref)

			// Simulate executing your bash script or docker commands
			cmd := exec.Command("/bin/bash", "./scripts/deploy.sh", job.ProjectName)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("❌ Deployment FAILED for %s: %v\nOutput: %s", job.ProjectName, err, string(output))
				continue
			}

			log.Printf("✅ Deployment SUCCESSFUL for %s\nOutput: %s", job.ProjectName, string(output))
		}
	}
}

// handleGitLabWebhook adapts our previous standalone handler to interface with our server struct
func (s *Server) handleGitLabWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Only allow POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 2. Extract and securely verify the GitLab secret token
		incomingToken := r.Header.Get("X-Gitlab-Token")
		if incomingToken == "" {
			http.Error(w, "Missing security token", http.StatusUnauthorized)
			return
		}

		// Constant-time cryptographic comparison against our server config
		if subtle.ConstantTimeCompare([]byte(incomingToken), []byte(s.gitlabSecret)) != 1 {
			http.Error(w, "Unauthorized payload", http.StatusUnauthorized)
			return
		}

		// 3. Safely decode the JSON body using our webhook package's struct
		var payload webhook.GitLabPayload
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// 4. Only queue deployment if it's a push to the main branch
		if payload.ObjectKind == "push" && payload.Ref == "refs/heads/main" {
			job := Job{
				ProjectName: payload.Project.Name,
				Ref:         payload.Ref,
			}

			// 5. Drop the job onto the channel without blocking the HTTP response
			select {
			case s.jobQueue <- job:
				w.WriteHeader(http.StatusAccepted)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"status":"queued","message":"Deployment triggered"}`))
			default:
				// Buffer is full (100 jobs pending)
				log.Printf("⚠️ Alert: Job queue is full! Dropping deployment for %s", job.ProjectName)
				http.Error(w, "Server overloaded, queue full", http.StatusServiceUnavailable)
			}
			return
		}

		// If it's a webhook event we don't care about (like a comment or tag)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ignored","message":"Not a main branch push event"}`))
	}
}

func (s *Server) handleGitHubWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 1. Verify signature and get raw body bytes
		bodyBytes, valid := webhook.VerifyGitHubSignature(r, s.githubSecret)
		if !valid {
			log.Println("❌ Webhook signature validation failed")
			http.Error(w, "Unauthorized signature mismatch", http.StatusUnauthorized)
			return
		}

		// 2. Catch the GitHub Event type from the header immediately
		githubEvent := r.Header.Get("X-GitHub-Event")

		// 3. Handle GitHub's initial handshake setup ("ping") right away!
		if githubEvent == "ping" {
			log.Println("🏓 GitHub ping received! Handshake successful.")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"pong"}`))
			return
		}

		// 4. Parse JSON payload if it's an actual event lifecycle step
		var payload webhook.GitHubPayload
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// 5. Only process actual push events to main
		if githubEvent == "push" {
			if payload.Ref != "refs/heads/main" {
				log.Printf("ℹ️ Push ignored: branch was %s, not main", payload.Ref)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ignored","message":"Not main branch"}`))
				return
			}

			job := Job{
				ProjectName: payload.Repository.Name,
				Ref:         payload.Ref,
			}

			// Drop onto background channel pipeline
			select {
			case s.jobQueue <- job:
				w.WriteHeader(http.StatusAccepted)
				w.Write([]byte(`{"status":"queued"}`))
			default:
				log.Printf("⚠️ Job queue full, skipping push for %s", job.ProjectName)
				http.Error(w, "Queue full", http.StatusServiceUnavailable)
			}
			return
		}

		// Fallback for any other event types (issues, stars, etc.)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ignored"}`))
	}
}
