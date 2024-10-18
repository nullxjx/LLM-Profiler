package param

// InferErrRsp 代码补全报错结果
type InferErrRsp struct {
	Object  string      `json:"object"`
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Param   interface{} `json:"param"`
	Code    interface{} `json:"code"`
}

// InferRsp 代码补全结果
type InferRsp struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []*Choice `json:"choices"`
	Usage   *Usage    `json:"usage"`
}

// Choice 推理结果相关
type Choice struct {
	Index        int          `json:"index"`
	Text         string       `json:"text"`
	Logprobs     *LogProbInfo `json:"logprobs"`
	FinishReason string       `json:"finish_reason"`
}

// Usage token 相关统计
type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

// LogProbInfo 概率分布信息
type LogProbInfo struct {
	TextOffset    []int32              `json:"text_offset,omitempty"`
	TokenLogprobs []float32            `json:"token_logprobs,omitempty"`
	Tokens        []string             `json:"tokens,omitempty"`
	TopLogprobs   []map[string]float32 `json:"top_logprobs,omitempty"`
}
