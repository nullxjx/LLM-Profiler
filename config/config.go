package config

import (
	"os"

	"github.com/spf13/viper"
)

const (
	SecretID   = "secretID"
	SecretKey  = "secretKey"
	WebhookUrl = "webhookUrl"
	Bucket     = "bucket"
	Region     = "region"
	SubFolder  = "subFolder"
)

type ModelConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// Config 整体配置
type Config struct {
	Model                 ModelConfig `yaml:"model"`
	ServerIp              string      `yaml:"serverIp"`
	Port                  int         `yaml:"port"`
	RequestTimeout        int         `yaml:"requestTimeout"` // 单位为毫秒
	Backend               string      `yaml:"backend"`
	StopWords             []string    `yaml:"stopWords"`
	MaxTokens             uint32      `yaml:"maxTokens"`
	Temperature           float32     `yaml:"temperature"`
	Stream                bool        `yaml:"stream"`
	InputTokens           int         `yaml:"inputTokens"`
	StartConcurrency      int         `yaml:"startConcurrency"`
	EndConcurrency        int         `yaml:"endConcurrency"`
	Increment             int         `yaml:"increment"`
	Duration              int         `yaml:"duration"`
	TimeThresholds        []int64     `yaml:"timeThresholds"`
	StreamSpeedThresholds int         `yaml:"streamSpeedThresholds"`
	MaxStreamSpeed        float64     `yaml:"maxStreamSpeed"`
	SaveDir               string      `yaml:"saveDir"`
	SendMsg               bool        `yaml:"sendMsg"`
	User                  string      `yaml:"user"`
	Save2Cos              bool        `json:"save2Cos"`
}

// ReadConf 读取配置
func ReadConf(configPath string) (*Config, error) {
	// 如果环境变量中存在值，则使用环境变量的值更新
	if cp := os.Getenv("CONFIG_PATH"); cp != "" {
		configPath = cp
	}
	//viper.AutomaticEnv()
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configPath)
	//viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	config := &Config{}
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	if config.Temperature == 0. { // 温度默认设置为1
		config.Temperature = 1
	}
	return config, nil
}
