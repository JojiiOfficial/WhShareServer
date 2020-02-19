# Stage 1 download dependencies
FROM golang:1.13.6-alpine as builder1

WORKDIR /app/WhShareServer

COPY ./go.* ./

RUN apk add --no-cache git
RUN go mod download

# Stage 2 build binary
FROM golang:1.13.6-alpine as builder2
WORKDIR /app/WhShareServer

COPY --from=builder1 /app/WhShareServer/ .
COPY ./*.go ./
COPY --from=builder1 /go /go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main

# Stage 3 create final image
FROM alpine:latest
COPY --from=builder1 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

WORKDIR /app
COPY --from=builder2 /app/WhShareServer/main .

RUN mkdir /app/data/

CMD [ "/app/main","server","start"]
