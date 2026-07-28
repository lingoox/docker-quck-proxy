# Multi-stage Dockerfile for docker-quck-proxy

ARG GO_VERSION=1.23-alpine
ARG ALPINE_VERSION=3.20

FROM golang:${GO_VERSION} AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /proxy .

FROM alpine:${ALPINE_VERSION}
RUN apk --no-cache add ca-certificates && \
    rm -rf /var/cache/apk/*
COPY --from=builder /proxy /usr/local/bin/proxy
EXPOSE 5000
ENTRYPOINT ["proxy"]
