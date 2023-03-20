FROM golang:alpine as builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o /reroller .

FROM alpine:latest

RUN apk add tini
COPY --from=builder /reroller /usr/local/bin/reroller

ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/reroller"]
