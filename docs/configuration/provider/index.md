# Outbound Provider

### Structure

```json
{
  "providers": [
    {
      "type": "",
      "tag": "",
      ... // Provider Fields
    }
  ]
}
```

### Fields

| Type     | Format           |
|----------|------------------|
| `remote` | [Remote](./remote) |
| `local`  | [Local](./local)   |
| `inline` | [Local](./local)   |

#### tag

The tag of the provider.

Provider outbounds are named after the following rules:

- A node without a tag is named `[provider_tag]0`, `[provider_tag]1`, ...
- A tagged node keeps its tag, unless it collides with a main outbound or a
  node from an earlier provider, in which case `[1]`, `[2]`, ... suffixes are
  appended until the tag is unique. Main outbounds win over providers, earlier
  providers win over later ones, and earlier nodes win over later ones.

### Subscription Format

The provider supports the following subscription formats, which are detected
automatically:

| Format                           | Description                     |
|----------------------------------|---------------------------------|
| sing-box JSON                    | Configuration with `outbounds`  |
| Clash                            | Configuration with `proxies`    |
| `ss://`                          | Shadowsocks                     |
| `vmess://`                       | VMess                           |
| `vless://`                       | VLESS                           |
| `trojan://`                      | Trojan                          |
| `tuic://`                        | TUIC                            |
| `hysteria://`                    | Hysteria                        |
| `hysteria2://`, `hy2://`         | Hysteria2                       |
| `anytls://`                      | AnyTLS                          |
| `snell://`                       | Snell                           |
| `naive+https://`, `naive+http://` | NaiveProxy                     |
| `wireguard://`                   | WireGuard                       |
