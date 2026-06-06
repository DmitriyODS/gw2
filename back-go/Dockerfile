# Многостейдж Dockerfile для любого сервиса Groove Work v4.0.
# Имя сервиса передаётся через build-arg SERVICE.
# Пример: docker build --build-arg SERVICE=auth -f back-go/Dockerfile back-go

# ───────── builder ─────────
FROM golang:1.25-alpine3.21 AS builder

ARG SERVICE
RUN test -n "$SERVICE" || (echo "build-arg SERVICE is required" && exit 1)

WORKDIR /src

# Кэшируем модульный слой
COPY go.work go.work
COPY shared/pkg/go.mod shared/pkg/go.mod
COPY gen/proto/go.mod gen/proto/go.mod
COPY services/auth/go.mod services/auth/go.mod
COPY services/users/go.mod services/users/go.mod
COPY services/tasks/go.mod services/tasks/go.mod
COPY services/messenger/go.mod services/messenger/go.mod
COPY services/calls/go.mod services/calls/go.mod
COPY services/ai/go.mod services/ai/go.mod
COPY shared/pkg/go.sum shared/pkg/go.sum
COPY gen/proto/go.sum gen/proto/go.sum
COPY services/auth/go.sum services/auth/go.sum
COPY services/users/go.sum services/users/go.sum
COPY services/tasks/go.sum services/tasks/go.sum
COPY services/messenger/go.sum services/messenger/go.sum
COPY services/calls/go.sum services/calls/go.sum
COPY services/ai/go.sum services/ai/go.sum
RUN cd services/${SERVICE} && go mod download

# Копируем исходники
COPY . .

# Сборка статического бинаря
RUN cd services/${SERVICE} && \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -ldflags="-s -w" \
    -o /out/service ./cmd/${SERVICE}

# ───────── runtime ─────────
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/service /service

EXPOSE 8080 9090

USER nonroot:nonroot
ENTRYPOINT ["/service"]
