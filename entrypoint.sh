#!/bin/ash

if [ ! -e "/etc/dtap/dtap.conf" ] ; then
  if [ "${DTAP_INPUT_UNIX_SOCKET}" != "" ] ; then
    cat <<- EOS >> /etc/dtap/dtap.conf
[[InputUnix]]
  Path="${DTAP_INPUT_UNIX_SOCKET}"
  User="${DTAP_INPUT_UNIX_SOCKET_USER}"
EOS
  fi
  if [ "${DTAP_INPUT_TCP_LISTEN_ADDR}" != "" ] ; then
    cat <<- EOS >> /etc/dtap/dtap.conf
[[InputTCP]]
  Address="${DTAP_INPUT_TCP_LISTEN_ADDR}"
  Port=${DTAP_INPUT_TCP_LISTEN_PORT}
EOS
  fi
  if [ "${DTAP_OUTPUT_TCP_HOST}" != "" ] ; then
    cat <<- EOS >> /etc/dtap/dtap.conf
[[OutputTCP]]
  Host="${DTAP_OUTPUT_TCP_HOST}"
  Port=${DTAP_OUTPUT_TCP_PORT}
EOS
  fi
  if [ "$DTAP_OUTPUT_UNIX_SOCKET" != "" ] ; then
    cat <<- EOS >> /etc/dtap/dtap.conf
[[OutputUnix]]
  Path="${DTAP_OUTPUT_UNIX_SOCKET}"
EOS
  fi
  if [ "${DTAP_OUTPUT_FLUENT_HOST}" != "" ] ; then
    cat <<- EOS >> /etc/dtap/dtap.conf
[[OutputFluent]]
  Host = "${DTAP_OUTPUT_FLUENT_HOST}"
  Port = ${DTAP_OUTPUT_FLUENT_PORT}
  Tag = "${DTAP_OUTPUT_FLUENT_TAG}"
  [OutputFluent.flat]
    IPv4Mask = ${DTAP_IPV4_MASK}
    IPv6Mask = ${DTAP_IPV6_MASK}
    EnableECS = ${DTAP_ENABLE_ECS}
    EnableHashIP = ${DTAP_ENABLE_HASH_IP}
    IPHashSaltPath = "${DTAP_HASH_SALT}"
EOS

  fi
  if [ "${DTAP_OUTPUT_KAFKA_HOSTS}" != "" ] ; then
    host=$(echo ${DTAP_OUTPUT_KAFKA_HOSTS} | sed 's/,/","/g')
    cat <<- EOS >> /etc/dtap/dtap.conf
[[OutputKafka]]
  Hosts = ["${host}"]
  ${registries}
  Topic  = "${DTAP_OUTPUT_KAFKA_TOPIC}"
EOS
    if [ "${DTAP_OUTPUT_SCHEMA_REGISTRIES}" != "" ] ; then
      r=$(echo ${DTAP_OUTPUT_SCHEMA_REGISTRIES} | sed 's/,/","/g')
      cat <<- EOS >> /etc/dtap/dtap.conf
  SchemaRegistries = ["${r}"]
EOS
    fi
  fi
  if [ "${DTAP_OUTPUT_NATS_HOST}" != "" ] ; then
    cat <<- EOS >> /etc/dtap/dtap.conf
[[OutputNats]]
  Hosts = "${DTAP_OUTPUT_NATS_HOST}"
  User = "${DTAP_OUTPUT_NATS_USER}"
  Password = "${DTAP_OUTPUT_NATS_PASS}"
  Token = "${DTAP_OUTPUT_NATS_TOKEN}"
  Subject  = "${DTAP_OUTPUT_NATS_SUBJECT}"
  [OutputNats.flat]
    IPv4Mask = ${DTAP_IPV4_MASK}
    IPv6Mask = ${DTAP_IPV6_MASK}
    EnableECS = ${DTAP_ENABLE_ECS}
    EnableHashIP = ${DTAP_ENABLE_HASH_IP}
    IPHashSaltPath = "${DTAP_HASH_SALT}"
EOS
  fi
  if [ "${DTAP_OUTPUT_STDOUT_TYPE}" != "" ] ; then
    cat <<- EOS >> /etc/dtap/dtap.conf
[[OutputStdout]]
  Type = "${DTAP_OUTPUT_STDOUT_TYPE}"
  Template = "${DTAP_OUTPUT_STDOUT_TEMPLATE}"
  [OutputStdout.flat]
    IPv4Mask = ${DTAP_IPV4_MASK}
    IPv6Mask = ${DTAP_IPV6_MASK}
    EnableECS = ${DTAP_ENABLE_ECS}
    EnableHashIP = ${DTAP_ENABLE_HASH_IP}
    IPHashSaltPath = "${DTAP_HASH_SALT}"
EOS
  fi
fi

cat /etc/dtap/dtap.conf

exec "$@"