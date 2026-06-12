# Realtime-шлюз (gatewaysvc): HTTP/Fiber :8096 — WebSocket /ws (комнаты
# all/user_{id}), presence в Redis, ринг-фаза звонков (gRPC callsvc),
# доставка событий всех Redis-каналов gw2:*:events клиентам.
# Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY gateway/go.mod gateway/go.sum gateway/
WORKDIR /src/gateway
RUN go mod download
COPY pkg/ /src/pkg/
COPY gateway/ /src/gateway/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/gatewaysvc ./cmd/gatewaysvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D gatewaysvc
COPY --from=build /out/gatewaysvc /usr/local/bin/gatewaysvc
USER gatewaysvc
EXPOSE 8096
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8096/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["gatewaysvc"]
