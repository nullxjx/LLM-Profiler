package throughput

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/config"
	"github.com/nullxjx/LLM-Profiler/infer"
	"github.com/nullxjx/LLM-Profiler/infer/param"

	log "github.com/sirupsen/logrus"
)

// ThroughputTest 吞吐量测试
func ThroughputTest(cfg *config.Config) (string, string) {
	clearCache()

	prompts, err := common.ReadPrompts(cfg.InputTokens)
	if err != nil {
		log.Errorf("read inputs error: %v", err)
		return "", ""
	}
	if cfg.StartConcurrency > cfg.EndConcurrency {
		log.Errorf("StartConcurrency > EndConcurrency")
		return "", ""
	}

	// 逐步增加并发度，测试吞吐量
	for concurrency := cfg.StartConcurrency; concurrency <= cfg.EndConcurrency; concurrency += cfg.Increment {
		log.Infof("🙏🙏🙏 start testing at concurrency %v", concurrency)
		if step(cfg, prompts, concurrency) {
			break
		}

		// 等待上一轮完全结束，为了避免这轮请求对下一轮造成影响 todo @nullxjx 需要改进，如何更精准判断上一轮已经结束
		time.Sleep(30 * time.Second)
	}

	return finish(cfg)
}

// step 进行一轮测试，返回true，则停止下一轮测试，返回false，则继续下一轮测试
func step(cfg *config.Config, prompts []string, concurrency int) bool {
	// 设置测试持续时间
	duration := time.Duration(cfg.Duration) * time.Minute

	wg := &sync.WaitGroup{}
	var mu sync.Mutex
	ticker := time.NewTicker(duration / time.Duration(concurrency))
	results := make(chan param.Result, concurrency)
	var totalCount int32 = 0
	var successCount int32 = 0
	var failedCount int32 = 0
	var inputIndex = 0

	startTime := time.Now()
	for start := time.Now(); time.Since(start) < duration; {
		select {
		case <-ticker.C:
			wg.Add(1)
			go sendRequest(&param.RequestParam{
				ServiceURL:   fmt.Sprintf("%s:%d", cfg.ServerIp, cfg.Port),
				ModelName:    cfg.Model.Name,
				ModelVersion: cfg.Model.Version,
				Wg:           wg,
				Prompts:      prompts,
				Result:       results,
				SuccessCount: &successCount,
				FailedCount:  &failedCount,
				TotalCount:   &totalCount,
				InputIndex:   inputIndex,
				Timeout:      cfg.RequestTimeout,
				StopWords:    cfg.StopWords,
				MaxTokens:    cfg.MaxTokens,
			}, cfg)
		default:
			time.Sleep(10 * time.Millisecond) // Avoid busy waiting
		}
		// 使用互斥锁保护对 inputIndex 的访问和更新
		mu.Lock()
		inputIndex = (inputIndex + 1) % len(prompts)
		mu.Unlock()
	}
	log.Infof("waiting for all goroutines to finish...")
	wg.Wait() // 阻塞，直到 WaitGroup 的计数器变为 0
	close(results)
	//log.Infof("all goroutines have finished")

	endTime := time.Now()
	startTimeStr := startTime.Format("2006-01-02 15:04:05")
	endTimeStr := endTime.Format("2006-01-02 15:04:05")
	timeSpent := float64(endTime.Sub(startTime)) / float64(time.Second)
	countMetrics(&StatisticsParam{
		Concurrency:    concurrency,
		Duration:       timeSpent, // 单位是秒
		Results:        results,
		TotalCount:     &totalCount,
		SuccessCount:   &successCount,
		FailedCount:    &failedCount,
		TimeThresholds: cfg.TimeThresholds,
		SaveDir:        cfg.SaveDir,
		StartTime:      startTimeStr,
		EndTime:        endTimeStr,
	})
	metric := statistics[concurrency]
	log.Debugf("[time: %.3fs, total: %v, success: %v, fail: %v] "+
		"| Server: [ %.3f tokens/s, %.3f req/s ] | Client: %.3f tokens/s "+
		"| Stream thresholds: %v%% | MaxStreamSpeed: %.1f tokens/s "+
		"| Prompt length: %v",
		timeSpent, metric.Total, metric.Success, metric.Fail,
		metric.OutputTokensPerSecond, metric.RequestPerSecond, metric.ClientOutputTokensPerSecond,
		cfg.StreamSpeedThresholds, cfg.MaxStreamSpeed, cfg.InputTokens)
	return stopCheck(cfg, concurrency)
}

func sendRequest(param *param.RequestParam, cfg *config.Config) {
	if cfg.Stream {
		go sendStreamRequest(param, cfg)
		return
	}

	if cfg.Backend == "vllm" {
		go infer.SendVllmRequest(param)
	} else if cfg.Backend == "tgi" {
		go infer.SendTgiRequest(param)
	} else if cfg.Backend == "triton-vllm" {
		go infer.SendTritonVllmRequest(param)
	} else if cfg.Backend == "trt" {
		go infer.SendTrtRequest(param)
	} else {
		panic(fmt.Sprintf("unsupported backend: %s", cfg.Backend))
	}
}

func sendStreamRequest(param *param.RequestParam, cfg *config.Config) {
	if cfg.Backend == "vllm" {
		go infer.SendVllmStreamRequest(param)
	} else if cfg.Backend == "trt" {
		go infer.SendTrtStreamRequest(param)
	} else {
		panic(fmt.Sprintf("unsupported stream on backend: %s", cfg.Backend))
	}
}

func saveResult(cfg *config.Config) {
	var valueList []*StatisticsSummary
	for _, value := range statistics {
		valueList = append(valueList, value)
	}
	// 按照Concurrency对结果排序
	sort.Slice(valueList, func(i, j int) bool {
		return valueList[i].Concurrency < valueList[j].Concurrency
	})

	common.Save2Json(valueList, fmt.Sprintf("%s/statistics_%s.json", cfg.SaveDir, time.Now().Format("2006-01-02-15-04-05")))
}

func finish(cfg *config.Config) (string, string) {
	// 有一些配置项比较敏感，不要存到结果中去，这里手动过滤掉这些配置
	log.Infof("saving config file...")

	cfg_ := *cfg

	common.Save2Json(cfg_, fmt.Sprintf("%s/config_%s.json", cfg.SaveDir, time.Now().Format("2006-01-02-15-04-05")))

	// 只保留最新一个统计文件
	common.KeepFinalResult(cfg.SaveDir)

	if !cfg.Save2Cos {
		log.Debugf("See result in %v", cfg.SaveDir)
		return "", ""
	}

	log.Debugf("saving all files to cos...")
	downloadUrl, dstDir, err := common.SaveFilesToCos(cfg)
	if err != nil {
		log.Errorf("❗️❗️❗ upload to cos failed")
		log.Debugf("🥳🤩🥰 Done!!!")
		return "", ""
	}

	if cfg.SendMsg {
		log.Debugf("download statistics result via 👉 %s 👈", downloadUrl)
		common.SendMsg(cfg, downloadUrl, dstDir)
	}
	log.Debugf("🥳🤩🥰 Done!!!")
	return downloadUrl, dstDir
}
