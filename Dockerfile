FROM golang:1.18 as builder

ENV GOPROXY="https://goproxy.cn,direct"

WORKDIR /app
COPY . .

RUN make all

FROM ubuntu:20.04

COPY --from=builder /app/releases/ /usr/bin/