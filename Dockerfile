# syntax=docker/dockerfile:1.7
ARG BUILD_IMAGE=golang:1.25.0-bookworm
ARG RUNTIME_IMAGE=alpine:3.22.1

FROM ${BUILD_IMAGE} AS build
WORKDIR /src
COPY go.mod go.sum ./
ARG INSTALL_COMMAND="go mod download"
RUN --mount=type=cache,target=/go/pkg/mod sh -c "${INSTALL_COMMAND}"
COPY . .
ARG BUILD_COMMAND="CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/tracklume-api ./cmd/api && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/tracklume-migrate ./cmd/migrate && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/tracklume-admin ./cmd/admin"
ARG RUNTIME_PREPARE_COMMAND="cp -a /out/tracklume-api /out/tracklume-migrate /out/tracklume-admin /src/migrations /src/openapi.yaml /src/scripts/tracklume-start /runtime/"
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build sh -c "${BUILD_COMMAND}" \
    && mkdir -p /runtime \
    && sh -c "${RUNTIME_PREPARE_COMMAND}" \
    && chmod 0755 /runtime/tracklume-start

FROM ${RUNTIME_IMAGE} AS runtime
RUN apk add --no-cache ca-certificates tzdata && addgroup -S -g 10001 tracklume && adduser -S -D -H -u 10001 -G tracklume tracklume
WORKDIR /app
ARG START_COMMAND="/app/tracklume-start"
ARG HEALTHCHECK_COMMAND="wget -qO- http://127.0.0.1:8080/health >/dev/null"
ENV START_COMMAND="${START_COMMAND}" \
    HEALTHCHECK_COMMAND="${HEALTHCHECK_COMMAND}"
COPY --from=build --chown=1001:1001 /runtime/ ./
RUN chown -R tracklume:tracklume /app
USER tracklume
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s CMD sh -c "${HEALTHCHECK_COMMAND}"
CMD ["sh", "-c", "exec $START_COMMAND"]
