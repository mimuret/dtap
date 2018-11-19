# DNSTAP Filter
```
# example
[[Filter]]
name = "Client-Query"
key = "Message.Type"
op = "eq"
value = "CQ"
```


## Dnstap
| key | type |
----|---- 
| Dnstap.identity | string |
| Dnstap.version  | string |
| Dnstap.extra  | string |

## Dnstap.Message
| key | type |
----|---- 
| Dnstap.Type | string SQ,CQ,RQ,AQ,AR,RR,CR,SR,FQ,FR |
| Dnstap.SocketFamily  | enum  |
| Dnstap.extra  | string |
