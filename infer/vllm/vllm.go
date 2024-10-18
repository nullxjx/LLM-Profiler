package vllm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/infer/param"
	"github.com/nullxjx/LLM-Profiler/infer/stream"
	"github.com/nullxjx/LLM-Profiler/infer/stream/postprocess"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	// DefaultTemperature 默认温度，控制推理结果的随机性
	defaultTemperature = 0.0
	// DefaultMaxTokens 默认最大 token 数
	defaultMaxTokens = 1000
	// DefaultTopN 默认取多少个结果
	defaultTopN = 1
	// 是否启用 BeamSearch
	defaultUseBeamSearch = false
)

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// Req 获取 CodeLLama 模型的请求参数
type Req struct {
	Model         string         `json:"model"`
	Prompt        string         `json:"prompt"`
	Stop          []string       `json:"stop"`
	Temperature   float32        `json:"temperature"`
	MaxTokens     uint32         `json:"max_tokens"`
	N             int            `json:"n"`
	UseBeamSearch bool           `json:"use_beam_search"`
	Stream        bool           `json:"stream"`
	IgnoreEos     bool           `json:"ignore_eos"`
	StreamOptions *StreamOptions `json:"stream_options"`
}

// Infer 调用 vLLM 的/v1/completions接口
func Infer(params *param.InferParams, url string) (*param.InferRsp, error) {
	if params.InferConfig.Temperature < 0.0 {
		params.InferConfig.Temperature = defaultTemperature
	}
	if params.InferConfig.MaxTokens <= 0 || params.InferConfig.MaxTokens > defaultMaxTokens {
		params.InferConfig.MaxTokens = defaultMaxTokens
	}
	req := &Req{
		Model:         params.ModelName,
		Prompt:        params.PromptList[0],
		Stop:          params.InferConfig.StopWords,
		Temperature:   params.InferConfig.Temperature,
		MaxTokens:     params.InferConfig.MaxTokens,
		N:             defaultTopN,
		UseBeamSearch: defaultUseBeamSearch,
		Stream:        false,
		IgnoreEos:     true,
	}

	url = fmt.Sprintf("http://%s/v1/completions", url)
	ctx := context.Background()
	// 设置超时时间
	//log.Infof("设置超时时间为 %d ms", params.Timeout)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Duration(params.Timeout)*time.Millisecond)
	defer cancel()
	body, err := common.Post(ctxWithTimeout, url, req)
	if err != nil {
		return nil, err
	}
	// 解析返回值，当模型有错误时，错误信息在返回体中，需要先做解析判断
	var errorRspData *param.InferErrRsp
	if err = json.Unmarshal(body, &errorRspData); err != nil {
		return nil, err
	}
	if errorRspData.Object == "error" {
		errMsg := errorRspData.Message
		return nil, errors.New(errMsg)
	}
	res := param.InferRsp{}
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func InferVLLM(params *param.InferParams, serviceURL string) ([]param.InferResult, error) {
	start := time.Now()
	result, err := Infer(params, serviceURL)
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

// StreamInferByVLLM vLLM的流式补全请求入口
func StreamInferByVLLM(ctx context.Context, url string, inferReq *param.InferParams) (
	<-chan []byte, error) {
	header := map[string]string{
		"Content-Type": "application/json",
	}
	req := &Req{
		Model:       inferReq.ModelName,
		Prompt:      inferReq.PromptList[0],
		Stop:        inferReq.InferConfig.StopWords,
		Temperature: inferReq.InferConfig.Temperature,
		MaxTokens:   inferReq.InferConfig.MaxTokens,
		N:           1,
		Stream:      true,
		StreamOptions: &StreamOptions{
			IncludeUsage: true,
		},
	}
	// 拼接vllm流式url
	url = fmt.Sprintf("http://%s/v1/completions", url)
	rsp, err := stream.Stream(ctx, url, header, nil, req)
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

func GetVllmStreamTokens(chunk string) (int, int, error) {
	// 使用正则表达式匹配 "data:" 开头的字符串
	re := regexp.MustCompile(`^data:\s*(\{.*\})`)
	matches := re.FindStringSubmatch(chunk)

	if len(matches) != 2 {
		return 0, 0, errors.New("invalid input format")
	}

	jsonData := matches[1]

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return 0, 0, err
	}

	usageData, ok := data["usage"].(map[string]interface{})
	if !ok {
		return 0, 0, errors.New("invalid usage data")
	}

	completionTokens, ok := usageData["completion_tokens"].(float64)
	if !ok {
		return 0, 0, errors.New("invalid completion tokens")
	}

	promptTokens, ok := usageData["prompt_tokens"].(float64)
	if !ok {
		return 0, 0, errors.New("invalid prompt tokens")
	}
	return int(completionTokens), int(promptTokens), nil
}
