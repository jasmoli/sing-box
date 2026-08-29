# Provider 提供者

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

提供者的标签。

提供者的出站按以下规则命名：

- 无标签的节点自动命名为 `[provider_tag]0`、`[provider_tag]1`……
- 有标签的节点直接使用其标签，若与主出站或更早 provider 的节点冲突，则追加 `[1]`、`[2]`…… 后缀直到唯一。去重优先级：主出站 > 配置靠前的 provider > provider 内靠前的节点。

### 订阅格式

提供者支持以下订阅格式，会自动检测：

- 带 `outbounds` 的 sing-box JSON 配置
- 带 `proxies` 的 Clash 配置
- 原生 URI 链接（`ss://`、`ssr://`、`vmess://`、`vless://`、`trojan://`、`tuic://`、`hysteria://`、`hy2://`）
