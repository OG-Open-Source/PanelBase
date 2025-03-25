# Build stage
FROM golang:1.19-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go.mod and go.sum files
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o panelbase ./cmd/panelbase

# Runtime stage
FROM alpine:latest

# Set working directory
WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create directories for logs and data
RUN mkdir -p /app/logs /app/configs /app/web

# Copy binary from builder stage
COPY --from=builder /app/panelbase /app/panelbase

# Copy config files and web resources
COPY --from=builder /app/configs /app/configs
COPY --from=builder /app/web /app/web

# Expose default port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/panelbase"] 