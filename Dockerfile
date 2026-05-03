# Step 1: Build the Go binary
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o chat-app cmd/server/main.go

# Step 2: Final lightweight image
FROM alpine:latest
WORKDIR /root/
# Copy the binary from the builder
COPY --from=builder /app/chat-app .
# Copy the web folder so the binary can find index.html
COPY --from=builder /app/web ./web

EXPOSE 8080
CMD ["./chat-app"]