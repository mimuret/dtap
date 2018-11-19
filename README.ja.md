# dtap - DNSTAP Message router
dtap はDNSTAP用のメッセージルータです。
様々な入力元からFSTRM形式のDNSTAPメッセージを取得し、
様々な出力先へ出力します。

## Install
```
go get -u github.com/mimuret/dtap/dtap
```

## 入力
### Unix Socket
サーバソフトウェアからの書き込みのため、Unixソケットを作りListenします。
ほとんどのソフトウェアはこの方式です。

Socketを作るPathと、Socketのオーナーを指定することができます。
```
[[InputUnix]]
Path="/var/run/unbound/dnstap.sock"
User="unbound"
```

### TCP Socket
TCPクライアントからの書き込みを受け付けます。
複数のDNSTAPをまとめる用途を想定しています。

Listenするアドレスと、ポートが指定できます。
```
[[InputTCP]]
Address="0.0.0.0"
Port=10053
```

### File
ファイルを1回読み込みます。
gz,bzip2,xzで圧縮されている場合でもそのまま読み込めます。

```
[[InputFile]]
Path="/var/dnscap/tap.fstrm.gz"
```

### Tail
ファイルをTailします。
Glob形式で、複数のファイルのtailにも対応します。

```
[[InputTail]]
Path=/var/dnstap/*.fstrm

```

## Output
### Unix Socket
Inputで受け取ったメッセージをUnixソケットに書き込みします。
Socketがない場合は、１秒おきに再接続を試みます。
```
[[OutputUnix]]
Path="/var/run/unbound/dnstap.sock"
```

### TCP Socket
Inputで受け取ったメッセージをTCPソケットを使い書き込みします。
Socketがない場合は、１秒おきに再接続を試みます。
```
[[OutputTCP]]
Host="otherhost.exmaple.jp"
Port=10053
```

### File
Inputで受け取ったメッセージをファイルに書き込みします。
File名はstrftime形式が使え、自動でローテートできます。
```
[[OutputFile]]
Path = "/var/dnstap/dnstap-%Y%m%d-%H%M.fstrm"
```

### Fluent
Inputで受け取ったメッセージをFlattingし、fluentdに転送します。

```
[[OutputFluent]]
Host = "fluent.example.jp"
Tag  = "dnstap.message"
```