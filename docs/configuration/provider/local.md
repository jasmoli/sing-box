# Local

### Structure

```json
{
  "type": "local",
  "tag": "local",
  "path": "./subscription.txt",

  "health_check": {
    "enabled": true,
    "url": "https://www.gstatic.com/generate_204",
    "interval": "10m",
    "timeout": "3s"
  }
}
```

### Fields

#### path

==Required==

The path of the subscription file. The file is watched for changes and the
provider is reloaded on modification.

#### health_check

Health check options for the provider outbounds.

| Key        | Format                          |
|------------|---------------------------------|
| `enabled`  | Enable health check. `false` if empty. Health check always runs once at start. |
| `url`      | The URL used for health check. Default is `https://www.gstatic.com/generate_204`. |
| `interval` | The interval for health check. `10m` if empty, minimum `1m`. |
| `timeout`  | The timeout for a single health check request. `3s` if empty. |

### Inline

The inline provider is a variant of the local provider with the outbounds
defined directly in the configuration.

```json
{
  "type": "inline",
  "tag": "inline",

  "outbounds": [
    {
      "type": "socks",
      "tag": "node1",
      "server": "127.0.0.1",
      "server_port": 1080
    }
  ],

  "health_check": {}
}
```
