# syntax=docker/dockerfile:1
# Builds the static dashboard. Production hosting is S3 + CloudFront, not this image.
# docker build --build-arg VITE_API_BASE_URL=http://<alb-dns> -o dist .

FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN npm run build

FROM alpine:3.21
RUN adduser -D -H -u 65532 pulse
COPY --from=build --chown=pulse:pulse /app/dist /dist
USER pulse
WORKDIR /dist
CMD ["ls", "-la", "/dist"]
