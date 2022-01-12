package conf

import (
	"liquidator/log"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type ConfigStruct struct {
	Chainid     int64
	Subgraph    string
	Infura      string
	Comptroller string
	Wallet      string
	Log         Log
}

type Log struct {
	FileDir  string
	FileName string
	Prefix   string
	Level    string
}

var Config ConfigStruct

func Init() {
	viper := viper.New()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./conf/")
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	viper.Unmarshal(&Config)
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		err = viper.ReadInConfig()
		if err == nil {
			viper.Unmarshal(&Config)
			log.Printf("config: %+v", Config)
		} else {
			log.Printf("ReadInConfig error: %s", err)
		}
	})
}
