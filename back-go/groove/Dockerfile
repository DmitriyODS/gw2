# Микросервис «Мой Groove» (groovesvc): HTTP/Fiber :8094 + gRPC :9094.
# Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY groove/go.mod groove/go.sum groove/
WORKDIR /src/groove
RUN go mod download
COPY pkg/ /src/pkg/
COPY groove/ /src/groove/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/groovesvc ./cmd/groovesvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D groovesvc
COPY --from=build /out/groovesvc /usr/local/bin/groovesvc
USER groovesvc
EXPOSE 8094 9094
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8094/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["groovesvc"]
