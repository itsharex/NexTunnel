FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/control-plane ./cmd/control-plane
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/relay-server ./cmd/relay
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/nat-detector ./cmd/nat-detector

# --- Runtime image ---
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/bin/ /app/

EXPOSE 7000

ENTRYPOINT ["/app/relay-server"]
