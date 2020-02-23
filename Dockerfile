
FROM golang:1.13.8-alpine as builder1

ENV GO111MODULE=on
WORKDIR /app/whshare
COPY go.mod .
COPY go.sum .

RUN go mod download
COPY ./*.go .

RUN go build -o main

FROM alpine:latest
COPY --from=builder1 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=builder1 /app/whshare/main .

RUN mkdir /app/data/
ENV S_LOG_LEVEL debug
CMD [ "/app/main","server","start"]