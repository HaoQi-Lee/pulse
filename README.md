# pulse

基于 [logrus](https://github.com/sirupsen/logrus) 的结构化日志库，通过 TCP 将 JSON 格式的日志发送到 Logstash 端点。

## 特性

- 结构化 JSON 日志输出，兼容 ELK 技术栈
- TCP 传输，连接断开时自动重连
- 自动附加元数据（主机名、PID、时间戳、调用位置）
- 异步写入，不阻塞业务逻辑
- 进程退出前自动 flush 缓冲区（信号处理 + 显式关闭）

## 安装

```bash
go get git.showcai.com.cn/tech/pulse
```

## 快速开始

```go
package main

import (
	"fmt"
	"git.showcai.com.cn/tech/pulse"
)

func main() {
	defer pulse.Setup("my-project", "logstash.example.com:5959")()

	pulse.Info("服务启动")
	pulse.Warn("磁盘空间不足")
	pulse.Debug("调试信息")
	pulse.Error(fmt.Errorf("连接超时"))
}
```

`Setup` 返回一个关闭函数，`defer` 确保进程正常退出前 flush 缓冲区中的剩余日志。同时内部监听 `SIGINT`/`SIGTERM` 信号，被 kill 时也会自动 flush。

## API

### `Setup(name string, logstash string) func()`

初始化日志器。`name` 为项目名称，`logstash` 为 Logstash TCP 地址。返回关闭函数，应在 `main` 中 `defer` 调用以确保退出前 flush。

### `Info(message string)` / `Debug(message string)` / `Warn(message string)`

输出对应级别的日志。

### `Error(err error)`

输出错误级别的日志，参数为 `error` 类型。

## 日志格式

每条日志以 JSON 格式通过 TCP 发送，结构如下：

```json
{
  "thread_name": "12345",
  "host": "web-server-01",
  "@timestamp": "2026-04-28T10:30:00.123Z",
  "logger_name": "pkg/handler.go",
  "@metadata": { "beat": "logback" },
  "fields": { "project": "my-project", "service": "golang" },
  "message": "服务启动"
}
```

## 退出时 flush

`Setup` 提供两层退出保障：

1. **显式关闭**：`defer pulse.Setup(...)()` — 进程正常 return 时 flush 缓冲区
2. **信号处理**：内部监听 `SIGINT`/`SIGTERM`，收到终止信号时自动 flush 并退出

## 自动重连

`TcpWriter` 在写入失败时自动尝试重新建立 TCP 连接。重连期间日志暂存于 1024 容量的缓冲区，缓冲区满时丢弃新日志并输出警告到 stderr，避免阻塞业务。

## 依赖

- Go 1.14+
- [github.com/sirupsen/logrus](https://github.com/sirupsen/logrus)
