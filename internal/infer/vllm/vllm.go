package vllm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/internal/infer/stream/postprocess"
	"github.com/nullxjx/llm_profiler/internal/infer/type/stream"
	"github.com/nullxjx/llm_profiler/pkg/http"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
)

// CompletionReq vllm 补全请求参数
type CompletionReq struct {
	openai.CompletionRequest
	*openai.StreamOptions `json:"stream_options,omitempty"`
	IgnoreEos             bool `json:"ignore_eos"`
}

// Completion 调用 vLLM 的/v1/completions接口
func Completion(params *param.InferParams, url string) (*param.InferRsp, error) {
	req := &CompletionReq{
		CompletionRequest: openai.CompletionRequest{
			Model:       params.ModelName,
			Prompt:      params.PromptList[0],
			Stop:        params.InferConfig.StopWords,
			Temperature: params.InferConfig.Temperature,
			MaxTokens:   int(params.InferConfig.MaxTokens),
			N:           1,
			Stream:      false,
		},
		IgnoreEos: true,
	}

	url = fmt.Sprintf("%s/v1/completions", url)
	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Millisecond)
	defer cancel()
	body, err := http.Post(ctxWithTimeout, url, req)
	if err != nil {
		return nil, err
	}
	var res param.InferRsp
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// CompletionByVLLM 调用 vLLM 的/v1/completions接口，并返回统计信息
func CompletionByVLLM(params *param.InferParams, serviceURL string) ([]param.InferResult, error) {
	start := time.Now()
	result, err := Completion(params, serviceURL)
	if err != nil {
		return nil, err
	}

	// 对推理结果进行提取
	var inferResults []param.InferResult
	for _, choice := range result.Choices {
		inferResults = append(inferResults, param.InferResult{
			Result:       choice.Text,
			TimeSpent:    time.Now().Sub(start).Milliseconds(),
			InputTokens:  int(result.Usage.PromptTokens),
			OutputTokens: int(result.Usage.CompletionTokens),
		})
	}

	return inferResults, nil
}

// StreamCompletionByVLLM vLLM的流式补全请求入口
func StreamCompletionByVLLM(ctx context.Context, url string, params *param.InferParams) (
	<-chan []byte, error) {
	header := map[string]string{
		"Content-Type": "application/json",
	}
	req := &CompletionReq{
		CompletionRequest: openai.CompletionRequest{
			Model:       params.ModelName,
			Prompt:      params.PromptList[0],
			Stop:        params.InferConfig.StopWords,
			Temperature: params.InferConfig.Temperature,
			MaxTokens:   int(params.InferConfig.MaxTokens),
			N:           1,
			Stream:      true,
		},
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
		IgnoreEos: true,
	}
	// 拼接vllm流式url
	url = fmt.Sprintf("%s/v1/completions", url)
	rsp, err := http.Stream(ctx, url, header, nil, req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Call vLLM completions API error, model:%v, err: %v", req.Model, err))
	}
	out := make(chan []byte, 4096)
	go func() {
		s := &postprocess.VllmStreamHandler{
			Type:  stream.Completion,
			Model: req.Model,
		}
		if err = s.Handle(ctx, out, rsp); err != nil {
			log.Errorf("Stream vLLM completions API error: %v", err)
		}
	}()
	return out, nil
}

// StreamChatByVLLM vLLM的流式对话请求入口
func StreamChatByVLLM(ctx context.Context, url string, params *param.InferParams) (
	<-chan []byte, error) {
	header := map[string]string{
		"Content-Type": "application/json",
	}
	var msgs []openai.ChatCompletionMessage
	msgs = append(msgs, openai.ChatCompletionMessage{
		Role:    "system",
		Content: "You are a helpful assistant.",
	})
	msgs = append(msgs, openai.ChatCompletionMessage{
		Role:    "user",
		Content: params.PromptList[0],
	})
	req := &openai.ChatCompletionRequest{
		Model:       params.ModelName,
		Messages:    msgs,
		MaxTokens:   int(params.InferConfig.MaxTokens),
		Temperature: params.InferConfig.Temperature,
		TopP:        params.InferConfig.TopP,
		N:           1,
		Stream:      true,
		Stop:        params.InferConfig.StopWords,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	}
	// 拼接vllm流式url
	url = fmt.Sprintf("%s/v1/chat/completions", url)
	rsp, err := http.Stream(ctx, url, header, nil, req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Call vLLM chat API error, model:%v, err: %v", req.Model, err))
	}
	out := make(chan []byte, 4096)
	go func() {
		s := &postprocess.VllmStreamHandler{
			Type:  stream.Chat,
			Model: req.Model,
		}
		if err = s.Handle(ctx, out, rsp); err != nil {
			log.Errorf("Stream vLLM chat API error: %v", err)
		}
	}()
	return out, nil
}
