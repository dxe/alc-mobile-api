FROM golang:1.16.2-alpine3.13 AS builder
RUN mkdir /app
ADD . /app
WORKDIR /app

RUN apk update && apk add bash
RUN go get github.com/githubnemo/CompileDaemon

ENTRYPOINT ./wait-for-it.sh db:3306 -- CompileDaemon --build="go build -o alc-mobile-api ." --command=./alc-mobile-api
