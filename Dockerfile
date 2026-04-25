# Build stage
FROM golang:1.26.1 AS builder

WORKDIR /app

# Copy go.mod and go.sum first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary from cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Copy the compiled binary
COPY --from=builder /app/server .

# Copy static files (your web UI)
COPY web ./web

EXPOSE 8080

CMD ["./server"]
