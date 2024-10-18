package triton

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/infer/param"
)

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
	body, err := common.Post(ctxWithTimeout, url, req)
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
			InputTokens:  int(inferRsp.Usage.PromptTokens),
			OutputTokens: int(inferRsp.Usage.CompletionTokens),
		})
	}
	return res, nil
}
