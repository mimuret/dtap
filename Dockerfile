FROM golang:1.19-alpine as base
WORKDIR /build
RUN apk --update --no-cache add git gcc musl-dev
COPY go.mod .
COPY go.sum .
RUN go mod download

FROM base as builder
WORKDIR /build
COPY . .
RUN go build

FROM alpine:3.16

COPY entrypoint.sh /
COPY --from=builder /build/dtap /usr/bin/dtap

ENTRYPOINT [ "/usr/bin/dtap" ]
