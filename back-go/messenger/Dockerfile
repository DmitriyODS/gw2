# Микросервис мессенджера (msgsvc): HTTP/Fiber :8092 + gRPC :9092.
# Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY messenger/go.mod messenger/go.sum messenger/
WORKDIR /src/messenger
RUN go mod download
COPY pkg/ /src/pkg/
COPY messenger/ /src/messenger/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/msgsvc ./cmd/msgsvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D msgsvc
COPY --from=build /out/msgsvc /usr/local/bin/msgsvc
USER msgsvc
EXPOSE 8092 9092
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8092/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["msgsvc"]
