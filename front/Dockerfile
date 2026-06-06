# ---- Сборка фронтенда ----
FROM node:24-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --quiet
COPY . .
RUN npm run build

# ---- Статика через nginx ----
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
# nginx.conf монтируется через volume в docker-compose, не копируется здесь
