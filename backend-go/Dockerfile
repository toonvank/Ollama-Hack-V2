# Build stage
FROM golang:1.22-alpine AS builder

# Install certificates for external HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with flags to reduce binary size (-w -s strips debug info)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -a -installsuffix cgo -o server ./cmd/server

# Final stage - using scratch (empty image) for the smallest possible size
FROM scratch

# Copy TLS certificates and the compiled binary
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/server /server

# Expose port
EXPOSE 8000

# Run the application
ENTRYPOINT ["/server"]
