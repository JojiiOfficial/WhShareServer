FROM golang:1.13.6-alpine as builder

WORKDIR /app/WhShareServer

COPY ./*.go ./

RUN apk add --no-cache git
RUN go get -d -v 
RUN CGO_ENABLED=0
RUN go build -o main
RUN pwd && ls -lah

FROM alpine:latest

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
WORKDIR /app

COPY --from=builder /app/WhShareServer/main .

RUN mkdir /app/data/
RUN ls -lath

ENV BRIDGE_DATA_PATH="/app/data/"

CMD [ "/app/main","server","start"]