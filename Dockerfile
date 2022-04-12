FROM golang:latest

WORKDIR /L0

COPY ./ /L0

RUN go mod download

