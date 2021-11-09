FROM golang:1.16-alpine AS builder

WORKDIR /build

ENV GO111MODULE=on
ENV GOPROXY=https://proxy.golang.org

RUN apk --no-cache add \
    libc-dev gcc bash git \
    && rm -rf /var/cache/apk/*

COPY src/* /build/

RUN cd /build && \
   go mod download && \
   go build -o /build/dist/kong-jwt-plugin kong-jwt-plugin.go

FROM kong:2.4-alpine

WORKDIR /app/go-plugins

COPY --from=builder /build/dist/ /app/go-plugins/
