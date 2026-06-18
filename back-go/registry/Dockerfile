# Микросервис реестров (registrysvc): HTTP/Fiber :8099 — настраиваемые
# таблицы-справочники компаний (структура полей + записи).
# Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY registry/go.mod registry/go.sum registry/
WORKDIR /src/registry
RUN go mod download
COPY pkg/ /src/pkg/
COPY registry/ /src/registry/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/registrysvc ./cmd/registrysvc

FROM alpine:3.21
# su-exec — сброс root→registrysvc в entrypoint после chown общего uploads-тома.
RUN apk add --no-cache ca-certificates su-exec && adduser -D registrysvc
COPY --from=build /out/registrysvc /usr/local/bin/registrysvc
COPY registry/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh
# USER не задаём: контейнер стартует root'ом, entrypoint чинит права на
# uploads-том и сам сбрасывает привилегии на registrysvc (su-exec).
EXPOSE 8099
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8099/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["registrysvc"]
