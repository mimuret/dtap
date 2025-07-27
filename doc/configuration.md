# Configuration File Documentation

## 1. Global Configuration

- Configuration files are written in HCL format.
- The configuration file path is specified with the `-c,--config` option. If the path is a directory, all files with the `.hcl` extension in that directory will be loaded.
- The Plugin configuration format is as follows:
```
"{{ blockType }}" "{{ pluginType }}" "{{ instanceName}}" {
  {{ blockContent }}
}
```
- blockType changes according to the Plugin type as follows:
  - For InputPlugin: `input`
  - For OutputPlugin: `output`
  - For FilterPlugin: `filter`
- pluginType is the name used to specify the plugin to be configured for each blockType.
- instanceName is the Plugin name specified by the user in the configuration file.
  - instanceName must be unique for the same blockType and pluginType configuration.
  - It is possible to specify the same instanceName for different blockType or pluginType.
- The name that identifies the configured plugin is called FullName. **FullName** uses `blockType.pluginType.instanceName`.
- blockContent varies depending on the Plugin configuration.

- InputPlugin, OutputPlugin, FilterPlugin can specify the following fields:
  - The `labels` field specifies labels to be attached to DNSTAP Messages.
  - `labels` can be used in output templates and OutputPlugins.

- InputPlugin and FilterPlugin require specification of the Plugin to perform the next processing.
  - Specified with the `forward_to` field.
  - The `forward_to` field specifies the FullName of OutputPlugin or FilterPlugin to perform the next processing.
  - The `forward_to` field can specify multiple Plugins.
  - The `forward_to` field is a required parameter and must specify one or more Plugins.

- OutputPlugin and FilterPlugin can specify the queue size for receiving DNSTAP Messages.
  - Specified with the `buffer_size` field.
  - The default value of the `buffer_size` field is `10000`.

- OutputPlugin can specify the number of parallel processes. However, some Plugins may not support parallel processing.
  - Specified with the `concurrency` field.

- Configuration Example

```hcl
input "unix" "example" {
  path = "/var/run/dtap.sock"
  forward_to = ["output.stdout.example","filter.sample.example"]
  labels {
    "from" = "unbound"
  }
}
filter "sample" "example" {
  sampling_rate = 0.1
  forward_to = ["output.tcp.example"]
}
output "stdout" "example" {}
output "tcp" "example" {
  host = "localhost"
  port = 12345
  concurrency = 2
}
```

## 2. Plugin-specific Configuration Items

### Input Plugins

#### file

##### Overview
Reads DNSTAP messages from a file only once.
Reads the specified file and processes messages in DNSTAP or DtapFrame format.

##### Configurable Fields
- `path` (string) - Required: File path
- `format` (string) - Optional: Message format. Default is "DNSTAP". Supports "DNSTAP" and "DtapFrame"

##### Configuration Examples

Minimal configuration:
```hcl
input "file" "example" {
  path = "/var/log/dnstap.log"
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
input "file" "example" {
  path = "/var/log/dnstap.log"
  format = "DNSTAP"
  forward_to = ["output.stdout.example"]
  labels {
    "source" = "file"
  }
}
```

#### nats

##### Overview
Retrieves messages from NATS server.
Subscribes to specified subject and queue, receiving messages in FSTRM format.
Supports authentication via Token or User/Password.

##### Configurable Fields
- `hosts` ([]string) - Required: List of NATS server URLs
- `subject` (string) - Required: NATS subject
- `format` (string) - Optional: Payload format. Default is "DtapFrame". Supports "DNSTAP" and "DtapFrame"
- `queue_name` (string) - Optional: NATS queue name
- `queue_len` (int) - Optional: NATS subscriber queue length. Default is 64
- `user` (string) - Optional: NATS user (when token is not specified)
- `password` (string) - Optional: NATS password (when token is not specified)
- `token` (string) - Optional: NATS token
- `secure` (bool) - Optional: Whether to use secure connection (TLS)
- `tls_config` (block) - Optional: [TLS configuration](#tls_config)

##### Configuration Examples

Minimal configuration:
```hcl
input "nats" "example" {
  hosts = ["nats://localhost:4222"]
  subject = "example.subject"
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
input "nats" "example" {
  hosts = ["nats://localhost:4222"]
  subject = "example.subject"
  queue_name = "example_queue"
  queue_len = 64
  user = "username"
  password = "password"
  format = "DtapFrame"
  secure = true
  tls_config {
    certificate = "/path/to/cert.pem"
    private_key = "/path/to/key.pem"
  }
  forward_to = ["output.stdout.example"]
  labels {
    "source" = "nats"
  }
}
```

#### pcap

##### Overview
Captures network packets from a specified network interface using the pcap library.
Processes DNS messages and forwards them to specified forwarders.

##### Configurable Fields
- `device` (string) - Required: Network interface to capture packets from
- `bpf` (string) - Optional: Berkeley Packet Filter expression to filter packets. Default is "port 53"
- `direction` (string) - Optional: Packet direction to capture. "in", "out", or "inout". Default is "inout"
- `resolver_query_enabled` (bool) - Optional: Whether to enable processing of resolver queries. Default is true
- `resolver_response_enabled` (bool) - Optional: Whether to enable processing of resolver responses. Default is true
- `client_query_enabled` (bool) - Optional: Whether to enable processing of client queries. Default is true
- `client_response_enabled` (bool) - Optional: Whether to enable processing of client responses. Default is true
- `worker_num` (int64) - Optional: Number of workers for parallel processing. Default is 1

##### Configuration Examples

Minimal configuration:
```hcl
input "pcap" "example" {
  device = "eth0"
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
input "pcap" "example" {
  device = "eth0"
  bpf = "port 53"
  direction = "inout"
  resolver_query_enabled = true
  resolver_response_enabled = true
  client_query_enabled = true
  client_response_enabled = true
  worker_num = 4
  forward_to = ["output.stdout.example"]
  labels {
    "source" = "pcap"
  }
}
```

#### tcp

##### Overview
Retrieves messages from TCP socket.
Starts a TCP server on the specified address and port, receiving messages in FSTRM format.
Supports TLS configuration.

##### Configurable Fields
- `address` (string) - Required: Address to bind to
- `port` (uint16) - Required: Port number to bind to
- `format` (string) - Optional: Message format. Default is "DNSTAP"
- `tls_config` (block) - Optional: [TLS configuration](#tls_config)

##### Configuration Examples

Minimal configuration:
```hcl
input "tcp" "example" {
  address = "127.0.0.1"
  port = 12345
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
input "tcp" "example" {
  address = "127.0.0.1"
  port = 12345
  format = "DNSTAP"
  tls_config {
    certificate = "/path/to/cert.pem"
    private_key = "/path/to/key.pem"
  }
  forward_to = ["output.stdout.example"]
  labels {
    "source" = "tcp"
  }
}
```

#### unix

##### Overview
Retrieves DNSTAP messages from UNIX socket.
Starts a UNIX socket server on the specified path and receives messages from connected clients.
Can specify the owner of the socket file.

##### Configurable Fields
- `path` (string) - Required: UNIX socket file path
- `format` (string) - Optional: Message format. Default is "DNSTAP"
- `user` (string) - Optional: Owner of the UNIX socket file

##### Configuration Examples

Minimal configuration:
```hcl
input "unix" "example" {
  path = "/var/run/dnstap.sock"
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
input "unix" "example" {
  path = "/var/run/dnstap.sock"
  format = "DNSTAP"
  user = "dnstap"
  forward_to = ["output.stdout.example"]
  labels {
    "source" = "unix"
  }
}
```

### Filter Plugins

#### expr

##### Overview
Filters DNSTAP messages using go-expr expressions.
Evaluates the specified expression against message fields and forwards messages that evaluate to true.

##### Configurable Fields
- `expression` (string) - Required: Expression to evaluate
- `error_log_enabled` (bool) - Optional: Whether to log errors during expression evaluation

##### Configuration Examples

Minimal configuration:
```hcl
filter "expr" "example" {
  expression = "qtype == 'A'"
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
filter "expr" "example" {
  expression = "qtype == 'A' && qname =~ '.*\\.example\\.com'"
  error_log_enabled = true
  forward_to = ["output.stdout.example"]
}
```

#### iphash

##### Overview
Creates QueryAddressHash and ResponseAddressHash labels from query and response addresses.
Generates SHA256 hashes using the specified salt to anonymize IP addresses.

##### Configurable Fields
- `salt` (string) - Required: Salt for hash creation
- `query_address_enabled` (bool) - Optional: Whether to create QueryAddressHash label. Default is true
- `response_address_enabled` (bool) - Optional: Whether to create ResponseAddressHash label. Default is true

##### Configuration Examples

Minimal configuration:
```hcl
filter "iphash" "example" {
  salt = "my-secret-salt"
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
filter "iphash" "example" {
  salt = "my-secret-salt"
  query_address_enabled = true
  response_address_enabled = false
  forward_to = ["output.stdout.example"]
}
```

#### mask

##### Overview
Masks IP addresses in DNSTAP messages for privacy protection.
Supports IPv4 and IPv6 address masking with configurable subnet masks.

##### Configurable Fields
- `query_address_enabled` (bool) - Optional: Whether to mask query addresses. Default is true
- `response_address_enabled` (bool) - Optional: Whether to mask response addresses. Default is true
- `ipv4_mask` (int) - Optional: IPv4 subnet mask bits. Default is 24
- `ipv6_mask` (int) - Optional: IPv6 subnet mask bits. Default is 48

##### Configuration Examples

Minimal configuration:
```hcl
filter "mask" "example" {
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
filter "mask" "example" {
  query_address_enabled = true
  response_address_enabled = true
  ipv4_mask = 24
  ipv6_mask = 48
  forward_to = ["output.stdout.example"]
}
```

#### nop

##### Overview
A filter plugin that performs no operations.
Used for debugging purposes and passes messages through unchanged.

##### Configurable Fields
No configurable fields.

##### Configuration Examples

Minimal configuration:
```hcl
filter "nop" "example" {
  forward_to = ["output.stdout.example"]
}
```

#### relabel

##### Overview
Adds or modifies labels on DNSTAP messages.
Allows dynamic label creation based on message content and static label assignment.

##### Configurable Fields
- `rules` ([]block) - Optional: List of relabeling rules
  - `source_labels` ([]string) - Source label names
  - `target_label` (string) - Target label name
  - `separator` (string) - Optional: Separator for joining source labels
  - `regex` (string) - Optional: Regex pattern for matching
  - `replacement` (string) - Optional: Replacement string
  - `action` (string) - Action to perform: "replace", "keep", "drop", "labelmap", "labeldrop", "labelkeep"

##### Configuration Examples

Minimal configuration:
```hcl
filter "relabel" "example" {
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
filter "relabel" "example" {
  rules {
    source_labels = ["query_name"]
    target_label = "domain"
    regex = "(.+)\\.example\\.com"
    replacement = "${1}"
    action = "replace"
  }
  forward_to = ["output.stdout.example"]
}
```

#### sample

##### Overview
Samples DNSTAP messages based on a specified sampling rate.
Randomly selects messages to forward based on the configured probability.

##### Configurable Fields
- `sampling_rate` (float64) - Required: Sampling rate (0.0 to 1.0)

##### Configuration Examples

Minimal configuration:
```hcl
filter "sample" "example" {
  sampling_rate = 0.1
  forward_to = ["output.stdout.example"]
}
```

#### static

##### Overview
Adds static labels to DNSTAP messages.
Labels specified in the configuration are added to messages.

##### Configurable Fields
- Any key name (string) - Key and value of labels to add

##### Configuration Examples

Minimal configuration:
```hcl
filter "static" "example" {
  forward_to = ["output.stdout.example"]
}
```

Full configuration:
```hcl
filter "static" "example" {
  environment = "production"
  datacenter = "tokyo"
  forward_to = ["output.stdout.example"]
}
```

### Output Plugins

#### dns

##### Overview
Sends DNS queries to actual DNS servers and retrieves responses.
Extracts DNS queries from DNSTAP messages and forwards them to specified servers.

##### Configurable Fields
- `host` (string) - Required: DNS server hostname or IP address
- `port` (uint16) - Required: DNS server port number
- `timeout` (duration) - Optional: Query timeout duration

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "dns" "example" {
  host = "8.8.8.8"
  port = 53
}
```

Full configuration:
```hcl
output "dns" "example" {
  host = "8.8.8.8"
  port = 53
  timeout = "5s"
  concurrency = 4
}
```

#### file

##### Overview
Outputs DNSTAP messages to files.
Supports output in JSON, protocol buffer, and custom template formats.

##### Configurable Fields
- `path` (string) - Required: Output file path
- `format` (string) - Optional: Output format. "json", "protobuf", "go-template". Default is "json"
- `output_filters` (block) - Optional: [JSON key filters](#output_filters)
- `template` (string) - Optional: Template string when format is "go-template"

##### Parallel Processing
No, parallel processing is not supported (MaxConcurrent = 1).

##### Configuration Examples

Minimal configuration:
```hcl
output "file" "example" {
  path = "/var/log/dtap.json"
}
```

Full configuration:
```hcl
output "file" "example" {
  path = "/var/log/dtap.json"
  format = "json"
  output_filters {
    include_keys = ["timestamp", "query_name"]
    exclude_keys = ["raw_message"]
  }
  template = "{{.timestamp}} {{.query_name}}"
}
```

#### kafka

##### Overview
Sends DNSTAP messages to Apache Kafka.
Supports output in JSON, Protocol Buffers, and Avro formats,
and can integrate with Schema Registry.

##### Configurable Fields
- `hosts` ([]string) - Required: List of Kafka broker addresses
- `topic` (string) - Required: Kafka topic to send messages to
- `key` (string) - Optional: Message key
- `output_type` (string) - Optional: Output type. "json", "protobuf", "avro". Default is "json"
- `schema_registries` ([]string) - Optional: List of Schema Registry addresses (when using avro)
- `retry` (uint) - Optional: Number of retries for message sending
- `output_filters` (block) - Optional: [JSON key filters](#output_filters)
- `max_retry` (uint) - Optional: Maximum retry count for server connection

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "kafka" "example" {
  hosts = ["localhost:9092"]
  topic = "dnstap"
}
```

Full configuration:
```hcl
output "kafka" "example" {
  hosts = ["localhost:9092"]
  topic = "dnstap"
  key = "dnstap-key"
  output_type = "json"
  schema_registries = ["http://localhost:8081"]
  retry = 3
  max_retry = 5
  output_filters {
    include_keys = ["timestamp", "query_name"]
  }
  concurrency = 4
}
```

#### loki

##### Overview
Outputs to Grafana's Loki as log messages.
Manages log streams using labels and enables efficient log searching.

##### Configurable Fields
- `url` (string) - Required: Loki server URL
- `max_stream` (int) - Optional: Maximum number of streams to send to Loki
- `max_line_size` (int) - Optional: Maximum line size to send to Loki. Default is 4096
- `max_line_size_truncate` (bool) - Optional: Whether to truncate when line size exceeds maximum
- `output_filters` (block) - Optional: [JSON key filters](#output_filters)
- `message_labels` ([]string) - Optional: List of field names to create Loki labels from DNSTAP values
- `const_labels` (map[string]string) - Optional: Constant labels added to all log entries
- `max_retry` (uint) - Optional: Maximum retry count for Loki server connection

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "loki" "example" {
  url = "http://localhost:3100/loki/api/v1/push"
}
```

Full configuration:
```hcl
output "loki" "example" {
  url = "http://localhost:3100/loki/api/v1/push"
  max_stream = 1000
  max_line_size = 4096
  max_line_size_truncate = true
  output_filters {
    include_keys = ["timestamp", "query_name"]
  }
  message_labels = ["query_name", "response_code"]
  const_labels = {
    level = "info"
    environment = "production"
  }
  max_retry = 3
  concurrency = 2
}
```

#### metrics

##### Overview
Exposes DNSTAP message metrics in Prometheus format.
Provides various metrics about DNS traffic for monitoring and observability.

##### Configurable Fields
- `address` (string) - Required: Address to bind the metrics HTTP server
- `port` (uint16) - Required: Port number for the metrics HTTP server
- `path` (string) - Optional: HTTP path for metrics endpoint. Default is "/metrics"

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "metrics" "example" {
  address = "0.0.0.0"
  port = 9090
}
```

Full configuration:
```hcl
output "metrics" "example" {
  address = "0.0.0.0"
  port = 9090
  path = "/metrics"
  concurrency = 1
}
```

#### nats

##### Overview
Sends DNSTAP messages to NATS server.
Supports message sending in DTAPFrame or DNSTAP format,
and performs batch processing based on message size and transmission interval.

##### Configurable Fields
- `hosts` ([]string) - Required: List of NATS server URLs
- `subject` (string) - Required: NATS subject to send messages to
- `user` (string) - Optional: NATS user (when token is not specified)
- `password` (string) - Optional: NATS password (when token is not specified)
- `token` (string) - Optional: NATS token
- `secure` (bool) - Optional: Whether to use secure connection (TLS)
- `format` (string) - Optional: Message format. "DtapFrame", "DNSTAP". Default is "DtapFrame"
- `max_size` (int) - Optional: Maximum payload size. Default is 1MB
- `interval` (duration) - Optional: Transmission interval. Default is 1 second
- `tls_config` (block) - Optional: [TLS configuration](#tls_config)
- `max_retry` (uint) - Optional: Maximum retry count for server connection

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "nats" "example" {
  hosts = ["nats://localhost:4222"]
  subject = "dnstap"
}
```

Full configuration:
```hcl
output "nats" "example" {
  hosts = ["nats://localhost:4222"]
  subject = "dnstap"
  user = "username"
  password = "password"
  secure = true
  format = "DtapFrame"
  max_size = 1048576
  interval = "1s"
  tls_config {
    certificate = "/path/to/cert.pem"
    private_key = "/path/to/key.pem"
  }
  max_retry = 3
  concurrency = 4
}
```

#### nop

##### Overview
An output plugin that performs no operations.
Simply discards received messages. Used for testing and performance measurement.

##### Configurable Fields
No configurable fields.

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "nop" "example" {}
```

#### otel-log

##### Overview
Sends logs using OpenTelemetry log exporter.
Supports output to OTLP, OTLP HTTP, and standard output.

##### Configurable Fields
- `output_filters` (block) - Optional: [JSON key filters](#output_filters)
- `attribute_names` ([]string) - Optional: List of attribute names to include in log records
- `logger_name` (string) - Optional: Logger name. Default is "dtap"
- `resource_attributes` (map[string]string) - Optional: Resource attributes
- `otlp` (block) - Optional: [OTLP configuration](#otlp)
- `otlp_http` (block) - Optional: [OTLP HTTP configuration](#otlp_http)
- `max_retry` (uint) - Optional: Maximum retry count for OTLP connection

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "otel-log" "example" {}
```

Full configuration:
```hcl
output "otel-log" "example" {
  output_filters {
    include_keys = ["timestamp", "query_name"]
  }
  attribute_names = ["query_name", "response_code"]
  logger_name = "dtap"
  resource_attributes = {
    service_name = "dtap"
    environment = "production"
  }
  otlp {
    endpoint = "localhost:4317"
    insecure = true
    headers = {
      "Authorization" = "Bearer token"
    }
  }
  max_retry = 3
  concurrency = 2
}
```

#### stdout

##### Overview
Outputs DNSTAP messages to standard output.
Supports output in JSON, protocol buffer, and custom template formats.

##### Configurable Fields
- `stderr` (bool) - Optional: If true, output to stderr instead of stdout
- `format` (string) - Optional: Output format. "json/v1", "go-template". Default is "json/v1"
- `output_filters` (block) - Optional: [JSON key filters](#output_filters)
- `template` (string) - Optional: Template string when format is "go-template"

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "stdout" "example" {}
```

Full configuration:
```hcl
output "stdout" "example" {
  stderr = false
  format = "json/v1"
  output_filters {
    include_keys = ["timestamp", "query_name"]
    exclude_keys = ["raw_message"]
  }
  template = "{{.timestamp}} {{.query_name}}"
  concurrency = 1
}
```

#### tcp

##### Overview
Sends DNSTAP messages to TCP server.
Connects to the specified host and port to forward messages, also supports TLS connections.

##### Configurable Fields
- `host` (string) - Required: TCP server hostname
- `port` (uint16) - Required: TCP server port number
- `max_retry` (uint) - Optional: Maximum retry count to open TCP socket
- `tls_config` (block) - Optional: [TLS configuration](#tls_config)

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "tcp" "example" {
  host = "127.0.0.1"
  port = 12345
}
```

Full configuration:
```hcl
output "tcp" "example" {
  host = "127.0.0.1"
  port = 12345
  max_retry = 3
  tls_config {
    certificate = "/path/to/cert.pem"
    private_key = "/path/to/key.pem"
  }
  concurrency = 2
}
```

#### unix

##### Overview
Outputs DNSTAP messages to UNIX socket.
Connects to the UNIX socket at the specified path and sends messages.

##### Configurable Fields
- `path` (string) - Required: UNIX socket path
- `max_retry` (uint) - Optional: Maximum retry count to open UNIX socket

##### Parallel Processing
Yes, parallel processing is supported.

##### Configuration Examples

Minimal configuration:
```hcl
output "unix" "example" {
  path = "/var/run/dtap.sock"
}
```

Full configuration:
```hcl
output "unix" "example" {
  path = "/var/run/dtap.sock"
  max_retry = 3
  concurrency = 2
}
```

## 3. Common Blocks

### tls_config

Used to configure TLS connections.

#### Configurable Fields
- `certificate` (string) - Optional: Certificate file path
- `private_key` (string) - Optional: Private key file path
- `ca_certificate` (string) - Optional: CA certificate file path
- `insecure_skip_verify` (bool) - Optional: Whether to skip server certificate verification

### output_filters

Configures key filtering for JSON output.

#### Configurable Fields
- `include_keys` ([]string) - Optional: List of keys to include in output
- `exclude_keys` ([]string) - Optional: List of keys to exclude from output

### otlp

OTLP (OpenTelemetry Protocol) gRPC connection configuration.

#### Configurable Fields
- `endpoint` (string) - Required: OTLP endpoint (host:port format)
- `insecure` (bool) - Optional: Whether to use insecure connection
- `headers` (map[string]string) - Optional: Headers to include in requests

### otlp_http

OTLP (OpenTelemetry Protocol) HTTP connection configuration.

#### Configurable Fields
- `endpoint` (string) - Required: OTLP HTTP endpoint URL
- `insecure` (bool) - Optional: Whether to use insecure connection
- `headers` (map[string]string) - Optional: Headers to include in requests
