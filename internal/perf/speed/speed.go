package speed

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/internal/infer/tgi"
	"github.com/nullxjx/llm_profiler/internal/infer/triton"
	"github.com/nullxjx/llm_profiler/internal/infer/type/backend"
	"github.com/nullxjx/llm_profiler/internal/infer/vllm"
	"github.com/nullxjx/llm_profiler/internal/utils"

	log "github.com/sirupsen/logrus"
)

// SpeedTest 单条速度测试
func SpeedTest(ip, modelName, backend string, port, promptLength int, temperature float32) {
	log.Infof("Single request speed test on model %v at %v:%v", modelName, ip, port)
	var speedValues []float64
	prompt := "The meaning of life is"
	tokens := 5
	if promptLength > 0 {
		inputs, err := utils.ReadPromptsWithTokens(promptLength)
		if err != nil {
			log.Errorf("read inputs error: %v", err)
			return
		}
		// 随机选择一个值
		rand.Seed(time.Now().UnixNano())      // 设置随机数种子
		randomIndex := rand.Intn(len(inputs)) // 生成一个随机索引
		prompt = inputs[randomIndex].Prompt
		tokens = inputs[randomIndex].Tokens
		log.Debugf("prompt index: %v", randomIndex)
	}
	log.Infof("prompt string len: %d, estimated tokens: %d", len(prompt), tokens)
	for m := 32; m <= 256; m += 32 {
		var successCnt = 0
		var totalTime float64 = 0
		var totalTokens = 0
		for i := 0; i < 10; i++ {
			start := time.Now()
			req := &param.InferParams{
				PromptList:   []string{prompt},
				ModelName:    modelName,
				ModelVersion: "1",
				Timeout:      100000,
				InferConfig: &param.InferConfig{
					StopWords:   []string{},
					MaxTokens:   uint32(m),
					Temperature: temperature,
					BeamWidth:   1,
					TopP:        1,
				},
			}
			outputTokens := sendRequest(ip, backend, port, req)
			elapsed := time.Since(start).Seconds() // 记录结束时间，计算经过的时间
			if outputTokens != 0 {
				successCnt += 1
				totalTokens += outputTokens
				totalTime += elapsed
			}
			time.Sleep(500 * time.Millisecond)
		}
		avgTime := totalTime / float64(successCnt)
		avgTokens := float64(totalTokens) / float64(successCnt)
		speedValues = append(speedValues, avgTokens/avgTime)
		log.Infof("output_tokens: %v, avg_time: %.1f s, tokens/s: %.1f", avgTokens, avgTime, avgTokens/avgTime)
	}
	log.Infof("speed for single request: %.1f tokens/s", utils.MeanWithoutMinMax(speedValues))
}

func sendRequest(ip, back string, port int, req *param.InferParams) int {
	metricHandlers := map[string]func(*param.InferParams, string) ([]param.InferResult, error){
		string(backend.VLLM): vllm.CompletionByVLLM,
		string(backend.TRT):  triton.InferTrt,
		string(backend.TGI):  tgi.InferTGI,
	}
	handler, ok := metricHandlers[strings.ToLower(back)]
	if !ok {
		panic(fmt.Sprintf("unsupported backend: %s", back))
	}
	res, err := handler(req, fmt.Sprintf("%s:%d", ip, port))
	if err != nil || len(res) == 0 {
		log.Errorf("send %v request error: %v", back, err)
		return 0
	}
	return res[0].OutputTokens
}
