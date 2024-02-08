### 结构

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
    "provider-a",
    "provider-b",
    "provider-c",
  ],
  "use_all_providers": false,
  "includes": [
    "^HK\\..+",
    "^TW\\..+",
    "^SG\\..+",
  ],
  "excludes": "^JP\\..+",
  "types": [
    "shadowsocks",
    "vmess",
    "vless",
  ],
  "ports": [
    "80",
    "2000:4000",
    "2000:",
    ":4000"
  ],
  "url": "",
  "interval": "",
  "tolerance": 50,
  "idle_timeout": "",
  "interrupt_exist_connections": false
}
```

!!! note ""

    当内容只有一项时，可以忽略 JSON 数组 [] 标签。

### 字段

#### outbounds

用于测试的出站标签列表。

#### providers

用于填充 `outbounds` 的提供者标签列表。

#### use_all_providers

使用所有提供者填充 `outbounds`。

#### includes

匹配提供者提供的出站标签正则表达式。

#### excludes

排除提供者提供的出站标签正则表达式。

#### types

匹配提供者提供的出站类型。

#### ports

匹配提供者提供的出站端口。

#### url

用于测试的链接。默认使用 `https://www.gstatic.com/generate_204`。

#### interval

测试间隔。 默认使用 `3m`。

#### tolerance

以毫秒为单位的测试容差。 默认使用 `50`。

#### idle_timeout

空闲超时。默认使用 `30m`。

#### interrupt_exist_connections

当选定的出站发生更改时，中断现有连接。

仅入站连接受此设置影响，内部连接将始终被中断。