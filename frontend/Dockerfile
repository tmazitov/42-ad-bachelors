# Builds the SPA and serves it with Caddy, which also proxies /api to the
# backend (see ../deploy/Caddyfile, mounted by compose.yaml).

FROM node:22-slim AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM caddy:2.11-alpine
COPY --from=build /app/dist /srv
