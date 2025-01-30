FROM golang:1.23-alpine as base
WORKDIR /build
RUN apk --update --no-cache add git gcc musl-dev libpcap-dev
COPY go.mod .
COPY go.sum .
RUN go mod download

FROM base as builder
WORKDIR /build
COPY . .
RUN go build -ldflags '-extldflags=-static' 

FROM alpine:3.18

COPY entrypoint.sh /
COPY --from=builder /build/dtap /usr/bin/dtap

ENTRYPOINT [ "/usr/bin/dtap" ]
