### 结构

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

    选择器目前只能通过 [Clash API](/zh/configuration/experimental/clash-api/) 来控制。

### 字段

#### outbounds

用于选择的出站标签列表。

#### providers

[提供者](/zh/configuration/provider/) 的标签列表。这些提供者的出站会加入分组。

#### exclude

过滤掉标签匹配该正则表达式的提供者出站。

#### include

仅保留标签匹配该正则表达式的提供者出站。

#### use_all_providers

使用配置中的所有提供者。

#### default

默认的出站标签。默认使用第一个出站。

#### interrupt_exist_connections

当选定的出站发生更改时，中断现有连接。

仅入站连接受此设置影响，内部连接将始终被中断。