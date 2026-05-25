package main

import (
	"github.com/HaoQi-Lee/pulse"
	pulsezap "github.com/HaoQi-Lee/pulse/plugins/zap"
	"go.uber.org/zap"
)

func main() {
	core := pulsezap.NewCore(pulse.Options{
		Project:  "zap-demo",
		Logstash: "a.b.c.d:4560",
	})
	defer core.Close()
	logger := zap.New(core)

	logger.Info("服务启动")
	logger.Warn("磁盘空间不足")
	logger.Debug("调试信息")
	logger.Error("连接超时")
}
