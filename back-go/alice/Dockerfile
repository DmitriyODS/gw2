# Микросервис навыка Алисы (alicesvc): HTTP/Fiber :8104 — публичный вебхук
# Яндекс.Диалогов (/api/alice/webhook). Голосовые команды исполняются gRPC-
# вызовами tasksvc/diarysvc/notesvc; ИИ-разбор фраз — через aisvc. Контекст
# сборки — back-go/ (общий модуль pkg подключается через replace ../pkg в
# go.mod). Стадия сборки бежит на платформе билдера ($BUILDPLATFORM).
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY alice/go.mod alice/go.sum alice/
WORKDIR /src/alice
RUN go mod download
COPY pkg/ /src/pkg/
COPY alice/ /src/alice/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/alicesvc ./cmd/alicesvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D alicesvc
COPY --from=build /out/alicesvc /usr/local/bin/alicesvc
USER alicesvc
EXPOSE 8104
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8104/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["alicesvc"]
