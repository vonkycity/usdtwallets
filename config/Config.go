package config

import (
	"log"

	"eth.url4g.com/myutils"
	"github.com/spf13/viper"
)

//SslConfig struct
type SslConfig struct {
	Crt  string `yaml:"crt"`
	Key  string `yaml:"key"`
	Port int    `yaml:"port"`
}

//ServerConfig struct
type ServerConfig struct {
	Addr    string `yaml:"addr"`
	Port    int    `yaml:"port"`
	AuthKey string `yaml:"authkey"`
}

//MysqlConfig struct
type MysqlConfig struct {
	Connect     string `yaml:"connect"`
	Tableprefix string `yaml:"tableprefix"`
}

//SecConfig struct
type SecConfig struct {
	Ethkey string `yaml:"ethkey"`
	Wallet string `yaml:"wallet"`
}

//MainConfig struct
type MainConfig struct {
	GethURL string       `yaml:"gethurl"`
	Ssl     SslConfig    `yaml:"ssl"`
	Server  ServerConfig `yaml:"server"`
	Mysql   MysqlConfig  `yaml:"mysql"`
	Sec     SecConfig    `yaml:"sec"`
}

var config *MainConfig

//Init 初始化
func Init() {
	log.Println("Config Init")
}

// GetConfig 获取config
func GetConfig() *MainConfig {
	if config == nil {
		vp := viper.New()
		vp.SetConfigType("yaml")
		vp.SetConfigName("config")
		vp.AddConfigPath(myutils.GetCurrentPath())
		err := vp.ReadInConfig()
		if err != nil {
			panic("config file error")
		}
		vp.Unmarshal(&config)
	}
	return config
	//
}
