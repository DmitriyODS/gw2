# Микросервис ежедневников (diarysvc): HTTP/Fiber :8101 — личные ежедневники
# пользователей (заметки-задачи, привязанные к дню, с временем/архивом и
# шарингом). Контекст сборки — back-go/ (общий модуль pkg подключается через
# replace ../pkg в go.mod). Стадия сборки бежит на платформе билдера
# ($BUILDPLATFORM): при сборке amd64-образа на Apple Silicon компилятор Go
# работает нативно (кросс через GOOS/GOARCH), а не под эмуляцией.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY diary/go.mod diary/go.sum diary/
WORKDIR /src/diary
RUN go mod download
COPY pkg/ /src/pkg/
COPY diary/ /src/diary/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/diarysvc ./cmd/diarysvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D diarysvc
COPY --from=build /out/diarysvc /usr/local/bin/diarysvc
USER diarysvc
EXPOSE 8101
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8101/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["diarysvc"]
