
FROM golang:1.13-alpine as builder1

# Setting up environment for builder1
ENV GO111MODULE=on
WORKDIR /app/whshare

# install required package(s)
RUN apk --no-cache add ca-certificates git

# Copy dependency list
COPY go.mod .
COPY go.sum .
RUN go mod download

# Copy files
COPY ./*.go ./
COPY ./models/*.go ./models/
COPY ./constants/*.go ./constants/
COPY ./services/*.go ./services/
COPY ./handlers/*.go ./handlers/
COPY ./storage/*.go ./storage/

# Compile
RUN go build -o main

# Create new stage based on alpine
FROM alpine:latest

#Copy ca certs
COPY --from=builder1 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy compiled binary from builder1
WORKDIR /app
RUN mkdir /app/data/

COPY --from=builder1 /app/whshare/main .

# Set Debuglevel and start the server
ENV S_LOG_LEVEL debug
CMD [ "/app/main","server","start"]
