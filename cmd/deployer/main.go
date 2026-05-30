package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gohub/internal/server"
)

func main() {
	// 1. Fetch configuration from environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Fallback default
	}

	githubSecret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	gitlabSecret := os.Getenv("GITLAB_WEBHOOK_SECRET")

	if githubSecret == "" && gitlabSecret == "" {
		log.Println("⚠️ Warning: Neither GITHUB_WEBHOOK_SECRET nor GITLAB_WEBHOOK_SECRET was provided.")
		log.Println("The server will boot, but webhook requests will fail validation.")
	}

	// 2. Instantiate our server configuration
	srv := server.NewServer(port, gitlabSecret, githubSecret)

	// 3. Create a context that listens for termination signals (Ctrl+C, Docker stop)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. Run the server in a separate goroutine so it doesn't block our main thread
	serverErrors := make(chan error, 1)
	go func() {
		if err := srv.Start(ctx); err != nil {
			serverErrors <- err
		}
	}()

	log.Println("🚀 Gohub GitOps engine successfully launched!")

	// 5. Block main execution until we get a shutdown signal or a server error occurs
	select {
	case err := <-serverErrors:
		log.Fatalf("❌ Critical server error triggered shutdown: %v", err)
	case <-ctx.Done():
		log.Println("🛑 Shutdown signal received. Cleaning up resources...")

		// Give any active background worker jobs a brief grace period to finish executing
		_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Println("👋 Gohub stopped cleanly. See ya!")
	}
}
