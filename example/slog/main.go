package main

import (
	"errors"
	"log/slog"

	"git.showcai.com.cn/tech/pulse"
	pulseslog "git.showcai.com.cn/tech/pulse/plugins/slog"
)

func main() {
	handler := pulseslog.NewHandler(pulse.Options{
		Project:  "demo",
		Logstash: "10.141.48.10:4560",
	})
	defer handler.Close()
	slog.SetDefault(slog.New(handler))

	slog.Info("服务启动")
	slog.Warn("磁盘空间不足")
	slog.Debug("调试信息")
	slog.Error("连接超时", "err", errors.New("timeout"))
}
