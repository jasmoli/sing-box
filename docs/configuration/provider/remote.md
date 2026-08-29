# Remote

### Structure

```json
{
  "type": "remote",
  "tag": "remote",
  "url": "https://example.com/subscription",
  "path": "./subscription.json",
  "user_agent": "sing-box",
  "http_client": "my-client",
  "update_interval": "24h",
  "download_detour": "direct",

  "exclude": "beta",
  "include": "v2ray",

  "health_check": {}
}
```

### Fields

#### url

==Required==

The URL of the subscription.

#### path

The path of the file the subscription is saved to. When set, the downloaded
subscription is persisted to this file and restored from it on start. The
cache file is used instead when empty.

#### http_client

The tag of the [HTTP client](/configuration/shared/http-client/) used to
download the subscription.

#### download_detour

The tag of the outbound used to download the subscription. Note that it will
not work when `http_client` is set.

> Note: This option is not available in sing-box 1.16.0 and later, use
> `http_client` instead.

#### user_agent

The `User-Agent` used when downloading the subscription.

Default is `sing-box <version>`.

#### update_interval

The interval of downloading the subscription. `24h` if empty, minimum `1m`.

#### download_detour

The tag of the outbound used to download the subscription.

The default outbound will be used if empty.

#### exclude

Filter outbounds whose tag matches this regexp.

#### include

Only keep outbounds whose tag matches this regexp.

#### health_check

Same as the local provider.
