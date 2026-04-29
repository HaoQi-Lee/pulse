package main

import (
	"fmt"
	"git.showcai.com.cn/tech/pulse"
)

func main() {
	defer pulse.Setup("demo", "10.141.48.10:4560")()

	pulse.Info("服务启动")
	pulse.Warn("磁盘空间不足")
	pulse.Debug("调试信息")
	pulse.Error(fmt.Errorf("连接超时"))
}
