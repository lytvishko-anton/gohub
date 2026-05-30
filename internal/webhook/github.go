package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

type GitHubPayload struct {
	Ref        string `json:"ref"` // e.g., "refs/heads/main"
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
}

func VerifyGitHubSignature(r *http.Request, secret string) ([]byte, bool) {
	signatureHeader := r.Header.Get("X-Hub-Signature-256")
	if signatureHeader == "" || !strings.HasPrefix(signatureHeader, "sha256=") {
		return nil, false
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, false
	}

	expectedSignatureHex := strings.TrimPrefix(signatureHeader, "sha256=")
	expectedSignature, err := hex.DecodeString(expectedSignatureHex)
	if err != nil {
		return nil, false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(bodyBytes)
	calculatedSignature := mac.Sum(nil)

	return bodyBytes, hmac.Equal(calculatedSignature, expectedSignature)
}
