# Микросервис авторизации (authsvc): HTTP/Fiber :8091.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/authsvc ./cmd/authsvc

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D authsvc
COPY --from=build /out/authsvc /usr/local/bin/authsvc
USER authsvc
EXPOSE 8091
# busybox wget — для healthcheck в docker-compose. Именно 127.0.0.1:
# localhost в alpine резолвится сначала в ::1, busybox wget не делает
# фолбэк на IPv4, а fiber слушает только tcp4 → ложный unhealthy.
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:8091/healthz >/dev/null 2>&1 || exit 1
ENTRYPOINT ["authsvc"]
