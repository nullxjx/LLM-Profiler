package speed

import (
	"context"
	"fmt"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/config"
	"github.com/nullxjx/LLM-Profiler/infer/param"
	"github.com/nullxjx/LLM-Profiler/infer/triton"
	"github.com/nullxjx/LLM-Profiler/infer/vllm"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CalculateStreamSpeed 计算流式场景下的最大速度（每秒token数）
func CalculateStreamSpeed(cfg *config.Config) (float64, error) {
	prompts, err := common.ReadPrompts(cfg.InputTokens)
	if err != nil {
		log.Errorf("read inputs error: %v", err)
		return 0, err
	}
	// 从prompts中选前20条进行测试，去掉最小最大值后取均值
	speed := make([]float64, 0)
	for _, prompt := range prompts[:20] {
		start := time.Now()
		stream, err := sendStreamRequest(cfg, prompt)
		if err != nil {
			continue
		}
		completionTokens := countOutputTokens(cfg, stream)
		// 如果生成的token数比设定的MaxTokens小，说明模型提前停止了，这部分数据要去掉，否则会不准
		if completionTokens < int(cfg.MaxTokens) {
			log.Warnf("stream tokens %v is less than max tokens %v, skip", completionTokens, cfg.MaxTokens)
			continue
		}
		timeSpent := float64(time.Now().Sub(start)) / float64(time.Second)
		speedVal := float64(completionTokens) / timeSpent
		log.Infof("stream tokens: %v, time: %.1fs, speed: %.1f tokens/s", completionTokens, timeSpent, speedVal)
		speed = append(speed, speedVal)
		// 保证前后2条请求不会一起被处理
		time.Sleep(500 * time.Millisecond)
	}
	if len(speed) == 0 {
		return 0, errors.New("stream infer error")
	}
	avgSpeed := common.MeanWithoutMinMax(speed)
	return avgSpeed, nil
}

// countOutputTokens 计算 stream infer 生成的token数
func countOutputTokens(cfg *config.Config, stream <-chan []byte) int {
	if cfg.Backend == "vllm" {
		completionTokens := 0
		for data := range stream {
			tokens, _, err := vllm.GetVllmStreamTokens(string(data))
			if err != nil {
				continue
			}
			completionTokens = tokens
		}
		return completionTokens
	} else if cfg.Backend == "trt" {
		completionTokens := -1
		for data := range stream {
			if string(data) == "\n" {
				continue
			}
			completionTokens += 1
		}
		return completionTokens
	} else {
		log.Errorf("unsupported backend: %s", cfg.Backend)
	}
	return 0
}

func sendStreamRequest(cfg *config.Config, prompt string) (<-chan []byte, error) {
	if cfg.Backend == "vllm" {
		return vllm.StreamInferByVLLM(context.Background(), fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port),
			&param.InferParams{
				PromptList:   []string{prompt},
				ModelName:    cfg.Model.Name,
				ModelVersion: cfg.Model.Version,
				InferConfig: &param.InferConfig{
					StopWords:   cfg.StopWords,
					MaxTokens:   cfg.MaxTokens,
					Temperature: 1,
				},
			})
	} else if cfg.Backend == "trt" {
		return triton.StreamInferByTrt(context.Background(), fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port),
			&param.InferParams{
				PromptList:   []string{prompt},
				ModelName:    cfg.Model.Name,
				ModelVersion: cfg.Model.Version,
				InferConfig: &param.InferConfig{
					StopWords:   cfg.StopWords,
					MaxTokens:   cfg.MaxTokens,
					Temperature: 1,
				},
			})
	} else {
		panic(fmt.Sprintf("unsupported backend: %s", cfg.Backend))
	}
}
