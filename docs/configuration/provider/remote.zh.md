# Remote 远程提供者

### 结构

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

### 字段

#### url

==必填==

订阅的 URL。

#### path

订阅保存到的文件路径。设置后，下载的订阅会保存到该文件，并在启动时从该文件恢复。为空时使用缓存文件。

#### http_client

用于下载订阅的 [HTTP 客户端](/zh/configuration/shared/http-client/) 标签。

#### download_detour

用于下载订阅的出站标签。设置 `http_client` 时此选项不会生效。

> 注意：该选项在 sing-box 1.16.0 及以后版本中不可用，请改用 `http_client`。

#### user_agent

下载订阅时使用的 `User-Agent`。

默认为 `sing-box <版本>`。

#### update_interval

下载订阅的间隔。为空时为 `24h`，最小为 `1m`。

#### download_detour

用于下载订阅的出站标签。

为空时使用默认出站。

#### exclude

过滤掉标签匹配该正则表达式的出站。

#### include

仅保留标签匹配该正则表达式的出站。

#### health_check

与本地提供者相同。
