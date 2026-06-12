# Микросервис звонков (callsvc): gRPC :9090 + HTTP/Fiber :8090.
# Стадия сборки бежит на платформе билдера ($BUILDPLATFORM): при сборке
# amd64-образа на Apple Silicon компилятор Go работает нативно (кросс через
# GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/callsvc ./cmd/callsvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D callsvc
COPY --from=build /out/callsvc /usr/local/bin/callsvc
USER callsvc
EXPOSE 8090 9090
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8090/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["callsvc"]
