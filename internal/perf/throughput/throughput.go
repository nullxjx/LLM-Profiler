package throughput

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nullxjx/llm_profiler/config"
	"github.com/nullxjx/llm_profiler/internal/infer"
	"github.com/nullxjx/llm_profiler/internal/infer/param"
	"github.com/nullxjx/llm_profiler/internal/infer/type/backend"
	"github.com/nullxjx/llm_profiler/internal/utils"
	"github.com/nullxjx/llm_profiler/pkg/store/cos"

	log "github.com/sirupsen/logrus"
)

// StartTest 开始吞吐量测试
func StartTest(cfg *config.Config) (string, string) {
	clearCache()

	prompts, err := utils.ReadPrompts(cfg.InputTokens)
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
		log.Infof("🙏🙏🙏 start testing at concurrency %v, duration: %v min", concurrency, cfg.Duration)
		step(cfg, prompts, concurrency)
		if stop(cfg, concurrency) {
			break
		}
		// 等待上一轮完全结束，为了避免这轮还未完成的请求对下一轮造成影响
		// todo(@nullxjx) 需要改进，如何更精准判断上一轮已经结束
		time.Sleep(30 * time.Second)
	}

	return finish(cfg)
}

// step 进行一轮测试
func step(cfg *config.Config, prompts []string, concurrency int) {
	wg := &sync.WaitGroup{}
	var mu sync.Mutex
	results := make(chan param.Result, concurrency)
	counter := &param.Counter{
		Success: 0,
		Failed:  0,
		Total:   0,
	}
	var inputIndex = 0
	startTime := time.Now()
	duration := time.Duration(cfg.Duration) * time.Minute
	ticker := time.NewTicker(duration / time.Duration(concurrency))
	for start := time.Now(); time.Since(start) < duration; {
		select {
		case <-ticker.C:
			wg.Add(1)
			go sendRequest(&param.RequestParam{
				Wg:      wg,
				Prompt:  prompts[inputIndex],
				Result:  results,
				Counter: counter,
				Config:  cfg,
			})
		default:
			time.Sleep(10 * time.Millisecond) // Avoid busy waiting
		}
		// 使用互斥锁保护对 inputIndex 的访问和更新
		mu.Lock()
		inputIndex = (inputIndex + 1) % len(prompts)
		mu.Unlock()
	}
	log.Debugf("Waiting for all goroutines to finish...")
	wg.Wait() // 阻塞，直到 WaitGroup 的计数器变为 0
	close(results)

	endTime := time.Now()
	timeSpent := float64(endTime.Sub(startTime)) / float64(time.Second)
	calMetrics(&StatisticsParam{
		Concurrency:    concurrency,
		Duration:       timeSpent, // 单位是秒
		Results:        results,
		TotalCount:     counter.Total,
		SuccessCount:   counter.Success,
		FailedCount:    counter.Failed,
		TimeThresholds: cfg.TimeThresholds,
		SaveDir:        cfg.SaveDir,
		StartTime:      startTime.Format(utils.TimeFormat),
		EndTime:        endTime.Format(utils.TimeFormat),
	})
	metric := statistics[concurrency]
	if cfg.Stream {
		log.Infof("[time: %.1f s, total: %v, success: %v, fail: %v] "+
			"| Server: [ %.1f tokens/s, %.1f req/s ] | Client: %.1f tokens/s "+
			"| Stream thresholds: %v%% | MaxStreamSpeed: %.1f tokens/s, FirstToken: %.1f ms "+
			"| Prompt length: %v",
			timeSpent, metric.Total, metric.Success, metric.Fail,
			metric.ServerOutputTokensPerSecond, metric.RequestPerSecond, metric.ClientOutputTokensPerSecond,
			cfg.StreamThresholds, cfg.MaxStreamSpeed, metric.FirstTokenTime, cfg.InputTokens)
	} else {
		log.Info("[time: %.1f s, total: %v, success: %v, fail: %v] "+
			"| Server: [ %.1f tokens/s, %.1f req/s ] | Prompt length: %v",
			timeSpent, metric.Total, metric.Success, metric.Fail,
			metric.ServerOutputTokensPerSecond, metric.RequestPerSecond, cfg.InputTokens)
	}
}

func sendRequest(req *param.RequestParam) {
	var backendHandlers map[string]func(*param.RequestParam)
	cfg := req.Config
	if cfg.Stream {
		backendHandlers = map[string]func(*param.RequestParam){
			string(backend.VLLM): infer.SendVllmStreamRequest,
			string(backend.TRT):  infer.SendTrtStreamRequest,
		}
	} else {
		backendHandlers = map[string]func(*param.RequestParam){
			string(backend.VLLM): infer.SendVllmRequest,
			string(backend.TRT):  infer.SendTrtRequest,
			string(backend.TGI):  infer.SendTgiRequest,
		}
	}
	handler, ok := backendHandlers[strings.ToLower(cfg.Backend)]
	if !ok {
		panic(fmt.Sprintf("unsupported backend: %s", cfg.Backend))
	}
	go handler(req)
}

func saveResult(cfg *config.Config) {
	var values []*StatisticsSummary
	for _, value := range statistics {
		values = append(values, value)
	}
	// 按照Concurrency对结果排序
	sort.Slice(values, func(i, j int) bool {
		return values[i].Concurrency < values[j].Concurrency
	})
	utils.Save2Json(values, fmt.Sprintf("%s/statistics_%s.json", cfg.SaveDir, time.Now().Format(utils.TimeFormat)))
}

func finish(cfg *config.Config) (string, string) {
	log.Debugf("saving config file...")
	utils.Save2Json(cfg, fmt.Sprintf("%s/config_%s.json", cfg.SaveDir, time.Now().Format(utils.TimeFormat)))
	// 只保留最新一个统计文件
	utils.KeepFinalResult(cfg.SaveDir)
	if !cfg.Save2Cos {
		log.Debugf("See result in %v", cfg.SaveDir)
		return "", ""
	}
	log.Debugf("saving all files to cos...")
	downloadUrl, dstDir, err := cos.SaveFilesToCos(cfg)
	if err != nil {
		log.Errorf("❗️❗️❗ upload to cos failed")
		log.Debugf("🥳🤩🥰 Done!!!")
		return "", ""
	}
	if cfg.SendMsg {
		log.Debugf("download statistics result via 👉 %s 👈", downloadUrl)
		utils.SendMsg(cfg, downloadUrl, dstDir)
	}
	log.Debugf("🥳🤩🥰 Done!!!")
	return downloadUrl, dstDir
}
