# Микросервис корпоративного портала (portalsvc): HTTP/Fiber :8102 — посты
# компании (с вложениями), комментарии, реакции, закрепление, тематические
# разделы; gRPC :9102 — системные посты (CreateSystemPost, зовёт petsvc,
# внутрисетевой). Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY portal/go.mod portal/go.sum portal/
WORKDIR /src/portal
RUN go mod download
COPY pkg/ /src/pkg/
COPY portal/ /src/portal/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/portalsvc ./cmd/portalsvc

FROM alpine:3.21
# su-exec — сброс root→portalsvc в entrypoint после chown общего uploads-тома.
RUN apk add --no-cache ca-certificates su-exec && adduser -D portalsvc
COPY --from=build /out/portalsvc /usr/local/bin/portalsvc
COPY portal/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
# USER не задаём: контейнер стартует root'ом, entrypoint чинит права на
# uploads-том и сам сбрасывает привилегии на portalsvc (su-exec).
EXPOSE 8102 9102
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8102/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["portalsvc"]
