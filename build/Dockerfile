FROM golang:1.17-alpine3.15 AS builder

RUN apk add build-base

COPY . /root/src
WORKDIR /root/src
RUN mkdir /root/binaries && \
    go mod vendor && \
    go build -o /root/binaries/leaderlog-api .

FROM alpine:3.15

RUN adduser -u 1402 lovelace -D

USER lovelace

VOLUME /var/lib/leaderlog-api

LABEL maintainer="Kevin Haller <keivn.haller@blockblu.io>"
LABEL version="${VERSION}"
LABEL description="Lightweight API for managing leaderlog for Web access"

COPY --from=builder /root/binaries/leaderlog-api /usr/local/bin

ENTRYPOINT ["leaderlog-api", "-db-path", "/var/lib/leaderlog-api"]
