package tgi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/infer/param"
)

const (
	// DefaultTemperature 默认温度，控制推理结果的随机性
	defaultTemperature = 0.2
	// DefaultMaxTokens 默认最大 token 数
	defaultMaxTokens = 1000
)

// InferTGI 调用 TGI 的generate 接口
func InferTGI(params *param.InferParams, url string) (*InferRsp, error) {
	if params.InferConfig.Temperature < 0.0 {
		params.InferConfig.Temperature = defaultTemperature
	}
	if params.InferConfig.MaxTokens <= 0 || params.InferConfig.MaxTokens > defaultMaxTokens {
		params.InferConfig.MaxTokens = defaultMaxTokens
	}
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

	url = fmt.Sprintf("http://%s/generate", url)
	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Millisecond)
	defer cancel()
	body, err := common.Post(ctxWithTimeout, url, req)
	if err != nil {
		return nil, err
	}
	var rsp *InferRsp
	if err = json.Unmarshal(body, &rsp); err != nil {
		return nil, err
	}
	return rsp, nil
}
