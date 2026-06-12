# Миграции схемы (goose): run-once контейнер, бежит ДО старта сервисов
# (compose: depends_on condition service_completed_successfully) — наследник
# `flask db upgrade` прежнего app-контейнера. Контекст сборки — back-go/.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY migrate/go.mod migrate/go.sum migrate/
WORKDIR /src/migrate
RUN go mod download
COPY migrate/ /src/migrate/
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && adduser -D migrate
COPY --from=build /out/migrate /usr/local/bin/migrate
USER migrate
ENTRYPOINT ["migrate"]
