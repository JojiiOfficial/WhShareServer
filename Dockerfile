FROM golang:1.13.6-alpine as builder

WORKDIR /app/WhShareServer

COPY ./*.go ./

RUN apk add --no-cache git
RUN go get -d -v 
RUN CGO_ENABLED=0 go build -o main

FROM alpine:latest

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app

COPY --from=builder /app/WhShareServer/main .

RUN mkdir /app/data/

CMD [ "/app/main","server","start"]
