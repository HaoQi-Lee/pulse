package main

import (
	"errors"

	"github.com/leehoawki/pulse"
	pulsezerolog "github.com/leehoawki/pulse/plugins/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	hook := pulsezerolog.NewHook(pulse.Options{
		Project:  "zerolog-demo",
		Logstash: "a.b.c.d:4560",
	})
	defer hook.Close()
	log.Logger = log.Hook(hook)

	log.Info().Msg("服务启动")
	log.Warn().Msg("磁盘空间不足")
	log.Debug().Msg("调试信息")
	log.Error().Err(errors.New("timeout")).Msg("连接超时")
}
