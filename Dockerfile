# Stage 1: Build binary
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy dependency definitions
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source code
COPY . .

# Build statically linked binary (< 15MB)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o server ./cmd/server

# Stage 2: Minimal runner
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy compiled binary from builder
COPY --from=builder /app/server /app/server

USER appuser

EXPOSE 8080

CMD ["./server"]
