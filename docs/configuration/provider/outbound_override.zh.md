# 出站覆盖

应用于出站提供者中所有出站的覆盖选项。

### 结构

```json
{
  "tag_prefix": "",
  "tag_suffix": "",

  ... // 拨号字段
  ... // TLS 字段
}
```

### 字段

#### tag_prefix

添加到出站提供者中每个出站标签的前缀。

#### tag_suffix

添加到出站提供者中每个出站标签的后缀。

#### 拨号字段

此处设置的字段会覆盖出站提供者中出站的对应字段，详情参阅 [拨号字段](/zh/configuration/shared/dial)。

!!! note ""

    `detour` 仅在出站自身未设置 `detour` 时应用。

#### TLS 字段

此处设置的 TLS 字段会覆盖出站提供者中出站的对应字段，详情参阅 [TLS](/zh/configuration/shared/tls)。

支持的字段：`enabled`、`disable_sni`、`server_name`、`insecure`、`kernel_tx` 和 `kernel_rx`。
