# Микросервис задач (tasksvc): HTTP/Fiber :8095 — ядро платформы
# (задачи, юниты, типы юнитов, этапы, отделы, комментарии, статистика).
# Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY tasks/go.mod tasks/go.sum tasks/
WORKDIR /src/tasks
RUN go mod download
COPY pkg/ /src/pkg/
COPY tasks/ /src/tasks/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/tasksvc ./cmd/tasksvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D tasksvc
COPY --from=build /out/tasksvc /usr/local/bin/tasksvc
USER tasksvc
EXPOSE 8095
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8095/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["tasksvc"]
