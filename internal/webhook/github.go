package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

// GitHubPayload extracts the repository info and git ref from a GitHub push event
type GitHubPayload struct {
	Ref        string `json:"ref"` // e.g., "refs/heads/main"
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
}

// VerifyGitHubSignature checks if the incoming payload matches the HMAC-SHA256 signature sent by GitHub
func VerifyGitHubSignature(r *http.Request, secret string) ([]byte, bool) {
	// GitHub sends the signature in the "X-Hub-Signature-256" header as "sha256=hex_encoded_signature"
	signatureHeader := r.Header.Get("X-Hub-Signature-256")
	if signatureHeader == "" || !strings.HasPrefix(signatureHeader, "sha256=") {
		return nil, false
	}

	// Read the raw body bytes (needed to calculate the HMAC)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, false
	}

	// Extract the hex-encoded signature string
	expectedSignatureHex := strings.TrimPrefix(signatureHeader, "sha256=")
	expectedSignature, err := hex.DecodeString(expectedSignatureHex)
	if err != nil {
		return nil, false
	}

	// Calculate our own HMAC-SHA256 using the raw body and our secret key
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(bodyBytes)
	calculatedSignature := mac.Sum(nil)

	// Cryptographically compare the two signatures safely
	return bodyBytes, hmac.Equal(calculatedSignature, expectedSignature)
}
