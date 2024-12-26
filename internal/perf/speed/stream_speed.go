package speed

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nullxjx/llm_profiler/config"
	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/internal/infer/stream"
	"github.com/nullxjx/llm_profiler/internal/infer/triton"
	"github.com/nullxjx/llm_profiler/internal/infer/type/backend"
	"github.com/nullxjx/llm_profiler/internal/infer/vllm"
	"github.com/nullxjx/llm_profiler/internal/utils"

	log "github.com/sirupsen/logrus"
)

// StreamSpeed 流式场景下的指标
type StreamSpeed struct {
	TokensPerSecond float64
	FirstTokenTime  float64
}

// CalStreamSpeed 计算流式场景下的相关指标
func CalStreamSpeed(cfg *config.Config) (*StreamSpeed, error) {
	prompts, err := utils.ReadPrompts(cfg.InputTokens)
	if err != nil {
		return nil, fmt.Errorf("read inputs error: %v", err)
	}
	// 从prompts中选前20条进行测试，去掉最小最大值后取均值
	speedList := make([]float64, 0)
	firstTokenTimeList := make([]float64, 0)
	for _, prompt := range prompts[:20] {
		start := time.Now()
		s, err := sendStreamRequest(cfg, prompt)
		if err != nil {
			continue
		}
		metrics := countOutputTokens(cfg, s, start)
		// 如果生成的token数比设定的MaxTokens小，说明模型提前停止了，这部分数据要去掉，否则会不准
		if metrics.OutputTokens < int(cfg.MaxTokens) {
			log.Warnf("stream tokens %v is less than max tokens %v, skip", metrics.OutputTokens, cfg.MaxTokens)
			continue
		}
		timeSpent := float64(time.Now().Sub(start)) / float64(time.Second)
		speedVal := float64(metrics.OutputTokens) / timeSpent
		speedList = append(speedList, speedVal)
		log.Debugf("stream output tokens: %v, time: %.1fs, speed: %.1f tokens/s, firstToken: %.1f ms",
			metrics.OutputTokens, timeSpent, speedVal, metrics.FirstTokenTime)

		if metrics.FirstTokenTime > 0 {
			firstTokenTimeList = append(firstTokenTimeList, metrics.FirstTokenTime)
		}
		// 保证前后2条请求不会一起被处理
		time.Sleep(500 * time.Millisecond)
	}
	return &StreamSpeed{
		TokensPerSecond: utils.MeanWithoutMinMax(speedList),
		FirstTokenTime:  utils.MeanWithoutMinMax(firstTokenTimeList),
	}, nil
}

// countOutputTokens 计算 stream infer 生成的token数
func countOutputTokens(cfg *config.Config, s <-chan []byte, startTime time.Time) *stream.StreamMetrics {
	metricHandlers := map[string]func(<-chan []byte, time.Time) *stream.StreamMetrics{
		string(backend.VLLM): stream.CalVllmMetrics,
		string(backend.TRT):  stream.CalTrtMetrics,
	}
	handler, ok := metricHandlers[strings.ToLower(cfg.Backend)]
	if !ok {
		panic(fmt.Sprintf("unsupported backend: %s", cfg.Backend))
	}
	return handler(s, startTime)
}

func sendStreamRequest(cfg *config.Config, prompt string) (<-chan []byte, error) {
	backendHandlers := map[string]func(ctx context.Context, url string, params *param.InferParams) (
		<-chan []byte, error){
		string(backend.VLLM): vllm.StreamCompletionByVLLM,
		string(backend.TRT):  triton.StreamInferByTrt,
	}
	handler, ok := backendHandlers[strings.ToLower(cfg.Backend)]
	if !ok {
		panic(fmt.Sprintf("unsupported backend: %s", cfg.Backend))
	}
	return handler(context.Background(),
		fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port),
		&param.InferParams{
			PromptList:   []string{prompt},
			ModelName:    cfg.Model.Name,
			ModelVersion: cfg.Model.Version,
			InferConfig: &param.InferConfig{
				StopWords:   cfg.StopWords,
				MaxTokens:   cfg.MaxTokens,
				Temperature: cfg.Temperature,
			},
		})
}
