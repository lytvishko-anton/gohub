# Gohub - Concurrent GitOps Webhook Deployer

A lightweight, high-performance continuous delivery daemon written in Go. This tool securely intercepts repository webhook lifecycles (GitHub/GitLab) and triggers automated infrastructure deployment workflows asynchronously.

## Key Features & Architecture

* **Asynchronous Execution Queue:** Leverages Go’s native goroutines and buffered channels to instantly offload heavy deployment scripts, ensuring the web server never freezes or blocks incoming HTTP traffic.
* **Burst Traffic & Failure Protection:** Implements non-blocking channel selection mechanisms to act as a natural rate-limiter, protecting internal infrastructure from webhook burst spam.
* **Edge Security Handshakes:** Features cryptographically secure validation pipelines, calculating incoming payload signatures via `crypto/hmac` (SHA-256) for GitHub and utilizing constant-time string comparisons (`crypto/subtle`) for GitLab headers.
* **Cloud-Native & Containerized:** Optimized via a minimal multi-stage Docker build to keep execution artifacts lightweight and completely portable.

---

## Project Structure

```text
gohub/
├── cmd/
│   └── deployer/
│       └── main.go       # Orchestration entry point & graceful OS shutdown
├── internal/
│   ├── server/           # Asynchronous worker engine & HTTP routing
│   └── webhook/          # Cryptographic signature validation layers
├── scripts/
│   └── deploy.sh         # Target shell deployment script
├── Dockerfile            # Optimized multi-stage Docker file
└── docker-compose.yml    # Declarative runtime specifications

```
---

## Quick Start
1. Configuration
Define your cryptographic webhook secrets inside the local environment setup within docker-compose.yml:
```text
environment:
  - PORT=8080
  - GITHUB_WEBHOOK_SECRET=your_secure_github_hmac_key
  - GITLAB_WEBHOOK_SECRET=your_secure_gitlab_token
```
2. Execution
```text
docker-compose up --build
```
