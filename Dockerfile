FROM golang:1.27-alpine as base
WORKDIR /build
RUN apk --update --no-cache add git gcc musl-dev
COPY go.mod .
COPY go.sum .
RUN go mod download

FROM base as builder
WORKDIR /build
COPY . .

ENV CGO_ENABLED=1

RUN go build \
    -buildmode=pie \
    -ldflags "-s -w -extldflags '-static-pie'" \
    -o dtap .

FROM scratch

COPY --from=builder /build/dtap /usr/bin/dtap

ENTRYPOINT [ "/usr/bin/dtap" ]
