# Микросервис пуш-уведомлений (pushsvc): HTTP/Fiber :8097 — регистрация
# токенов устройств + рассылка FCM по событиям микросервисов.
# Контекст сборки — back-go/ (общий модуль pkg через replace ../pkg).
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY push/go.mod push/go.sum push/
WORKDIR /src/push
RUN go mod download
COPY pkg/ /src/pkg/
COPY push/ /src/push/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/pushsvc ./cmd/pushsvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D pushsvc
COPY --from=build /out/pushsvc /usr/local/bin/pushsvc
USER pushsvc
EXPOSE 8097
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8097/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["pushsvc"]
