package webhook

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

type GitLabPayload struct {
	ObjectKind string `json:"object_kind"`
	Ref        string `json:"ref"`
	Project    struct {
		Name string `json:"name"`
		URL  string `json:"web_url"`
	} `json:"project"`
}

func HandleGitLabWebhook(expectedToken string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		incomingToken := r.Header.Get("X-Gitlab-Token")
		if incomingToken == "" {
			http.Error(w, "Missing security token", http.StatusUnauthorized)
			return
		}

		if subtle.ConstantTimeCompare([]byte(incomingToken), []byte(expectedToken)) != 1 {
			http.Error(w, "Unauthorized payload", http.StatusUnauthorized)
			return
		}

		var payload GitLabPayload
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		if payload.ObjectKind != "push" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Event ignored: not a push event"))
			return
		}

		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Webhook received, deployment triggered asynchronously"))
	}
}
