package config

import (
	"flag"
	"github.com/BurntSushi/toml"
	zjcLog "github.com/zhengjingcheng/zjcgo/log"
	"os"
)

var Conf = &ZjcConfig{
	logger: zjcLog.Default(),
}

type ZjcConfig struct {
	logger *zjcLog.Logger
	Log    map[string]any
	Pool   map[string]any
}

func init() {
	loadToml()
}
func loadToml() {
	//如果不指定就用默认的
	configFile := flag.String("conf", "conf/app.toml", "app  config file")
	flag.Parse()
	if _, err := os.Stat(*configFile); err != nil {
		Conf.logger.Info("conf/app.toml file not load，because not exist")
		return
	}
	_, err := toml.DecodeFile(*configFile, Conf)
	if err != nil {
		Conf.logger.Info("conf/app.toml decode fail check format")
		panic(err)
	}
}
