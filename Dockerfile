FROM golang:1.19-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o panelbase ./cmd/panelbase

# Runtime image
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/panelbase .
COPY --from=builder /app/.env .
COPY --from=builder /app/web /app/web

# Create necessary directories
RUN mkdir -p data logs web/static/css web/static/js web/static/img

# Set execution permissions
RUN chmod +x /app/panelbase

# Expose the port from .env (default 8080)
EXPOSE 8080

# Run the application
CMD ["/app/panelbase"]
