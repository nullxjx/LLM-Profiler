package stream

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
)

// StreamMetrics 流式输出相关的指标
type StreamMetrics struct {
	OutputTokens     int
	TokensPerSec     float64
	FirstTokenTime   float64
	TimeSpentSeconds float64
}

// CalVllmMetrics 计算 vllm stream infer 相关指标
func CalVllmMetrics(stream <-chan []byte, startTime time.Time) *StreamMetrics {
	var once sync.Once
	var firstTokenTime float64 // 单位毫秒

	completionTokens := 0
	count := -1
	for data := range stream {
		once.Do(func() {
			firstTokenTime = float64(time.Now().Sub(startTime).Milliseconds())
		})
		if string(data) == "\n" || strings.Contains(string(data), "DONE") {
			continue
		}
		count += 1

		tokens, _, err := getVllmChatStreamTokens(string(data))
		if err != nil {
			continue
		}
		if tokens > 1 {
			completionTokens = tokens
		}
	}
	timeSpentSeconds := float64(time.Now().Sub(startTime)) / float64(time.Second)
	if completionTokens > 0 {
		return &StreamMetrics{
			OutputTokens:     completionTokens,
			FirstTokenTime:   firstTokenTime,
			TokensPerSec:     float64(completionTokens) / timeSpentSeconds,
			TimeSpentSeconds: timeSpentSeconds,
		}
	}
	// 有些vllm版本的接口不会返回这个统计信息，那就返回手动统计的token数量
	return &StreamMetrics{
		OutputTokens:     count,
		FirstTokenTime:   firstTokenTime,
		TokensPerSec:     float64(count) / timeSpentSeconds,
		TimeSpentSeconds: timeSpentSeconds,
	}
}

// CalTrtMetrics 计算 trt stream infer 相关指标
func CalTrtMetrics(stream <-chan []byte, startTime time.Time) *StreamMetrics {
	var once sync.Once
	var firstTokenTime float64 // 单位毫秒

	completionTokens := -1
	for data := range stream {
		once.Do(func() {
			firstTokenTime = float64(time.Now().Sub(startTime).Milliseconds())
		})
		if string(data) == "\n" {
			continue
		}
		completionTokens += 1
	}
	timeSpentSeconds := float64(time.Now().Sub(startTime)) / float64(time.Second)
	return &StreamMetrics{
		OutputTokens:     completionTokens,
		FirstTokenTime:   firstTokenTime,
		TokensPerSec:     float64(completionTokens) / timeSpentSeconds,
		TimeSpentSeconds: timeSpentSeconds,
	}
}

// getVllmChatStreamTokens 获取vllm流式对话的token数量
func getVllmChatStreamTokens(chunk string) (int, int, error) {
	// 使用正则表达式匹配 "data:" 开头的字符串
	re := regexp.MustCompile(`^data:\s*(\{.*})`)
	matches := re.FindStringSubmatch(chunk)

	if len(matches) != 2 {
		return 0, 0, errors.New("invalid input format")
	}

	jsonData := matches[1]
	var data openai.ChatCompletionResponse
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return 0, 0, errors.New("invalid input format")
	}
	return data.Usage.CompletionTokens, data.Usage.PromptTokens, nil
}
