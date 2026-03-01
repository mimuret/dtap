---
mode: 'Agent'
tools: ["changes","codebase"]
description: 'generate configuration.md'
---

設定ファイルのドキュメントを生成します。
全ての出力は、マークダウン形式にしてください。
出力結果を`doc/configuration.md`に保存してください。

## 1. 全体設定

全体設定の内容は以下の通りです。マークダウン形式で出力してください。

- 設定ファイルは、hcl形式で記述されます。
- 設定ファイルのパスは、`-c,--config`オプションで指定します。パスがディレクトリの場合は、そのディレクトリの拡張子が`.hcl`のファイルを全て読み込みます
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

- InputPlugin, OutputPlugin, FilterPluginは、以下のフィールドを指定することができます。
  - `labels`フィールドで、DNSTAP Messageに付与するラベルを指定します。
  - `labels`は出力時のテンプレートや、OutputPluginで利用できます。

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

各Plugin毎の設定の内容をマークダウン形式で出力してください。

出力内容は以下の通りです。
- 1. 見出し
  - PluginのpluginType
- 2. 概要
  - Pluginの処理の概要を３行以内で
- 3. 設定可能なフィールド
  - PluginのBlockに指定できるフィールド名と型,必須パラメータ、デフォルト値
  - hclのtagにremainが含まれている場合はマージしてください。
  - tagがblockの場合
    - 同じgoパッケージの場合は、見出しがhclのtag名で、設定可能なフィールドを作成し、そこへアンカーリンクしてください。
    - 同じパッケージではない場合は、全てのPluginの記述後に、共通blockという章に設定可能なフィールドを作成し、そこへリンクしてください。
- 4. 並列可能
  - OutputPluginの場合は、並列設定が可能かどうかを記載してください。
- 5. 設定例
  - 設定項目が最小限の設定例を記載してください。
  - 設定項目が全て記載された設定例を記載してください。

上記内容を取得するため以下を実行してください
  - `pkg/plugis/**.go`ファイル（`*_test.go`を除く）を検出対象としてください
    - 各PluginのGoファイルを読み込み、`init()`関数内で、`RegisterInputPlugin`, `RegisterOutputPlugin`, `RegisterFilterPlugin`の呼び出しを探してください。
      - RegisterInputPluginが呼び出されている場合、blockTypeは`input`
      - RegisterOutputPluginが呼び出されている場合、blockTypeは`output`
      - RegisterFilterPluginが呼び出されている場合、blockTypeは`filter`
      - 第１引数で指定された文字列が、pluginType、第２引数で指定された関数が、Pluginの設定を行う`Setup関数`になります。
        - `Setup関数`の引数は、InputPluginの場合は`*config.InputBlock`、OutputPluginの場合は`*config.OutputBlock`、FilterPluginの場合は`*config.FilterBlock`になります。
      - Setup関数の中で、gohcl.DecodeBodyを使っている部分を探してください。第３引数が、Pluginの設定を行う構造体です。
        - この構造体のstruct tagに`hcl`が指定されているフィールドが、blockContentになります。
      - Setup関数の戻り値がOutputPluginの場合は、並列で稼働可能か、MaxConcurrentの返り値が、1より大きいかを確認してください。

