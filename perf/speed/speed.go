package speed

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/infer/param"
	"github.com/nullxjx/LLM-Profiler/infer/tgi"
	"github.com/nullxjx/LLM-Profiler/infer/triton"
	"github.com/nullxjx/LLM-Profiler/infer/vllm"

	log "github.com/sirupsen/logrus"
)

// SpeedTest 单条速度测试
func SpeedTest(ip, modelName, backend string, port, promptLength int) {
	log.Infof("Single request speed test on model %v at %v:%v", modelName, ip, port)
	var speedValues []float64
	prompt := "The meaning of life is"
	tokens := 5
	if promptLength > 0 {
		inputs, err := common.ReadPromptsWithTokens(promptLength)
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
					Temperature: 1,
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
			time.Sleep(1 * time.Second)
		}
		avgTime := totalTime / float64(successCnt)
		avgTokens := float64(totalTokens) / float64(successCnt)
		speedValues = append(speedValues, avgTokens/avgTime)
		log.Infof("output_tokens: %v, avg_time: %.3fs, tokens/s: %.3f", avgTokens, avgTime, avgTokens/avgTime)
	}
	log.Infof("speed for single request: %.3f tokens/s", common.MeanWithoutMinMax(speedValues))
}

func sendRequest(ip, backend string, port int, req *param.InferParams) int {
	serviceURL := fmt.Sprintf("%s:%d", ip, port)
	outputTokens := 0
	if backend == "vllm" {
		result, err := vllm.InferVLLM(req, serviceURL)
		if err == nil {
			outputTokens = result[0].OutputTokens
		}
	} else if backend == "tgi" {
		result, err := tgi.InferTGI(req, serviceURL)
		if err == nil {
			outputTokens = result.Details.GeneratedTokens
		}
	} else if backend == "triton-vllm" {
		result, err := triton.InferVllmInTriton(req, serviceURL)
		if err == nil {
			outputTokens = result[0].OutputTokens
		}
	} else if backend == "triton-trt" {
		result, err := triton.InferTrt(req, serviceURL)
		if err == nil {
			outputTokens = result[0].OutputTokens
		}
	} else {
		panic(fmt.Sprintf("unsupported backend: %s", backend))
	}
	return outputTokens
}
