FROM golang:1.16.2-alpine3.13 AS builder
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o alc-mobile-api

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/alc-mobile-api /alc-mobile-api
ENTRYPOINT ["./alc-mobile-api"]