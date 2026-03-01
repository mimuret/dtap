# design


## plugins

DTAPは３つのpluginから構成されます

### input plugin 

Input pluginは、DNSTAP Message を生成することが役割です。
受信したメッセージをforward_toで設定したpluginのキューに送信します。
forward_toには、filter plguinとoutput pluginが指定できます。
また、複数のpluginを指定することも可能です。

### filter plugin

filter pluginは、DNSTAP Messageの中身を操作すのが主な役割です。
Labelの付与や、メッセージのDROP、IPアドレスのMaskなどを行うことができます。
処理後は、forward_toで設定したpluginのキューに送信します。

### output plugin

output pluginは、DNSTAP Messageを出力するのが主な役割です。
出力方式としては、unix socketやtcp経由でDNSTAPを送信したり、
JSON形式での出力、prometheusのmetricsなど様々な形式をサポートしています。

