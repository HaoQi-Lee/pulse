package main

import (
	"errors"
	"log/slog"

	"github.com/leehoawki/pulse"
	pulseslog "github.com/leehoawki/pulse/plugins/slog"
)

func main() {
	handler := pulseslog.NewHandler(pulse.Options{
		Project:  "slog-demo",
		Logstash: "10.141.48.10:4560",
	})
	defer handler.Close()
	slog.SetDefault(slog.New(handler))

	slog.Info("服务启动")
	slog.Warn("磁盘空间不足")
	slog.Debug("调试信息")
	slog.Error("连接超时", "err", errors.New("timeout"))
}
