# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
# RUN go mod download # Uncomment once you add dependencies
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o gitops-deployer ./cmd/deployer/main.go

# Stage 2: Final lightweight image
FROM alpine:latest
RUN apk --no-cache add ca-certificates bash git
WORKDIR /root/
COPY --from=builder /app/gitops-deployer .
COPY scripts/ ./scripts/

EXPOSE 8080
CMD ["./gitops-deployer"]