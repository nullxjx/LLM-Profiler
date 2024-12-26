package param

import (
	"sync"

	"github.com/nullxjx/llm_profiler/config"

	"github.com/sashabaranov/go-openai"
)

type Counter struct {
	Success int32
	Failed  int32
	Total   int32
}
type RequestParam struct {
	Wg      *sync.WaitGroup
	Result  chan<- Result
	Prompt  string
	Counter *Counter
	Config  *config.Config
}

type InferConfig struct {
	StopWords   []string
	MaxTokens   uint32
	TopK        uint32
	TopP        float32
	BeamWidth   uint32
	Temperature float32
}

type InferParams struct {
	PromptList   []string
	ModelName    string
	ModelVersion string
	Timeout      int // 超时时间，单位为毫秒
	InferConfig  *InferConfig
}

type PromptSpentTime struct {
	Prompt    string  `json:"prompt"`
	SpentTime float64 `json:"spentTime"`
}

// Result 该轮次调用结果记录
type Result struct {
	Prompt          string  `json:"prompt"`
	InputLen        int     `json:"inputLen"`
	InputTokens     int     `json:"inputTokens"`
	Output          string  `json:"output"`
	OutputLen       int     `json:"outputLen"`
	OutputTokens    int     `json:"outputTokens"`
	TimeSpent       int64   `json:"timeSpent"`
	TokensPerSecond float64 `json:"tokensPerSecond"` // 每秒输出token数目
	FirstTokenTime  float64 `json:"firstTokenTime"`
}

type InferResult struct {
	Result       string `json:"result"`
	TimeSpent    int64  `json:"timeSpent"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}

type InferRsp openai.CompletionResponse
