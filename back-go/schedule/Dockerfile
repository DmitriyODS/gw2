# Микросервис расписаний (schedulesvc): HTTP/Fiber :8110 — регулярные
# расписания с циклом недель, занятия, шаринг на чтение. Контекст сборки —
# back-go/ (общий модуль pkg подключается через replace ../pkg в go.mod). Стадия
# сборки бежит на платформе билдера ($BUILDPLATFORM): при сборке amd64-образа на
# Apple Silicon компилятор Go работает нативно (кросс через GOOS/GOARCH), а не
# под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY schedule/go.mod schedule/go.sum schedule/
WORKDIR /src/schedule
RUN go mod download
COPY pkg/ /src/pkg/
COPY schedule/ /src/schedule/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/schedulesvc ./cmd/schedulesvc

FROM alpine:3.21
# tzdata — у расписания своя зона (time.LoadLocation при проверке настроек).
RUN apk add --no-cache ca-certificates tzdata && adduser -D schedulesvc
COPY --from=build /out/schedulesvc /usr/local/bin/schedulesvc
USER schedulesvc
EXPOSE 8110
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8110/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["schedulesvc"]
