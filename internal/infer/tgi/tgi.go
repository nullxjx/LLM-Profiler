package tgi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/pkg/http"
)

type Parameters struct {
	MaxNewTokens        uint32  `json:"max_new_tokens"`
	Details             bool    `json:"details"`
	DecoderInputDetails bool    `json:"decoder_input_details"`
	Temperature         float32 `json:"temperature"`
	DoSample            bool    `json:"do_sample"`
}

type InferReq struct {
	Inputs     string     `json:"inputs"`
	Parameters Parameters `json:"parameters"`
}

type Token struct {
	ID      int     `json:"id"`
	Text    string  `json:"text"`
	Logprob float64 `json:"logprob"`
	Special bool    `json:"special"`
}

type Prefill struct {
	ID      int     `json:"id"`
	Text    string  `json:"text"`
	Logprob float64 `json:"logprob"`
}

type Details struct {
	FinishReason    string    `json:"finish_reason"`
	GeneratedTokens int       `json:"generated_tokens"`
	Seed            uint64    `json:"seed"`
	Prefill         []Prefill `json:"prefill"`
	Tokens          []Token   `json:"tokens"`
}

type InferRsp struct {
	GeneratedText string  `json:"generated_text"`
	Details       Details `json:"details"`
}

// InferTGI 调用 TGI 的generate 接口
func InferTGI(params *param.InferParams, url string) ([]param.InferResult, error) {
	req := &InferReq{
		Inputs: params.PromptList[0],
		Parameters: Parameters{
			MaxNewTokens:        params.InferConfig.MaxTokens,
			Details:             true,
			DecoderInputDetails: true,
			Temperature:         params.InferConfig.Temperature,
			DoSample:            true,
		},
	}
	start := time.Now()
	url = fmt.Sprintf("http://%s/generate", url)
	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Millisecond)
	defer cancel()
	body, err := http.Post(ctxWithTimeout, url, req)
	if err != nil {
		return nil, err
	}
	var rsp *InferRsp
	if err = json.Unmarshal(body, &rsp); err != nil {
		return nil, err
	}
	return []param.InferResult{
		{
			Result:       rsp.GeneratedText,
			TimeSpent:    time.Now().Sub(start).Milliseconds(),
			InputTokens:  len(rsp.Details.Prefill),
			OutputTokens: len(rsp.Details.Tokens),
		},
	}, nil
}
