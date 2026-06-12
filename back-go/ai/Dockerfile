# Микросервис работы с ИИ (aisvc): HTTP/Fiber :8093 + gRPC :9093.
# Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY ai/go.mod ai/go.sum ai/
WORKDIR /src/ai
RUN go mod download
COPY pkg/ /src/pkg/
COPY ai/ /src/ai/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/aisvc ./cmd/aisvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D aisvc
COPY --from=build /out/aisvc /usr/local/bin/aisvc
USER aisvc
EXPOSE 8093 9093
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8093/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["aisvc"]
