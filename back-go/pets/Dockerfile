# Микросервис «Питомцы-грувики» (petsvc): HTTP/Fiber :8094 + gRPC :9094.
# Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY pets/go.mod pets/go.sum pets/
WORKDIR /src/pets
RUN go mod download
COPY pkg/ /src/pkg/
COPY pets/ /src/pets/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/petsvc ./cmd/petsvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D petsvc
COPY --from=build /out/petsvc /usr/local/bin/petsvc
USER petsvc
EXPOSE 8094 9094
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8094/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["petsvc"]
