# Local 本地提供者

### 结构

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

### 字段

#### path

==必填==

订阅文件的路径。文件会被监听，修改后自动重新加载提供者。

#### health_check

提供者出站的健康检查选项。

| 键         | 格式                           |
|------------|--------------------------------|
| `enabled`  | 是否启用健康检查。为空时为 `false`。启动时始终会进行一次健康检查。 |
| `url`      | 健康检查使用的 URL。默认为 `https://www.gstatic.com/generate_204`。 |
| `interval` | 健康检查的间隔。为空时为 `10m`，最小为 `1m`。 |
| `timeout`  | 单次健康检查请求的超时时间。为空时为 `3s`。 |

### 内联

内联提供者是本地提供者的变体，直接在配置中定义出站。

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
