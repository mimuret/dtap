FROM golang:1.15-alpine as base
WORKDIR /build
RUN apk --update --no-cache add git gcc musl-dev
copy go.mod .
copy go.sum .
RUN go mod download

FROM base as builder
WORKDIR /build
COPY . .
RUN cd cmd/dtap && go build

FROM alpine:3.10

ENV DTAP_INPUT_UNIX_SOCKET "/dtap/dnstap.sock"
ENV DTAP_INPUT_UNIX_SOCKET_USER "daemon"
ENV DTAP_INPUT_TCP_LISTEN_ADDR ""
ENV DTAP_INPUT_TCP_LISTEN_PORT "10053"
ENV DTAP_OUTPUT_TCP_HOST ""
ENV DTAP_OUTPUT_TCP_PORT "10053"
ENV DTAP_OUTPUT_UNIX_SOCKET ""
ENV DTAP_OUTPUT_FLUENT_HOSTS ""
ENV DTAP_OUTPUT_FLUENT_PORT "24224"
ENV DTAP_OUTPUT_FLUENT_TAG "dnstap"
ENV DTAP_OUTPUT_KAFKA_HOSTS ""
ENV DTAP_OUTPUT_KAFKA_TOPIC "dnstap_message"
ENV DTAP_OUTPUT_NATS_HOST ""
ENV DTAP_OUTPUT_NATS_SUBJECT "dnstap_message"
ENV DTAP_OUTPUT_NATS_USER ""
ENV DTAP_OUTPUT_NATS_PASS ""
ENV DTAP_OUTPUT_NATS_TOKEN ""
ENV DTAP_IPV4_MASK 24
ENV DTAP_IPV6_MASK 48
ENV DTAP_ENABLE_ECS "false"
ENV DTAP_ENABLE_HASH_IP "false"
ENV DTAP_HASH_SALT ""

COPY entrypoint.sh /
COPY --from=builder /build/cmd/dtap/dtap /usr/bin/dtap

RUN mkdir /etc/dtap \
&& chmod 755 /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]
CMD ["/usr/bin/dtap", "-c", "/etc/dtap/dtap.conf"]