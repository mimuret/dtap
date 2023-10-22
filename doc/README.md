# Configuration


## Configuration file

- [see godoc](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/config#Config)


## filter_plugin

- [iphash](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/filter/iphash#IPHash)
- [label](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/filter/label#Label)
- [mask](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/filter/mask#Mask)
- [matcher](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/filter/matcher#MatcherConfig)
- [metrics](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/filter/metrics#Metrics)

### for debug

- [nop](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/filter/nop#Nop)
- [static](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/filter/static#Static)

## input plugin

- [file](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/input/file#File)
- [nats](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/input/nats#Nats)
- [tcp](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/input/tcp#TCPSocket)
- [unix](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/input/unix#UnixSocket)
- [pcap](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/input/pcap#PCAP)

## output plugins

- [file](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/file#Output)
- [nats](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/nats#Nats)
- [stdout](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/stdout#Stdout)
- [tcp](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/tcp#TCP)
- [unix](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/unix#Unix)

### debug and experimental plugins

- [dns](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/dns#DNS)
- [fluent](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/fluent#Fluent)
- [kafka](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/kafka#Kafka)
- [loki](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/loki#Loki)
- [nop](https://pkg.go.dev/github.com/mimuret/dtap/v2/pkg/plugin/output/nop#NOP)
