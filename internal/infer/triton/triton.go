package triton

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/pkg/http"
)

type TrtReq struct {
	TextInput   string  `json:"text_input"`
	MaxTokens   int32   `json:"max_tokens"`
	BadWords    string  `json:"bad_words"`
	StopWords   string  `json:"stop_words"`
	Stream      bool    `json:"stream"`
	TopP        float32 `json:"top_p"`
	Temperature float32 `json:"temperature"`
}

type TrtRsp struct {
	ModelName     string `json:"model_name"`
	ModelVersion  string `json:"model_version"`
	SequenceEnd   bool   `json:"sequence_end"`
	SequenceId    int    `json:"sequence_id"`
	SequenceStart bool   `json:"sequence_start"`
	TextOutput    string `json:"text_output"`
}

type VllmReq struct {
	Stream      bool    `json:"stream"`
	Temperature float32 `json:"temperature"`
	MaxTokens   int32   `json:"max_tokens"`
	TextInput   string  `json:"text_input"`
}

type VllmRsp struct {
	ModelName    string          `json:"model_name"`
	ModelVersion string          `json:"model_version"`
	TextOutput   json.RawMessage `json:"text_output"`
}

type ProxyRequest struct {
	Model         string   `json:"model"`
	Prompt        string   `json:"prompt"`
	Stop          []string `json:"stop,omitempty"`
	Temperature   float32  `json:"temperature"`
	MaxTokens     uint32   `json:"max_tokens"`
	UseBeamSearch bool     `json:"use_beam_search"`
	Stream        bool     `json:"stream"`
	IgnoreEos     bool     `json:"ignore_eos"`
}

func InferVllmInTriton(p *param.InferParams, url string) ([]param.InferResult, error) {
	req := &ProxyRequest{
		Model:         p.ModelName,
		Prompt:        p.PromptList[0],
		Stop:          p.InferConfig.StopWords,
		Temperature:   p.InferConfig.Temperature,
		MaxTokens:     p.InferConfig.MaxTokens,
		UseBeamSearch: false,
		Stream:        false,
		IgnoreEos:     true,
	}

	start := time.Now()
	url = fmt.Sprintf("http://%s/v1/completions", url)
	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(p.Timeout)*time.Millisecond)
	defer cancel()
	body, err := http.Post(ctxWithTimeout, url, req)
	if err != nil {
		return nil, err
	}

	var inferRsp param.InferRsp
	err = json.Unmarshal(body, &inferRsp)
	if err != nil {
		fmt.Println("Error unmarshaling InferRsp:", err)
		return nil, err
	}

	var res []param.InferResult
	for _, r := range inferRsp.Choices {
		// 如果设置beamWidth > 1，对于每条输入，都会有多条输出，这里简单起见，只取第一条输出作为最后的输出
		res = append(res, param.InferResult{
			Result:       r.Text,
			TimeSpent:    time.Now().Sub(start).Milliseconds(),
			InputTokens:  inferRsp.Usage.PromptTokens,
			OutputTokens: inferRsp.Usage.CompletionTokens,
		})
	}
	return res, nil
}
