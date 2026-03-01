# 設定ファイルドキュメント

## 1. 全体設定

- 設定ファイルは、hcl形式で記述されます。
- 設定ファイルのパスは、`-c,--config`オプションで指定します。パスがディレクトリの場合は、そのディレクトリの拡張子が`hcl`のファイルを全て読み込みます

- Pluginの設定フォーマットは以下の通りになります。
```
"{{ blockType }}" "{{ pluginType }}" "{{ instanceName}}" {
  {{ blockContent }}
}
```

- blockType は、Pluginの種類に応じて以下のように変わります。
  - InputPluginの場合: `input`
  - OutputPluginの場合: `output`
  - FilterPluginの場合: `filter`
- pluginTypeは、blockType毎に設定するpluginを指定するための名前になります。
- instanceNameは、ユーザが設定ファイルで指定するPluginの名前になります。
  - instanceNameは、同じ、blockTypeとpluginTypeの設定で、uniqueである必要があります。
  - 別のblockTypeやpluginTypeで同じinstanceNameを指定することは可能です。
- 設定したpluginを識別する名前をFullNameと呼びます。**FullName**は`blockType.pluginType.instanceName`を使います。
- blockContentは、Pluginの設定に応じて異なります。

- 全てのPluginは、blockContentとして、以下のフィールドを指定することができます。
  - `labels`フィールドで、DNSTAP Messageに付与するラベルを指定します。

- InputPlugin,FilterPluginは、次の処理を行うPluginの指定が必要です。
  - `forward_to`フィールドで指定します。
  - `forward_to`フィールドは、次の処理を行うOutputPlugin、FilterPluginのFullNameを指定します。
  - `forward_to`フィールドは、複数のPluginを指定することができます。
  - `forward_to`フィールドは、必須パラメータで、１つ以上のPluginを指定する必要があります。

- OutputPlugin, FilterPluginは、DNSTAP Messageを受け取るキューのサイズを指定することができます。
  - `buffer_size`フィールドで指定します。
  - `buffer_size`フィールドのデフォルト値は、`10000`です。

- OutputPluginは、並列で処理を行う数を指定できます。ただし、Pluginによっては、並列処理をサポートしていない場合があります。
  - `concurrency`フィールドで指定します。

- 設定例

```
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

## 2. 各Plugin別の設定項目

### Input Plugin

#### file

##### 概要

ファイルからDNSTAPメッセージを一度だけ読み込みます。
指定されたファイルを読み込み、DNSTAPまたはDtapFrame形式のメッセージを処理します。

##### 設定可能なフィールド

- `path` (string) - 必須: ファイルパス
- `format` (string) - オプション: メッセージフォーマット。DNSTAP、[DtapFrame](#DTapFrame)がサポートされています。デフォルトは"DNSTAP"。

##### 設定例

```hcl
input "file" "example" {
  path = "/var/log/dnstap.log"
  forward_to = ["output.stdout.example"]
}

input "file" "example2" {
  path = "/var/log/dnstap.log"
  format = "DNSTAP"
  forward_to = ["output.stdout.example"]
  labels {
    "source" = "file"
  }
}
```

#### nats

##### 概要
NATSサーバーからメッセージを取得します。
指定されたサブジェクトとキューを購読し、FSTRM形式のメッセージを受信します。
TokenまたはUser/Passwordによる認証をサポートしています。

##### 設定可能なフィールド
- `hosts` ([]string) - 必須: NATSサーバーのURL
- `subject` (string) - 必須: NATSサブジェクト
- `format` (string) - オプション: メッセージフォーマット。DNSTAP、[DtapFrame](#DTapFrame)がサポートされています。デフォルトは`DtapFrame`。
- `queue_name` (string) - オプション: NATSキュー名
- `queue_len` (int) - オプション: NATSサブスクライバーキューの長さ。デフォルトは64
- `user` (string) - オプション: NATSユーザー（tokenが指定されていない場合）
- `password` (string) - オプション: NATSパスワード（tokenが指定されていない場合）
- `token` (string) - オプション: NATSトークン
- `secure` (bool) - オプション: セキュア接続(TLS)を使用するかどうか
- `tls_config` (block) - オプション: [TLS設定](#tlsclinetconfig)

##### 設定例

```hcl
input "nats" "example" {
  hosts = ["nats://localhost:4222"]
  subject = "example.subject"
  forward_to = ["output.stdout.example"]
}

input "nats" "example2" {
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

##### 概要
指定されたネットワークインターフェースからpcapライブラリを使用してネットワークパケットをキャプチャします。
DNSメッセージを処理し、指定されたフォワーダーに転送します。

##### 設定可能なフィールド
- `device` (string) - 必須: パケットをキャプチャするネットワークインターフェース
- `bpf` (string) - オプション: パケットをフィルタリングするBerkeley Packet Filter式。デフォルトは"port 53"
- `direction` (string) - オプション: キャプチャするパケットの方向。"in"、"out"、"inout"。デフォルトは"inout"
- `resolver_query_enabled` (bool) - オプション: リゾルバークエリの処理を有効にするかどうか。デフォルトはfalse
- `resolver_response_enabled` (bool) - オプション: リゾルバーレスポンスの処理を有効にするかどうか。デフォルトはfalse
- `client_query_enabled` (bool) - オプション: クライアントクエリの処理を有効にするかどうか。デフォルトはfalse
- `client_response_enabled` (bool) - オプション: クライアントレスポンスの処理を有効にするかどうか。デフォルトはfalse
- `worker_num` (int64) - オプション: 並列処理のワーカー数。デフォルトは1

##### 設定例

```hcl
input "pcap" "example" {
  device = "eth0"
  forward_to = ["output.stdout.example"]
}

```hcl
input "pcap" "example2" {
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

##### 概要

TCPソケットからメッセージを取得します。
指定されたアドレスとポートでTCPサーバーを起動し、FSTRM形式のメッセージを受信します。
TLS設定をサポートしています。

##### 設定可能なフィールド

- `address` (string) - 必須: バインドするアドレス
- `port` (uint16) - 必須: バインドするポート番号
- `format` (string) - オプション: メッセージフォーマット。DNSTAP、[DtapFrame](#DTapFrame)がサポートされています。デフォルトは`DNSTAP`。
- `tls_config` (block) - オプション: [TLS設定](#tls_config)

##### 設定例

```hcl
input "tcp" "example" {
  address = "127.0.0.1"
  port = 12345
  forward_to = ["output.stdout.example"]
}

input "tcp" "example2" {
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

##### 概要

UNIXソケットからDNSTAPメッセージを取得します。
指定されたパスでUNIXソケットサーバーを起動し、接続されたクライアントからメッセージを受信します。
ソケットファイルの所有者を指定することができます。

##### 設定可能なフィールド
- `path` (string) - 必須: UNIXソケットファイルパス
- `format` (string) - オプション: メッセージフォーマット。DNSTAP、[DtapFrame](#DTapFrame)がサポートされています。デフォルトは`DNSTAP`。
- `user` (string) - オプション: UNIXソケットファイルの所有者

##### 設定例

```hcl
input "unix" "example" {
  path = "/var/run/dnstap.sock"
  forward_to = ["output.stdout.example"]
}

input "unix" "example2" {
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

##### 概要

DNSMessageを許可する ilter処理を行います。
filterは[expr](https://github.com/expr-lang/expr)のフォーマットで記述できます。使用できる変数はConvertV1MapStringで出力される

##### 設定可能なフィールド

- `expression` (string) - 必須: filter式
- `error_log_enabled` (bool) - オプション: trueの場合フィルタ式評価失敗時にエラーログを出力する。filter式のデバック用に使用してください。

#### iphash

##### 概要

クエリアドレスとレスポンスアドレスからQueryAddressHashとResponseAddressHashラベルを作成します。
指定されたソルトを使用してSHA256ハッシュを生成し、IPアドレスを匿名化します。

##### 設定可能なフィールド

- `salt` (string) - 必須: ハッシュ作成用のソルト
- `query_address_enabled` (bool) - オプション: QueryAddressHashラベルを作成するかどうか。デフォルトはtrue
- `response_address_enabled` (bool) - オプション: ResponseAddressHashラベルを作成するかどうか。デフォルトはtrue

##### 設定例

```hcl
filter "iphash" "example" {
  salt = "my-secret-salt"
  forward_to = ["output.stdout.example"]
}

filter "iphash" "example2" {
  salt = "my-secret-salt"
  query_address_enabled = true
  response_address_enabled = false
  forward_to = ["output.stdout.example"]
  labels = {
    "source" = "iphash"
  }
}
```

#### nop

##### 概要

何も操作を行わないフィルタープラグインです。
デバッグ用途で使用され、メッセージをそのまま通過させます。

##### 設定可能なフィールド

##### 設定例

```hcl
filter "nop" "example" {
  forward_to = ["output.stdout.example"]
}
```

#### static

##### 概要
DNSTAPメッセージに静的なラベルを追加します。
設定で指定されたラベルがメッセージに付与されます。

##### 設定可能なフィールド

- `deny` (bool) - オプション: trueの場合、メッセージを全てdropする。デフォルトはfalse

##### 設定例

```hcl
filter "static" "example" {
  forward_to = ["output.stdout.example"]
}

filter "static" "example2" {
  deny = false
  forward_to = ["output.stdout.example"]
  labels = {
    "filter" = "static"
  }
}
```

### Output Plugins

#### dns

##### 概要

DNSTAP メッセージからDNSクエリを抽出し、指定されたDNSサーバーに送信します。

##### 設定可能なフィールド

- `host` (string) - 必須: DNSサーバーのホスト名またはIPアドレス
- `port` (uint16) - 必須: DNSサーバーのポート番号
- `protocol` (string) - オプション: 送信時のTransportプロトコル、udp,tcp,tcp-tls,https,https-get,https-postが指定できます。デフォルトはudp
- `timeout` (duration) - オプション: クエリのタイムアウト時間
- `use_response_question` (bool) - Optional: レスポンスメッセージ内のQuestion Sectionを利用する。デフォルトはfalse

##### 並列実行

並列設定が可能です。

##### 設定例

```hcl
output "dns" "example" {
  host = "127.0.0.1"
  port = 53
}

```hcl
output "dns" "example2" {
  host = "127.0.0.1"
  port = 53
  protocol = "tcp-tls"
  timeout = "5s"
  concurrency = 4
}
```

#### file

##### 概要
DNSTAPメッセージをファイルに出力します。
JSON、プロトコルバッファ、カスタムテンプレート形式での出力をサポートします。

##### 設定可能なフィールド
- `path` (string) - 必須: 出力ファイルパス、strftimeフォーマットで指定可能
- `format` (string) - オプション: 出力フォーマット。"json/v1"、"protobuf"、"go-template"。デフォルトは"json/v1"
- `output_filters` (block) - オプション: [JSON キーフィルター](#output_filters)
- `template` (string) - オプション: format が "go-template" の場合のテンプレート文字列、変数はtypes.MsgValue

##### 並列可能

並列設定はサポートしていません

##### 設定例

```hcl
output "file" "example" {
  path = "/var/log/dtap%Y%M%D.json"
}

output "file" "example2" {
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

##### 概要

Apache KafkaにDNSTAPメッセージを送信します。
JSON、Protocol Buffers、Avro形式での出力をサポートし、Schema Registryとの連携も可能です。

##### 設定可能なフィールド

- `hosts` ([]string) - 必須: Kafkaブローカーのアドレスリスト
- `topic` (string) - 必須: メッセージを送信するKafkaトピック
- `key` (string) - オプション: メッセージのキー
- `output_type` (string) - オプション: 出力タイプ。"json"、"protobuf"、"avero"。デフォルトは"json"
- `schema_registries` ([]string) - オプション: Schema Registryのアドレスリスト（avero使用時）
- `retry` (uint) - オプション: メッセージ送信のリトライ回数
- `output_filters` (block) - オプション: [JSON キーフィルター](#output_filters)
- `max_retry` (uint) - オプション: サーバー接続の最大リトライ回数

##### 並列可能

並列設定が可能です。

##### 設定例

```hcl
output "kafka" "example" {
  hosts = ["localhost:9092"]
  topic = "dnstap"
}
```

全設定項目:
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

##### 概要

GrafanaのLoki APIにログメッセージとして出力します。
ラベルを使用してログストリームを管理し、効率的なログ検索を可能にします。

##### 設定可能なフィールド

- `url` (string) - 必須: LokiサーバーのURL
- `max_stream` (int) - オプション: Lokiに送信する最大ストリーム数
- `max_line_size` (int) - オプション: Lokiに送信する行の最大サイズ。デフォルトは4096
- `max_line_size_truncate` (bool) - オプション: 行サイズが最大値を超えた場合に切り詰めるかどうか
- `output_filters` (block) - オプション: [JSON キーフィルター](#output_filters)
- `message_labels` ([]string) - オプション: DNSTAPの値からLokiラベルを作成するフィールド名のリスト
- `const_labels` (map[string]string) - オプション: 全ログエントリに追加される定数ラベル
- `max_retry` (uint) - オプション: Lokiサーバー接続の最大リトライ回数

##### 並列可能
並列設定が可能です。

##### 設定例

```hcl
output "loki" "example" {
  url = "http://localhost:3100/loki/api/v1/push"
}

output "loki" "example2" {
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

##### 概要

prometehusのmetricsを作成し、DNSTAP Messageをカウントします
plugin設定の名前が、prometheusのmetrics名として利用されます。

##### 設定可能なフィールド

- `message_labels` ([]string) - オプション: prometehusのラベルに使用する、key。types.ConvertV1MapStringで出力されるmapのkeyとDNSTAP Messageのラベルのキー名を指定できる。
- `description` (string) - オプション: HELPで出力されるメッセージ

##### 並列可能
並列設定できません。

##### 設定例

-  全てのメッセージをカウントする場合

```hcl
output "metrics" "dtap_message_total" {}
```

- rcodeをカウントする場合

```hcl
output "metrics" "dtap_message_total_by_rcode" {
  message_labels = ["rcode"]
}
```

#### nats

##### 概要

NATSサーバーにDNSTAPメッセージを送信します。
DTAPFrameまたはDNSTAP形式でのメッセージ送信をサポートし、
メッセージサイズと送信間隔に基づいてバッチ処理を行います。

##### 設定可能なフィールド
- `hosts` ([]string) - 必須: NATSサーバーのURLリスト
- `subject` (string) - 必須: メッセージを送信するNATSサブジェクト
- `user` (string) - オプション: NATSユーザー（tokenが指定されていない場合）
- `password` (string) - オプション: NATSパスワード（tokenが指定されていない場合）
- `token` (string) - オプション: NATSトークン
- `secure` (bool) - オプション: セキュア接続(TLS)を使用するかどうか
- `format` (string) - オプション: メッセージフォーマット。"DtapFrame"、"DNSTAP"。デフォルトは"DtapFrame"
- `max_size` (int) - オプション: 最大ペイロードサイズ。デフォルトは1MB
- `interval` (duration) - オプション: 送信間隔。デフォルトは1秒
- `tls_config` (block) - オプション: [TLS設定](#tls_config)
- `max_retry` (uint) - オプション: サーバー接続の最大リトライ回数

##### 並列可能
並列設定が可能です。

##### 設定例

```hcl
output "nats" "example" {
  hosts = ["nats://localhost:4222"]
  subject = "dnstap"
}

output "nats" "exampl2" {
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

##### 概要
何も操作を行わない出力プラグインです。
受信したメッセージを単純に破棄します。テスト用途やパフォーマンス測定に使用されます。

##### 設定可能なフィールド
設定可能なフィールドはありません。

##### 並列可能
並列設定が可能です。

##### 設定例

```hcl
output "nop" "example" {}
```

#### otel-log

##### 概要
OpenTelemetryのログエクスポーターを使用してログを送信します。
OTLP、OTLP HTTP、標準出力への出力をサポートしています。

##### 設定可能なフィールド
- `output_filters` (block) - オプション: [JSON キーフィルター](#output_filters)
- `attribute_names` ([]string) - オプション: ログレコードに含める属性名のリスト
- `logger_name` (string) - オプション: ロガー名。デフォルトは"dtap"
- `resource_attributes` (map[string]string) - オプション: リソース属性
- `otlp` (block) - オプション: [OTLP設定](#otlp)
- `otlp_http` (block) - オプション: [OTLP HTTP設定](#otlp_http)
- `max_retry` (uint) - オプション: OTLP接続の最大リトライ回数

##### 並列可能
並列設定が可能です。

##### 設定例

```hcl
output "otel-log" "example" {}

output "otel-log" "example2" {
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

##### 概要
標準出力にDNSTAPメッセージを出力します。
JSON、プロトコルバッファ、カスタムテンプレート形式での出力をサポートします。

##### 設定可能なフィールド
- `format` (string) - オプション: 出力フォーマット。"json"、"protobuf"、"go-template"。デフォルトは"json"
- `output_filters` (block) - オプション: [JSON キーフィルター](#output_filters)
- `template` (string) - オプション: format が "go-template" の場合のテンプレート文字列

##### 並列可能
並列設定が可能です。

##### 設定例

最小限の設定:
```hcl
output "stdout" "example" {}

output "stdout" "example2" {
  format = "json"
  output_filters {
    include_keys = ["timestamp", "query_name"]
    exclude_keys = ["raw_message"]
  }
  template = "{{.timestamp}} {{.query_name}}"
  concurrency = 1
}
```

#### tcp

##### 概要
TCPサーバーにDNSTAPメッセージを送信します。
指定されたホストとポートに接続してメッセージを転送し、TLS接続もサポートしています。

##### 設定可能なフィールド
- `host` (string) - 必須: TCPサーバーのホスト名
- `port` (uint16) - 必須: TCPサーバーのポート番号
- `max_retry` (uint) - オプション: UNIXソケットを開く最大リトライ回数
- `tls_config` (block) - オプション: [TLS設定](#tls_config)

##### 並列可能
並列設定が可能です。

##### 設定例

```hcl
output "tcp" "example" {
  host = "127.0.0.1"
  port = 12345
}

output "tcp" "example2" {
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

##### 概要
UNIXソケットにDNSTAPメッセージを出力します。
指定されたパスのUNIXソケットに接続してメッセージを送信します。

##### 設定可能なフィールド
- `path` (string) - 必須: UNIXソケットパス
- `max_retry` (uint) - オプション: UNIXソケットを開く最大リトライ回数

##### 並列可能
並列設定が可能です。

##### 設定例

```hcl
output "unix" "example" {
  path = "/var/run/dtap.sock"
}

output "unix" "example" {
  path = "/var/run/dtap.sock"
  max_retry = 3
  concurrency = 2
}
```

## 3. 共通Block

### TLSClinetConfig

TLS接続の設定を行うために使用されます。

#### 設定可能なフィールド
- `certificate` (string) - オプション: 証明書ファイルのパス
- `private_key` (string) - オプション: 秘密鍵ファイルのパス
- `ca_certificate` (string) - オプション: CA証明書ファイルのパス
- `insecure_skip_verify` (bool) - オプション: サーバー証明書の検証をスキップするかどうか

### TLSServerConfig

TLSサーバの設定を行うために使用されます。

#### 設定可能なフィールド

- `inter_certificates` ([]string) - オプション: 中間証明書ファイルのパス
- `certificate` (string) - 証明書ファイルのパス
- `private_key` (string) - 秘密鍵ファイルのパス
- `client_ca_certificate` (string) - オプション: mTLS用の証明書検証用のCA証明書

### output_filters

JSON出力時のキーフィルタリング設定を行います。

#### 設定可能なフィールド
- `include_keys` ([]string) - オプション: 出力に含めるキーのリスト
- `exclude_keys` ([]string) - オプション: 出力から除外するキーのリスト

### otlp

OTLP（OpenTelemetry Protocol）のgRPC接続設定です。

#### 設定可能なフィールド
- `endpoint` (string) - 必須: OTLPエンドポイント（host:port形式）
- `insecure` (bool) - オプション: 非セキュア接続を使用するかどうか
- `headers` (map[string]string) - オプション: リクエストに含めるヘッダー

### otlp_http

OTLP（OpenTelemetry Protocol）のHTTP接続設定です。

#### 設定可能なフィールド
- `endpoint` (string) - 必須: OTLP HTTPエンドポイントのURL
- `insecure` (bool) - オプション: 非セキュア接続を使用するかどうか
- `headers` (map[string]string) - オプション: リクエストに含めるヘッダー
