package infer

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/internal/infer/stream"
	"github.com/nullxjx/llm_profiler/internal/infer/tgi"
	"github.com/nullxjx/llm_profiler/internal/infer/triton"
	"github.com/nullxjx/llm_profiler/internal/infer/vllm"

	log "github.com/sirupsen/logrus"
)

// SendVllmRequest 发送 vllm 请求
func SendVllmRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	atomic.AddInt32(&req.Counter.Total, 1)
	cfg := req.Config
	result, err := vllm.CompletionByVLLM(&param.InferParams{
		PromptList:   []string{req.Prompt},
		ModelName:    cfg.Model.Name,
		ModelVersion: cfg.Model.Version,
		Timeout:      cfg.RequestTimeout,
		InferConfig: &param.InferConfig{
			StopWords:   cfg.StopWords,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
		},
	}, fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port))
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(&req.Counter.Failed, 1)
		return
	}

	atomic.AddInt32(&req.Counter.Success, 1)
	req.Result <- param.Result{
		Prompt:       req.Prompt,
		InputLen:     len(req.Prompt),
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
	atomic.AddInt32(&req.Counter.Total, 1)
	cfg := req.Config
	start := time.Now()
	s, err := vllm.StreamCompletionByVLLM(context.Background(), fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port),
		&param.InferParams{
			PromptList:   []string{req.Prompt},
			ModelName:    cfg.Model.Name,
			ModelVersion: cfg.Model.Version,
			Timeout:      cfg.RequestTimeout,
			InferConfig: &param.InferConfig{
				StopWords:   cfg.StopWords,
				MaxTokens:   cfg.MaxTokens,
				Temperature: cfg.Temperature,
			},
		})
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(&req.Counter.Failed, 1)
		return
	}
	metrics := stream.CalVllmMetrics(s, start)
	if metrics.OutputTokens >= int(req.Config.MaxTokens) {
		log.Debugf("stream output tokens: %d, time spent: %v s, speed: %.1f tokens/s, first_token: %.1f ms",
			metrics.OutputTokens, metrics.TimeSpentSeconds, metrics.TokensPerSec, metrics.FirstTokenTime)
	}
	atomic.AddInt32(&req.Counter.Success, 1)
	req.Result <- param.Result{
		Prompt:          req.Prompt,
		InputLen:        len(req.Prompt),
		OutputTokens:    metrics.OutputTokens,
		TimeSpent:       time.Now().Sub(start).Milliseconds(),
		TokensPerSecond: metrics.TokensPerSec,
		FirstTokenTime:  metrics.FirstTokenTime,
	}
}

// SendTgiRequest 发送 Tgi 请求
func SendTgiRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	atomic.AddInt32(&req.Counter.Total, 1)
	cfg := req.Config
	start := time.Now()
	result, err := tgi.InferTGI(&param.InferParams{
		PromptList:   []string{req.Prompt},
		ModelName:    cfg.Model.Name,
		ModelVersion: cfg.Model.Version,
		Timeout:      cfg.RequestTimeout,
		InferConfig: &param.InferConfig{
			StopWords:   cfg.StopWords,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
		},
	}, fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port))
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(&req.Counter.Failed, 1)
		return
	}
	atomic.AddInt32(&req.Counter.Success, 1)
	req.Result <- param.Result{
		Prompt:       req.Prompt,
		InputLen:     len(req.Prompt),
		InputTokens:  result[0].InputTokens,
		Output:       result[0].Result,
		OutputLen:    len(result[0].Result),
		OutputTokens: result[0].OutputTokens,
		TimeSpent:    time.Now().Sub(start).Milliseconds(),
	}
}

// SendTrtRequest 发送 TensorRT-LLM 请求
func SendTrtRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	atomic.AddInt32(&req.Counter.Total, 1)
	cfg := req.Config
	result, err := triton.InferTrt(&param.InferParams{
		PromptList:   []string{req.Prompt},
		ModelName:    cfg.Model.Name,
		ModelVersion: cfg.Model.Version,
		Timeout:      cfg.RequestTimeout,
		InferConfig: &param.InferConfig{
			StopWords:   cfg.StopWords,
			MaxTokens:   cfg.MaxTokens,
			Temperature: cfg.Temperature,
		},
	}, fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port))
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(&req.Counter.Failed, 1)
		return
	}
	atomic.AddInt32(&req.Counter.Success, 1)
	req.Result <- param.Result{
		Prompt:       req.Prompt,
		InputLen:     len(req.Prompt),
		InputTokens:  result[0].InputTokens,
		Output:       result[0].Result,
		OutputLen:    len(result[0].Result),
		OutputTokens: int(cfg.MaxTokens),
		TimeSpent:    result[0].TimeSpent,
	}
}

// SendTrtStreamRequest 发送 TensorRT-LLM 流式请求
func SendTrtStreamRequest(req *param.RequestParam) {
	defer req.Wg.Done()
	atomic.AddInt32(&req.Counter.Total, 1)
	cfg := req.Config

	start := time.Now()
	s, err := triton.StreamInferByTrt(context.Background(), fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port),
		&param.InferParams{
			PromptList:   []string{req.Prompt},
			ModelName:    cfg.Model.Name,
			ModelVersion: cfg.Model.Version,
			Timeout:      cfg.RequestTimeout,
			InferConfig: &param.InferConfig{
				StopWords:   cfg.StopWords,
				MaxTokens:   cfg.MaxTokens,
				Temperature: cfg.Temperature,
			},
		})
	if err != nil {
		log.Errorf("😭😭😭 infer error: %v", err)
		atomic.AddInt32(&req.Counter.Failed, 1)
		return
	}

	metrics := stream.CalTrtMetrics(s, start)
	if metrics.OutputTokens >= int(cfg.MaxTokens) {
		log.Debugf("stream output tokens: %d, time spent: %.1f s, speed: %.1f tokens/s, first_token: %.1f ms",
			metrics.OutputTokens, metrics.TimeSpentSeconds, metrics.TokensPerSec, metrics.FirstTokenTime)
	}
	atomic.AddInt32(&req.Counter.Success, 1)
	req.Result <- param.Result{
		Prompt:          req.Prompt,
		InputLen:        len(req.Prompt),
		OutputTokens:    metrics.OutputTokens,
		TimeSpent:       time.Now().Sub(start).Milliseconds(),
		TokensPerSecond: metrics.TokensPerSec,
		FirstTokenTime:  metrics.FirstTokenTime,
	}
}
