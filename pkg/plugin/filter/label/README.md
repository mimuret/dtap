## Label filter plguin

The label filter plugin adds labels to DNSTAP extra field.
It can get label value by [dnsutils getter functions](https://pkg.go.dev/github.com/mimuret/dnsutils/getter).


### Parameters

|Name|type|Required|Default|Description|
|:-:|:-:|:-:|:-:|:-:|
|Add|[]AddLabel|no||Add settings|
|Del|[]DelLabel|no||Del settings|

AddLabel

|Name|type|Required|Default|Description|
|:-:|:-:|:-:|:-:|:-:|
|Name|string|yes||Label name|
|Type|enum("DNSTAP","DNS","STATIC")|yes|||
|Value|string|yes||Type: DNSTAP, Value Type: getter.DnstapGetterName<br />Type: DNS, Value Type: getter.DnsMsgGetterName<br />Type: STATIC, Value Type: add this value|

DelLabel

|Name|type|Required|Default|Description|
|:-:|:-:|:-:|:-:|:-:|
|Name|string|yes||Label name|
