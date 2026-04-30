package main

import (
	"errors"

	"git.showcai.com.cn/tech/pulse"
	pulselogrus "git.showcai.com.cn/tech/pulse/plugins/logrus"
	"github.com/sirupsen/logrus"
)

func main() {
	hook := pulselogrus.NewHook(pulse.Options{
		Project:  "logrus-demo",
		Logstash: "10.141.48.10:4560",
	})
	defer hook.Close()
	logrus.AddHook(hook)

	logrus.Info("服务启动")
	logrus.Warn("磁盘空间不足")
	logrus.Debug("调试信息")
	logrus.Error("连接超时: ", errors.New("timeout"))
}
