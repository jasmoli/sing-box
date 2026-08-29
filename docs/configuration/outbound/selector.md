### Structure

```json
{
  "type": "selector",
  "tag": "select",
  
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
  "default": "proxy-c",
  "interrupt_exist_connections": false
}
```

!!! quote ""

    The selector can only be controlled through the [Clash API](/configuration/experimental#clash-api-fields) currently.

### Fields

#### outbounds

List of outbound tags to select.

#### providers

List of [provider](/configuration/provider/) tags. The outbounds of these
providers are included in the group.

#### exclude

Filter out provider outbounds whose tag matches this regexp.

#### include

Only keep provider outbounds whose tag matches this regexp.

#### use_all_providers

Use all providers in the configuration.

#### default

The default outbound tag. The first outbound will be used if empty.

#### interrupt_exist_connections

Interrupt existing connections when the selected outbound has changed.

Only inbound connections are affected by this setting, internal connections will always be interrupted.
