### Structure

```json
{
  "type": "urltest",
  "tag": "auto",
  
  "outbounds": [
    "proxy-a",
    "proxy-b",
    "proxy-c"
  ],
  "providers": [
    "provider-a"
  ],
  "exclude": "beta",
  "include": "v2ray",
  "use_all_providers": false,
  "url": "",
  "interval": "",
  "tolerance": 0,
  "idle_timeout": "",
  "interrupt_exist_connections": false
}
```

### Fields

#### outbounds

List of outbound tags to test.

#### providers

List of [provider](/configuration/provider/) tags. The outbounds of these
providers are included in the group.

#### exclude

Filter out provider outbounds whose tag matches this regexp.

#### include

Only keep provider outbounds whose tag matches this regexp.

#### use_all_providers

Use all providers in the configuration.

#### url

The URL to test. `https://www.gstatic.com/generate_204` will be used if empty.

#### interval

The test interval. `3m` will be used if empty.

#### tolerance

The test tolerance in milliseconds. `50` will be used if empty.

#### idle_timeout

The idle timeout. `30m` will be used if empty.

#### interrupt_exist_connections

Interrupt existing connections when the selected outbound has changed.

Only inbound connections are affected by this setting, internal connections will always be interrupted.
