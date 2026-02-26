# Build Stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the web server
RUN go build -o gopher-web cmd/web/main.go

# Runtime Stage
FROM alpine:latest

# Install certificates for HTTPS (if needed properly for external calls)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/gopher-web .

# Create storage directory for uploads
RUN mkdir -p storage

# Use PORT environment variable if provided, default to 8080
ENV PORT=8080

EXPOSE 8080

CMD ["./gopher-web"]