FROM golang:1.12 as builder
COPY . /build
RUN cd /build/cmd/dtap && go build

FROM alpine:latest
COPY --from=builder /build/cmd/dtap/dtap /usr/bin/dtap
COPY misc/dtap.toml /config.toml

CMD ["/usr/bin/dtap"]
