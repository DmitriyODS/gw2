# Микросервис подписок и магазина (billingsvc): HTTP :8107 — REST /api/billing
# (витрина, покупки, заказы, промокоды, «Аудит платформы») и вебхук платёжного
# шлюза; gRPC :9107 — лимиты тарифа, проверка хранилища и токены доступа к ИИ
# для остальных сервисов. Контекст сборки — back-go/ (общий модуль pkg
# подключается через replace ../pkg в go.mod). Стадия сборки бежит на платформе
# билдера ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор
# Go работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY billing/go.mod billing/go.sum billing/
WORKDIR /src/billing
RUN go mod download
COPY pkg/ /src/pkg/
COPY billing/ /src/billing/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/billingsvc ./cmd/billingsvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D billingsvc
COPY --from=build /out/billingsvc /usr/local/bin/billingsvc
USER billingsvc
EXPOSE 8107 9107
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8107/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["billingsvc"]
