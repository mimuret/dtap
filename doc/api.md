# api

## metrics

DTAP provides metrics.

```
GET: /metrics
```


## Reload

DTAP can reload the confog file at runtime.
After the config file is successfully loaded,
all currently running plugin processes and API servers are terminated.
Then, start Plugin and the API server based on the reloaded configuration.
If the Plugin fails to start, the program terminates.

```
POST: /reload
```

