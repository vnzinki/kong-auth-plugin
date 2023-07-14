FROM golang:1.17-alpine AS builder

WORKDIR /build

ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org

RUN apk --no-cache add \
    libc-dev gcc bash git \
    && rm -rf /var/cache/apk/*

COPY . /build
WORKDIR /build

RUN go mod download && \
    go build -o ./dist/kong-jwt-plugin ./jwt/kong-jwt-plugin.go &&\
    go build -o ./dist/kong-forward-auth-plugin ./forward-auth/kong-forward-auth-plugin.go



FROM kong:2.7
USER root
WORKDIR /app/go-plugins

COPY --from=builder /build/dist/ /app/go-plugins/
COPY --from=builder /build/utils.lua /usr/local/share/lua/5.1/kong/tools/utils.lua
