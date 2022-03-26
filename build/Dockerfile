FROM golang:1.18-bullseye AS builder

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        build-essential \
    && rm -rf /var/lib/apt/lists/*
    

COPY . /root/src
WORKDIR /root/src
RUN mkdir /root/binaries \
    && go mod vendor \
    && go build -o /root/binaries/leaderlog-api .

FROM debian:bullseye

RUN adduser -u 1402 lovelace --disabled-password --gecos ""

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        netbase \
    && rm -rf /var/lib/apt/lists/*

USER lovelace

VOLUME /var/lib/leaderlog-api

LABEL maintainer="Kevin Haller <keivn.haller@blockblu.io>"
LABEL version="${VERSION}"
LABEL description="Lightweight API for managing leaderlog for Web access"

COPY --from=builder /root/binaries/leaderlog-api /usr/local/bin

ENTRYPOINT ["leaderlog-api", "-db-path", "/var/lib/leaderlog-api"]
