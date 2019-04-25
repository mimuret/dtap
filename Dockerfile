FROM golang:1.12 as builder
COPY . /build
RUN cd /build && go build

FROM alpine:latest
COPY --from=builer /build/dtap /usr/bin/dtap
COPY misc/dtap.toml /config.toml

CMD ["/usr/bin/dtap"]
