# Stage 1: Build
FROM node:20-alpine AS builder

ARG BUILD_ENV_FILE

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

COPY ${BUILD_ENV_FILE:-.env.build} .env

RUN npm run build

# Stage 2: Serve
FROM nginx:1.27-alpine

COPY --from=builder /app/dist /usr/share/nginx/html

# Replace default nginx config
COPY nginx.conf /etc/nginx/conf.d/default.conf