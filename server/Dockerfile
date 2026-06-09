FROM golang:1.25-alpine AS builder

WORKDIR /build

# 以仓库根目录作为构建上下文，保证 server/go.mod 中的 ../pkg replace 可用。
COPY server/go.mod server/go.sum ./server/
COPY pkg/go.mod pkg/go.sum ./pkg/
RUN cd server && go mod download

COPY server ./server
COPY pkg ./pkg
RUN cd server && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/control-plane ./cmd/control-plane
RUN cd server && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/relay-server ./cmd/relay
RUN cd server && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/nat-detector ./cmd/nat-detector
RUN cd server && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/bin/dashboard ./cmd/dashboard

FROM node:22-alpine AS web-builder

WORKDIR /web
COPY server/web/package*.json ./
RUN npm ci
COPY server/web ./
RUN npm run build

# --- Runtime image ---
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /app

COPY --from=builder /build/bin/ /app/
COPY --from=web-builder /web/dist/ /app/web/dashboard/

EXPOSE 7000 7443 8080 9090 3478/udp

ENTRYPOINT ["/app/relay-server"]
