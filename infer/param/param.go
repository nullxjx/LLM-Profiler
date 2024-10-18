package param

import (
	"sync"
)

type RequestParam struct {
	Wg           *sync.WaitGroup
	Prompts      []string
	Result       chan<- Result
	SuccessCount *int32
	FailedCount  *int32
	TotalCount   *int32
	InputIndex   int

	ServiceURL   string
	ModelName    string
	ModelVersion string
	Timeout      int // 单位为毫秒

	StopWords []string
	MaxTokens uint32
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
	Prompt      string `json:"prompt"`
	InputLen    int    `json:"inputLen"`
	InputTokens int    `json:"inputTokens"`

	Output       string `json:"output"`
	OutputLen    int    `json:"outputLen"`
	OutputTokens int    `json:"outputTokens"`

	TimeSpent int64 `json:"timeSpent"`

	OutputTokensPerSecond float64 `json:"outputTokensPerSecond"`
}

type InferResult struct {
	Result       string `json:"result"`
	TimeSpent    int64  `json:"timeSpent"`
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
}
