package throughput

import (
	"fmt"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/infer/param"

	"github.com/montanaflynn/stats"
	log "github.com/sirupsen/logrus"
)

type StatisticsSummary struct {
	Concurrency int `json:"concurrency"` // 并发度，即给定时间内发送的请求个数

	Success int32 `json:"success"` // 请求成功数
	Fail    int32 `json:"fail"`    // 请求失败数
	Total   int32 `json:"total"`   // 请求总数

	AvgTimeServerSide float64 `json:"avgTimeServerSide"` // 客户端平均耗时
	AvgTimeClientSide float64 `json:"avgTimeClientSide"` // 总耗时/请求数，描述了服务端观察到的请求平均耗时

	AvgInputLen     float64 `json:"avgInputLen"`     // 平均输入字符数
	AvgOutputLen    float64 `json:"avgOutputLen"`    // 平均输出字符数
	AvgInputTokens  float64 `json:"avgInputTokens"`  // 平均输入token数
	AvgOutputTokens float64 `json:"avgOutputTokens"` // 平均输出token数

	InputTokensPerSecond        float64 `json:"inputTokensPerSecond"`        // 平均每秒输入token
	OutputTokensPerSecond       float64 `json:"outputTokensPerSecond"`       // 服务端角度统计到的平均每秒输出token
	ClientOutputTokensPerSecond float64 `json:"clientOutputTokensPerSecond"` // 客户端角度统计到的平均每秒输出token

	RequestPerSecond float64        `json:"requestPerSecond"` // 平均每秒处理的请求数
	TimeSpentSummary map[string]int `yaml:"timeSpentSummary"` // 不同时间内的请求数量统计

	StartTime string `json:"startTime"` // 本轮次开始时间
	EndTime   string `json:"endTime"`   // 本轮次结束时间

	P99 float64 `yaml:"p99"` // 毫秒
	P90 float64 `yaml:"p90"` // 毫秒
	P80 float64 `yaml:"p80"` // 毫秒
}

type StatisticsParam struct {
	Concurrency    int                 // 并发度，即给定时间内发送的请求个数
	Duration       float64             // 请求持续时间
	Results        <-chan param.Result // 该轮次调用结果记录
	TotalCount     *int32              // 总请求个数
	SuccessCount   *int32              // 成功请求个数
	FailedCount    *int32              // 失败请求个数
	TimeThresholds []int64             // 请求时间阈值
	SaveDir        string              // 保存路径
	StartTime      string              // 开始时间
	EndTime        string              // 结束时间
}

var statistics = make(map[int]*StatisticsSummary) // 记录了每轮次的统计结果

// countMetrics 统计一轮的指标
func countMetrics(s *StatisticsParam) {
	var totalTime int64
	var resultList []param.Result
	var inputLen int     // 输入字符串的长度
	var inputTokens int  // 输入token数目
	var outputLen int    // 输出字符串的长度
	var outputTokens int //输出token数目
	var timeSpentList []int64
	var clientOutputTokensPerSecond []float64
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
		if result.OutputTokensPerSecond != 0 {
			clientOutputTokensPerSecond = append(clientOutputTokensPerSecond, result.OutputTokensPerSecond)
		}
	}

	var avgTimeServerSide float64 = 0
	var avgTimeClientSide float64 = 0
	if *s.SuccessCount > 0 {
		avgTimeServerSide = s.Duration * 1000 / float64(*s.SuccessCount)
		avgTimeClientSide = float64(totalTime) / float64(*s.SuccessCount)
	}
	nowStr := time.Now().Format("2006-01-02-15-04-05")
	common.Save2Json(resultList, fmt.Sprintf("%s/results_%s_concurrency_%d.json", s.SaveDir, nowStr, s.Concurrency))

	// 将 int64 数据转换为 float64 类型
	floatData := make(stats.Float64Data, len(timeSpentList))
	for i, v := range timeSpentList {
		floatData[i] = float64(v)
	}
	// 计算 P99, P90, 和 P80
	p99, _ := stats.Percentile(floatData, 99)
	p90, _ := stats.Percentile(floatData, 90)
	p80, _ := stats.Percentile(floatData, 80)

	avgClientOutputTokensPerSecond := common.MeanWithoutMinMax(clientOutputTokensPerSecond)
	statistics[s.Concurrency] = &StatisticsSummary{
		Concurrency: s.Concurrency,

		Success: *s.SuccessCount,
		Fail:    *s.FailedCount,
		Total:   *s.TotalCount,

		AvgTimeServerSide: avgTimeServerSide,
		AvgTimeClientSide: avgTimeClientSide,

		AvgInputTokens:  float64(inputTokens) / float64(*s.SuccessCount),
		AvgOutputTokens: float64(outputTokens) / float64(*s.SuccessCount),
		AvgInputLen:     float64(inputLen) / float64(*s.SuccessCount),
		AvgOutputLen:    float64(outputLen) / float64(*s.SuccessCount),

		InputTokensPerSecond:        float64(inputTokens) / s.Duration,
		OutputTokensPerSecond:       float64(outputTokens) / s.Duration,
		ClientOutputTokensPerSecond: avgClientOutputTokensPerSecond,

		RequestPerSecond: float64(*s.SuccessCount) / s.Duration,
		TimeSpentSummary: timeSpentSummary,

		StartTime: s.StartTime,
		EndTime:   s.EndTime,

		P99: p99,
		P90: p90,
		P80: p80,
	}
}

func clearCache() {
	log.Infof("Clearing statistics cache...")
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
			maxInputTokensPerSecond = value.InputTokensPerSecond
			maxOutputTokensPerSecond = value.OutputTokensPerSecond
		}
	}

	return maxRequestPerSecond, maxInputTokensPerSecond, maxOutputTokensPerSecond
}
