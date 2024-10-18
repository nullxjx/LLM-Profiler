package infer

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/nullxjx/LLM-Profiler/infer/param"
	"github.com/nullxjx/LLM-Profiler/infer/tgi"
	"github.com/nullxjx/LLM-Profiler/infer/triton"
	"github.com/nullxjx/LLM-Profiler/infer/vllm"
	log "github.com/sirupsen/logrus"
)

func SendTritonVllmRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	prompt := req.Prompts[req.InputIndex]

	inferParam := &param.InferParams{
		PromptList:   []string{prompt},
		ModelName:    req.ModelName,
		ModelVersion: req.ModelVersion,
		Timeout:      req.Timeout,
		InferConfig: &param.InferConfig{
			StopWords:   req.StopWords,
			MaxTokens:   req.MaxTokens,
			Temperature: 1,
		},
	}
	result, err := triton.InferVllmInTriton(inferParam, req.ServiceURL)
	atomic.AddInt32(req.TotalCount, 1)
	if err != nil {
		log.Errorf("😭 infer error: %v", err)
		atomic.AddInt32(req.FailedCount, 1)
		return
	}
	atomic.AddInt32(req.SuccessCount, 1)
	req.Result <- param.Result{
		Prompt:       prompt,
		InputLen:     len(prompt),
		InputTokens:  result[0].InputTokens,
		Output:       result[0].Result,
		OutputLen:    len(result[0].Result),
		OutputTokens: result[0].OutputTokens,
		TimeSpent:    result[0].TimeSpent,
	}
}

// SendVllmRequest 发送 vllm 请求
func SendVllmRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	prompt := req.Prompts[req.InputIndex]

	result, err := vllm.InferVLLM(&param.InferParams{
		PromptList:   []string{prompt},
		ModelName:    req.ModelName,
		ModelVersion: req.ModelVersion,
		Timeout:      req.Timeout,
		InferConfig: &param.InferConfig{
			StopWords:   req.StopWords,
			MaxTokens:   req.MaxTokens,
			Temperature: 1,
		},
	}, req.ServiceURL)
	atomic.AddInt32(req.TotalCount, 1)
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(req.FailedCount, 1)
		return
	}
	atomic.AddInt32(req.SuccessCount, 1)
	req.Result <- param.Result{
		Prompt:       prompt,
		InputLen:     len(prompt),
		InputTokens:  result[0].InputTokens,
		Output:       result[0].Result,
		OutputLen:    len(result[0].Result),
		OutputTokens: result[0].OutputTokens,
		TimeSpent:    result[0].TimeSpent,
	}
}

// SendVllmStreamRequest 发送 vllm 流式请求
func SendVllmStreamRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	prompt := req.Prompts[req.InputIndex]
	atomic.AddInt32(req.TotalCount, 1)

	start := time.Now()
	stream, err := vllm.StreamInferByVLLM(context.Background(), req.ServiceURL,
		&param.InferParams{
			PromptList:   []string{prompt},
			ModelName:    req.ModelName,
			ModelVersion: req.ModelVersion,
			Timeout:      req.Timeout,
			InferConfig: &param.InferConfig{
				StopWords:   req.StopWords,
				MaxTokens:   req.MaxTokens,
				Temperature: 1,
			},
		})
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(req.FailedCount, 1)
		return
	}

	completionTokens := 0
	promptTokens := 0
	for data := range stream {
		cTokens, pTokens, err := vllm.GetVllmStreamTokens(string(data))
		if err != nil {
			continue
		}
		completionTokens = cTokens
		promptTokens = pTokens
	}

	var outputTokensPerSecond float64 = 0
	timeSpentSeconds := float64(time.Now().Sub(start)) / float64(time.Second)
	speed := float64(completionTokens) / timeSpentSeconds
	if completionTokens == int(req.MaxTokens) {
		log.Infof("stream completion tokens: %d, time spent: %.1fs, speed: %.1ftokens/s",
			completionTokens, timeSpentSeconds, speed)
		outputTokensPerSecond = speed
	}
	atomic.AddInt32(req.SuccessCount, 1)
	req.Result <- param.Result{
		Prompt:                prompt,
		InputLen:              len(prompt),
		InputTokens:           promptTokens,
		Output:                "", // 不重要，暂时不统计
		OutputLen:             0,  // 不重要，暂时不统计
		OutputTokens:          completionTokens,
		TimeSpent:             time.Now().Sub(start).Milliseconds(),
		OutputTokensPerSecond: outputTokensPerSecond,
	}
}

// SendTgiRequest 发送 Tgi 请求
func SendTgiRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	prompt := req.Prompts[req.InputIndex]

	start := time.Now()
	inferParam := &param.InferParams{
		PromptList:   []string{prompt},
		ModelName:    req.ModelName,
		ModelVersion: req.ModelVersion,
		Timeout:      req.Timeout,
		InferConfig: &param.InferConfig{
			StopWords:   req.StopWords,
			MaxTokens:   req.MaxTokens,
			Temperature: 1,
		},
	}
	result, err := tgi.InferTGI(inferParam, req.ServiceURL)
	atomic.AddInt32(req.TotalCount, 1)
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(req.FailedCount, 1)
		return
	}
	atomic.AddInt32(req.SuccessCount, 1)
	req.Result <- param.Result{
		Prompt:       prompt,
		InputLen:     len(prompt),
		InputTokens:  len(result.Details.Prefill),
		Output:       result.GeneratedText,
		OutputLen:    len(result.GeneratedText),
		OutputTokens: len(result.Details.Tokens),
		TimeSpent:    time.Now().Sub(start).Milliseconds(),
	}
}

// SendTrtRequest 发送 TensorRT-LLM 请求
func SendTrtRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	prompt := req.Prompts[req.InputIndex]

	inferParam := &param.InferParams{
		PromptList:   []string{prompt},
		ModelName:    req.ModelName,
		ModelVersion: req.ModelVersion,
		Timeout:      req.Timeout,
		InferConfig: &param.InferConfig{
			StopWords:   req.StopWords,
			MaxTokens:   req.MaxTokens,
			Temperature: 1,
		},
	}
	result, err := triton.InferTrt(inferParam, req.ServiceURL)
	atomic.AddInt32(req.TotalCount, 1)
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(req.FailedCount, 1)
		return
	}
	atomic.AddInt32(req.SuccessCount, 1)
	req.Result <- param.Result{
		Prompt:       prompt,
		InputLen:     len(prompt),
		InputTokens:  result[0].InputTokens,
		Output:       result[0].Result,
		OutputLen:    len(result[0].Result),
		OutputTokens: int(req.MaxTokens),
		TimeSpent:    result[0].TimeSpent,
	}
}

// SendTrtStreamRequest 发送 TensorRT-LLM 流式请求
func SendTrtStreamRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	prompt := req.Prompts[req.InputIndex]
	atomic.AddInt32(req.TotalCount, 1)

	start := time.Now()
	stream, err := triton.StreamInferByTrt(context.Background(), req.ServiceURL,
		&param.InferParams{
			PromptList:   []string{prompt},
			ModelName:    req.ModelName,
			ModelVersion: req.ModelVersion,
			Timeout:      req.Timeout,
			InferConfig: &param.InferConfig{
				StopWords:   req.StopWords,
				MaxTokens:   req.MaxTokens,
				Temperature: 1,
			},
		})
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(req.FailedCount, 1)
		return
	}

	completionTokens := -1
	// 经过测试，trt每个chunk都是一个单独的token，所以这么统计
	for data := range stream {
		if string(data) == "\n" {
			continue
		}
		completionTokens += 1
	}

	var outputTokensPerSecond float64 = 0
	timeSpentSeconds := float64(time.Now().Sub(start)) / float64(time.Second)
	speed := float64(completionTokens) / timeSpentSeconds
	if completionTokens == int(req.MaxTokens) {
		log.Infof("stream completion tokens: %d, time spent: %.1fs, speed: %.1ftokens/s",
			completionTokens, timeSpentSeconds, speed)
		outputTokensPerSecond = speed
	}
	atomic.AddInt32(req.SuccessCount, 1)
	req.Result <- param.Result{
		Prompt:                prompt,
		InputLen:              len(prompt),
		OutputTokens:          completionTokens,
		TimeSpent:             time.Now().Sub(start).Milliseconds(),
		OutputTokensPerSecond: outputTokensPerSecond,
	}
}
