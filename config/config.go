package config

import (
	"os"

	"github.com/spf13/viper"
)

const (
	EnvSecretID   = "secretID"
	EnvSecretKey  = "secretKey"
	EnvWebhookUrl = "webhookUrl"
	EnvBucket     = "bucket"
	EnvRegion     = "region"
	EnvSubFolder  = "subFolder"
	EnvConfigPath = "configPath"
)

type ModelConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// Config 整体配置
type Config struct {
	Model            ModelConfig `yaml:"model"`            // 模型配置
	ServerIp         string      `yaml:"serverIp"`         // 模型服务ip
	Port             int         `yaml:"port"`             // 模型服务端口
	RequestTimeout   int         `yaml:"requestTimeout"`   // 单位为毫秒
	Backend          string      `yaml:"backend"`          // 推理后端类型，例如 vllm、trt、tgi
	StopWords        []string    `yaml:"stopWords"`        // stop words
	MaxTokens        uint32      `yaml:"maxTokens"`        // 生成token的最大数量
	Temperature      float32     `yaml:"temperature"`      // 模型温度
	Stream           bool        `yaml:"stream"`           // 是否流式
	InputTokens      int         `yaml:"inputTokens"`      // 输入token数量
	StartConcurrency int         `yaml:"startConcurrency"` // 开始并发度，并发度指的是给定时间内发送的请求数目
	EndConcurrency   int         `yaml:"endConcurrency"`   // 结束并发度
	Increment        int         `yaml:"increment"`        // 并发度每一轮跟上一轮的增量
	Duration         int         `yaml:"duration"`         // 每一轮请求持续时间，单位是分钟
	TimeThresholds   []int64     `yaml:"timeThresholds"`   // 请求时间阈值
	StreamThresholds int         `yaml:"streamThresholds"` // 流式模式下，当客户端流式速度低于最大流式速度的百分比时，停止发送请求
	MaxStreamSpeed   float64     `yaml:"maxStreamSpeed"`   // 最大流式速度，在流式场景才有效，如果没有设置，则会先测试最大流式速度
	SaveDir          string      `yaml:"saveDir"`          // 压测结果保存路径
	SendMsg          bool        `yaml:"sendMsg"`          // 是否发送企微webhook消息
	User             string      `yaml:"user"`             // 企微群中的用户
	Save2Cos         bool        `json:"save2Cos"`         // 是否保存结果到cos
}

// ReadConf 读取配置
func ReadConf(configPath string) (*Config, error) {
	// 如果环境变量中存在值，则使用环境变量的值更新
	if cp := os.Getenv(EnvConfigPath); cp != "" {
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
