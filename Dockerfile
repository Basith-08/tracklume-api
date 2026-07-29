# syntax=docker/dockerfile:1.7
FROM golang:1.25.0-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/tracklume-api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/tracklume-migrate ./cmd/migrate

FROM alpine:3.22.1 AS runtime
RUN apk add --no-cache ca-certificates tzdata && addgroup -S -g 10001 tracklume && adduser -S -D -H -u 10001 -G tracklume tracklume
WORKDIR /app
COPY --from=build /out/tracklume-api /app/tracklume-api
COPY --from=build /out/tracklume-migrate /app/tracklume-migrate
COPY migrations ./migrations
COPY openapi.yaml ./openapi.yaml
RUN chown -R tracklume:tracklume /app
USER tracklume
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1
ENTRYPOINT ["/app/tracklume-api"]
