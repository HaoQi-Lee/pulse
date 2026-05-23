package main

import (
	"errors"

	"github.com/leehoawki/pulse"
	pulselogrus "github.com/leehoawki/pulse/plugins/logrus"
	"github.com/sirupsen/logrus"
)

func main() {
	hook := pulselogrus.NewHook(pulse.Options{
		Project:  "logrus-demo",
		Logstash: "a.b.c.d:4560",
	})
	defer hook.Close()
	logrus.AddHook(hook)

	logrus.Info("服务启动")
	logrus.Warn("磁盘空间不足")
	logrus.Debug("调试信息")
	logrus.Error("连接超时: ", errors.New("timeout"))
}
