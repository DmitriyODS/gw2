# Микросервис напоминаний (remindersvc): HTTP/Fiber :8106 — личные напоминания
# пользователей (разовые и повторяющиеся, привязки к ежедневнику/календарю) и
# планировщик их срабатывания. Контекст сборки — back-go/ (общий модуль pkg
# подключается через replace ../pkg в go.mod). Стадия сборки бежит на платформе
# билдера ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор
# Go работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY reminder/go.mod reminder/go.sum reminder/
WORKDIR /src/reminder
RUN go mod download
COPY pkg/ /src/pkg/
COPY reminder/ /src/reminder/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/remindersvc ./cmd/remindersvc

FROM alpine:3.21
# tzdata — повторы считаются в зоне пользователя (time.LoadLocation).
RUN apk add --no-cache ca-certificates tzdata && adduser -D remindersvc
COPY --from=build /out/remindersvc /usr/local/bin/remindersvc
USER remindersvc
EXPOSE 8106
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8106/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["remindersvc"]
