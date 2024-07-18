package conf

import (
	"gopkg.in/yaml.v3"
	"os"
	"tx/logs"
)

type AppConfig struct {
	AppId              string `yaml:"appId"`
	ClientSecret       string `yaml:"clientSecret"`
	QqBaseUrl          string `yaml:"qqBaseUrl"`
	OpenaiBaseUrl      string `yaml:"openaiBaseUrl"`
	OpenaiKey          string `yaml:"openaiKet"`
	OpenaiDefaultModel string `yaml:"openaiDefaultModel"`
	OpenaiMaxHistory   int    `yaml:"openaiMaxHistory"`
}

var appConf = &AppConfig{}

func LoadFromFile(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return LoadContent(content)
}

func LoadContent(content []byte) error {
	conf := &AppConfig{}
	err := yaml.Unmarshal(content, conf)
	if err != nil {
		return err
	}
	logs.Logger.Debugf("Load app config: %v", conf)
	SetAppConf(conf)
	return nil
}

func SetAppConf(cfg *AppConfig) {
	logs.Logger.Info("Load app config success")
	appConf = cfg
}

func GetAppConf() *AppConfig {
	return appConf
}
