# Микросервис рассылки писем (mailsvc): gRPC :9098 — Send(шаблон),
# HTTP :8098 — /healthz. Stateless, доставка по SMTP.
# Контекст сборки — back-go/ (общий модуль pkg через replace ../pkg).
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY pkg/go.mod pkg/go.sum pkg/
COPY mail/go.mod mail/go.sum mail/
WORKDIR /src/mail
RUN go mod download
COPY pkg/ /src/pkg/
COPY mail/ /src/mail/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/mailsvc ./cmd/mailsvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D mailsvc
COPY --from=build /out/mailsvc /usr/local/bin/mailsvc
USER mailsvc
EXPOSE 8098 9098
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8098/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["mailsvc"]
