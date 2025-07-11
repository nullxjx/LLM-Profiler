package triton

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/internal/infer/stream/postprocess"
	"github.com/nullxjx/llm_profiler/internal/infer/type/stream"
	"github.com/nullxjx/llm_profiler/pkg/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func InferTrt(params *param.InferParams, url string) ([]param.InferResult, error) {
	req := &TrtReq{
		TextInput:   params.PromptList[0],
		MaxTokens:   int32(params.InferConfig.MaxTokens),
		StopWords:   strings.Join(params.InferConfig.StopWords, ","),
		Stream:      false,
		TopP:        params.InferConfig.TopP,
		Temperature: params.InferConfig.Temperature,
	}
	start := time.Now()
	url = fmt.Sprintf("%s/v2/models/%s/generate", url, params.ModelName)
	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Millisecond)
	defer cancel()
	body, err := http.Post(ctxWithTimeout, url, req)
	if err != nil {
		return nil, err
	}
	var rsp *TrtRsp
	if err = json.Unmarshal(body, &rsp); err != nil {
		return nil, err
	}

	var res *param.InferRsp
	if err = json.Unmarshal([]byte(rsp.TextOutput), &res); err != nil {
		return nil, err
	}

	var inferResults []param.InferResult
	for _, r := range res.Choices {
		// 如果设置beamWidth > 1，对于每条输入，都会有多条输出，这里简单起见，只取第一条输出作为最后的输出
		inferResults = append(inferResults, param.InferResult{
			Result:       r.Text,
			TimeSpent:    time.Now().Sub(start).Milliseconds(),
			InputTokens:  int(res.Usage.PromptTokens),
			OutputTokens: int(res.Usage.CompletionTokens),
		})
	}
	return inferResults, nil
}

// StreamInferByTrt trt的流式请求入口
func StreamInferByTrt(ctx context.Context, url string, params *param.InferParams) (
	<-chan []byte, error) {
	header := map[string]string{
		"Content-Type": "application/json",
	}
	req := &TrtReq{
		TextInput:   params.PromptList[0],
		MaxTokens:   int32(params.InferConfig.MaxTokens),
		StopWords:   strings.Join(params.InferConfig.StopWords, ","),
		Stream:      true,
		TopP:        params.InferConfig.TopP,
		Temperature: params.InferConfig.Temperature,
	}
	// 拼接trt流式url
	url = fmt.Sprintf("%s/v2/models/%s/generate_stream", url, params.ModelName)
	rsp, err := http.Stream(ctx, url, header, nil, req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Call trt stream API error, model:%v, err: %v", params.ModelName, err))
	}
	out := make(chan []byte, 4096)
	go func() {
		s := &postprocess.TrtStreamHandler{
			Type:  stream.Completion,
			Model: params.ModelName,
		}
		if err = s.Handle(ctx, out, rsp); err != nil {
			log.Errorf("call trt stream API error: %v", err)
		}
	}()
	return out, nil
}
