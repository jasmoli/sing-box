# 出站提供者

### 结构

```json
{
  "providers": [
    {
      "type": "",
      "tag": "",
      ... // Provider 字段
    }
  ]
}
```

### 字段

| 类型     | 格式            |
|----------|-----------------|
| `remote` | [远程](./remote) |
| `local`  | [本地](./local)   |
| `inline` | [本地](./local)   |

#### tag

出站提供者的标签。

出站提供者的出站按以下规则命名：

- 无标签的节点自动命名为 `[provider_tag]0`、`[provider_tag]1`……
- 有标签的节点直接使用其标签，若与主出站或更早的出站提供者的节点冲突，则追加 `[1]`、`[2]`…… 后缀直到唯一。去重优先级：主出站 > 配置靠前的出站提供者 > 出站提供者内靠前的节点。

### 订阅格式

出站提供者支持以下订阅格式，会自动检测：

| 格式                           | 说明                          |
|----------------------------------|---------------------------------|
| sing-box JSON                    | 带 `outbounds` 的配置           |
| Clash                            | 带 `proxies` 的配置             |
| `ss://`                          | Shadowsocks                     |
| `vmess://`                       | VMess                           |
| `vless://`                       | VLESS                           |
| `trojan://`                      | Trojan                          |
| `tuic://`                        | TUIC                            |
| `hysteria://`                    | Hysteria                        |
| `hysteria2://`、`hy2://`         | Hysteria2                       |
| `anytls://`                      | AnyTLS                          |
| `snell://`                       | Snell                           |
| `naive+https://`、`naive+http://` | NaiveProxy                     |
| `wireguard://`                   | WireGuard                       |
