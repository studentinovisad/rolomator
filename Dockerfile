# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Install git (needed for go modules)
RUN apk add --no-cache git

# Copy go.mod and go.sum first (for caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o bot main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the binary and config from builder
COPY --from=builder /app/bot .
COPY --from=builder /app/config.json .

# Minimal dependencies for SSL if needed (optional)
RUN apk add --no-cache ca-certificates

# Run the bot (token passed via env at runtime)
CMD ["./bot"]
