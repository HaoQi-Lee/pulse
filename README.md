# pulse

日志库的 **TCP 传输增强层**。Core 统一定义日志条目结构和 TCP 传输，各插件作为适配器桥接原生日志库事件到 Core。

## 架构

```
pulse (core)                  # 无第三方日志库依赖
├── plugins/slog              # slog.Handler 适配器（标准库）
├── plugins/zap               # zapcore.Core 适配器
├── plugins/zerolog           # zerolog.Hook 适配器
└── plugins/logrus            # logrus.Hook 适配器
```

应用只引入需要的插件，不会拉到其他日志库的依赖。

## 特性

- JSON 输出格式跨所有插件完全一致，兼容 ELK 技术栈
- TCP 传输，连接断开时自动重连
- 自动附加元数据（主机名、PID、时间戳、调用位置）
- 异步写入，不阻塞业务逻辑
- 通过 `go.work` 管理多模块，插件独立版本化

## 快速开始

### slog

```go
package main

import (
    "log/slog"
    "github.com/HaoQi-Lee/pulse"
    pulseslog "github.com/HaoQi-Lee/pulse/plugins/slog"
)

func main() {
    handler := pulseslog.NewHandler(pulse.Options{
        Project:  "demo",
        Logstash: "a.b.c.d:4560",
    })
    defer handler.Close()
    slog.SetDefault(slog.New(handler))

    slog.Info("服务启动")
}
```

### zap

```go
package main

import (
    "go.uber.org/zap"
    "github.com/HaoQi-Lee/pulse"
    pulsezap "github.com/HaoQi-Lee/pulse/plugins/zap"
)

func main() {
    core := pulsezap.NewCore(pulse.Options{
        Project:  "demo",
        Logstash: "a.b.c.d:4560",
    })
    defer core.Close()
    logger := zap.New(core)

    logger.Info("服务启动")
}
```

### zerolog

```go
package main

import (
    "github.com/rs/zerolog/log"
    "github.com/HaoQi-Lee/pulse"
    pulsezerolog "github.com/HaoQi-Lee/pulse/plugins/zerolog"
)

func main() {
    hook := pulsezerolog.NewHook(pulse.Options{
        Project:  "demo",
        Logstash: "a.b.c.d:4560",
    })
    defer hook.Close()
    log.Logger = log.Hook(hook)

    log.Info().Msg("服务启动")
}
```

### logrus

```go
package main

import (
    "github.com/sirupsen/logrus"
    "github.com/HaoQi-Lee/pulse"
    pulselogrus "github.com/HaoQi-Lee/pulse/plugins/logrus"
)

func main() {
    hook := pulselogrus.NewHook(pulse.Options{
        Project:  "demo",
        Logstash: "a.b.c.d:4560",
    })
    defer hook.Close()
    logrus.AddHook(hook)

    logrus.Info("服务启动")
}
```

## Options

```go
type Options struct {
    Project    string // 项目名
    Logstash   string // Logstash TCP 地址，如 "a.b.c.d:4560"
    Service    string // 默认 "golang"
    Beat       string // 默认 "logback"
    Level      string // 默认 "info"，可选 debug/info/warn/error
    BufferSize int    // 默认 1024
}
```

## 日志格式

所有插件输出统一的 JSON 格式：

```json
{
  "thread_name": "12345",
  "host": "server-01",
  "@timestamp": "2026-04-29T10:30:00.123Z",
  "logger_name": "main.go:15",
  "@metadata": { "beat": "logback" },
  "fields": { "project": "demo", "service": "golang" },
  "message": "服务启动",
  "level": "info"
}
```

`Extra map[string]any` 字段用于承载各日志库的扩展字段（如 request_id、trace_id），`omitempty` 空时省略。

## 依赖

- Go 1.22+
