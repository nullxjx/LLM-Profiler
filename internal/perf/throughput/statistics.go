package throughput

import (
	"fmt"
	"time"

	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/internal/utils"

	"github.com/montanaflynn/stats"
	log "github.com/sirupsen/logrus"
)

// StatisticsSummary 每一轮的统计结果
type StatisticsSummary struct {
	Concurrency                 int            `json:"concurrency"`                     // 并发度，即给定时间内发送的请求个数
	Success                     int32          `json:"success"`                         // 请求成功数
	Fail                        int32          `json:"fail"`                            // 请求失败数
	Total                       int32          `json:"total"`                           // 请求总数
	AvgTimeServerSide           float64        `json:"avg_time_server_side"`            // 客户端平均耗时
	AvgTimeClientSide           float64        `json:"avg_time_client_side"`            // 总耗时/请求数，描述了服务端观察到的请求平均耗时
	AvgInputLen                 float64        `json:"avg_input_len"`                   // 平均输入字符数
	AvgOutputLen                float64        `json:"avg_output_len"`                  // 平均输出字符数
	AvgInputTokens              float64        `json:"avg_input_tokens"`                // 平均输入token数
	AvgOutputTokens             float64        `json:"avg_output_tokens"`               // 平均输出token数
	ServerInputTokensPerSecond  float64        `json:"server_input_tokens_per_second"`  // 服务端平均每秒输入token
	ServerOutputTokensPerSecond float64        `json:"server_output_tokens_per_second"` // 服务端平均每秒输出token
	ClientOutputTokensPerSecond float64        `json:"client_output_tokens_per_second"` // 客户端平均每秒输出token，仅在流式场景下存在
	FirstTokenTime              float64        `json:"first_token_time"`                // 首token时间，仅在流式场景下存在
	RequestPerSecond            float64        `json:"request_per_second"`              // 平均每秒处理的请求数
	TimeSpentSummary            map[string]int `yaml:"time_spent_summary"`              // 不同时间内的请求数量统计
	StartTime                   string         `json:"start_time"`                      // 本轮次开始时间
	EndTime                     string         `json:"end_time"`                        // 本轮次结束时间
	P99                         float64        `yaml:"p99"`                             // 毫秒
	P90                         float64        `yaml:"p90"`                             // 毫秒
	P80                         float64        `yaml:"p80"`                             // 毫秒
}

type StatisticsParam struct {
	Concurrency    int                 // 并发度，即给定时间内发送的请求个数
	Duration       float64             // 请求持续时间
	Results        <-chan param.Result // 该轮次调用结果记录
	TotalCount     int32               // 总请求个数
	SuccessCount   int32               // 成功请求个数
	FailedCount    int32               // 失败请求个数
	TimeThresholds []int64             // 请求时间阈值
	SaveDir        string              // 保存路径
	StartTime      string              // 开始时间
	EndTime        string              // 结束时间
}

var statistics = make(map[int]*StatisticsSummary) // 记录了每轮次的统计结果

// calMetrics 统计一轮的指标
func calMetrics(s *StatisticsParam) {
	var totalTime int64
	var resultList []param.Result
	var inputLen int     // 输入字符串的长度
	var inputTokens int  // 输入token数目
	var outputLen int    // 输出字符串的长度
	var outputTokens int //输出token数目
	var timeSpentList []int64
	var tokensPerSecond []float64
	var firstTokenTime []float64
	timeSpentSummary := make(map[string]int)
	for result := range s.Results {
		resultList = append(resultList, result)

		inputLen += result.InputLen
		inputTokens += result.InputTokens
		outputLen += result.OutputLen
		outputTokens += result.OutputTokens

		totalTime += result.TimeSpent
		timeSpentList = append(timeSpentList, result.TimeSpent)
		for _, timeThreshold := range s.TimeThresholds {
			if result.TimeSpent <= timeThreshold {
				key := fmt.Sprintf("less than %d ms", timeThreshold)
				timeSpentSummary[key]++
			}
		}
		if result.TokensPerSecond != 0 {
			tokensPerSecond = append(tokensPerSecond, result.TokensPerSecond)
		}
		if result.FirstTokenTime != 0 {
			firstTokenTime = append(firstTokenTime, result.FirstTokenTime)
		}
	}

	var avgTimeServerSide float64 = 0
	var avgTimeClientSide float64 = 0
	if s.SuccessCount > 0 {
		avgTimeServerSide = s.Duration * 1000 / float64(s.SuccessCount)
		avgTimeClientSide = float64(totalTime) / float64(s.SuccessCount)
	}
	nowStr := time.Now().Format(utils.TimeFormat)
	utils.Save2Json(resultList, fmt.Sprintf("%s/results_%s_concurrency_%d.json", s.SaveDir, nowStr, s.Concurrency))

	// 将 int64 数据转换为 float64 类型
	floatData := make(stats.Float64Data, len(timeSpentList))
	for i, v := range timeSpentList {
		floatData[i] = float64(v)
	}
	// 计算 P99, P90, 和 P80
	p99, _ := stats.Percentile(floatData, 99)
	p90, _ := stats.Percentile(floatData, 90)
	p80, _ := stats.Percentile(floatData, 80)
	statistics[s.Concurrency] = &StatisticsSummary{
		Concurrency:                 s.Concurrency,
		Success:                     s.SuccessCount,
		Fail:                        s.FailedCount,
		Total:                       s.TotalCount,
		AvgTimeServerSide:           avgTimeServerSide,
		AvgTimeClientSide:           avgTimeClientSide,
		AvgInputTokens:              float64(inputTokens) / float64(s.SuccessCount),
		AvgOutputTokens:             float64(outputTokens) / float64(s.SuccessCount),
		AvgInputLen:                 float64(inputLen) / float64(s.SuccessCount),
		AvgOutputLen:                float64(outputLen) / float64(s.SuccessCount),
		ServerInputTokensPerSecond:  float64(inputTokens) / s.Duration,
		ServerOutputTokensPerSecond: float64(outputTokens) / s.Duration,
		ClientOutputTokensPerSecond: utils.MeanWithoutMinMax(tokensPerSecond), // 仅在流式场景下存在
		FirstTokenTime:              utils.MeanWithoutMinMax(firstTokenTime),  // 仅在流式场景下存在
		RequestPerSecond:            float64(s.SuccessCount) / s.Duration,
		TimeSpentSummary:            timeSpentSummary,
		StartTime:                   s.StartTime,
		EndTime:                     s.EndTime,
		P99:                         p99,
		P90:                         p90,
		P80:                         p80,
	}
}

func clearCache() {
	log.Debugf("Clearing statistics cache...")
	for k := range statistics {
		delete(statistics, k)
	}
}

func GetMaxThroughput() (float64, float64, float64) {
	var maxInputTokensPerSecond float64 = 0
	var maxOutputTokensPerSecond float64 = 0
	var maxRequestPerSecond float64 = 0

	for _, value := range statistics {
		if maxRequestPerSecond < value.RequestPerSecond {
			maxRequestPerSecond = value.RequestPerSecond
			maxInputTokensPerSecond = value.ServerInputTokensPerSecond
			maxOutputTokensPerSecond = value.ServerOutputTokensPerSecond
		}
	}

	return maxRequestPerSecond, maxInputTokensPerSecond, maxOutputTokensPerSecond
}
