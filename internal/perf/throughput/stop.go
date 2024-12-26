package throughput

import (
	"github.com/nullxjx/llm_profiler/config"
	"github.com/nullxjx/llm_profiler/internal/utils"

	log "github.com/sirupsen/logrus"
)

// stop 停止判断
func stop(cfg *config.Config, current int) bool {
	if cfg.Stream {
		return streamStopCheck(cfg, current)
	}
	return nonStreamStopCheck(cfg, current)
}

// nonStreamStopCheck 非流式场景停止判断
func nonStreamStopCheck(cfg *config.Config, current int) bool {
	defer saveResult(cfg)

	start := cfg.StartConcurrency
	increment := cfg.Increment

	// 成功率小于0.95
	successRate := float64(statistics[current].Success) / float64(statistics[current].Total)
	if successRate < 0.95 {
		log.Warnf("The success rate is %v", successRate)
		delete(statistics, current)
		last := current - cfg.Increment
		_, ok := statistics[last]
		if ok {
			log.Infof("Max throughput %.3f tokens/s, %.3f req/s, prompt length: %v",
				statistics[last].ServerOutputTokensPerSecond, statistics[last].RequestPerSecond, cfg.InputTokens)
		} else {
			log.Infof("Max throughput is zero, please decrease your concurrency")
		}
		return true
	}

	window := 5
	if current-(window+1)*increment < start {
		return false
	}

	// 当前吞吐量
	currentThroughput := statistics[current].ServerOutputTokensPerSecond
	var pastThroughput float64 = 0
	for i := 1; i <= window; i++ {
		pastThroughput += statistics[current-i*increment].ServerOutputTokensPerSecond
	}
	historyAvg := pastThroughput / float64(window)

	// 吞吐量收敛了，停止
	if utils.IsClose(currentThroughput, historyAvg, 0.02) {
		log.Debugf("The throughput have converged to %v, early stop", historyAvg)
		return true
	}

	// 吞吐量出现恶化趋势
	if currentThroughput < historyAvg*0.95 {
		log.Warnf("The throughput begin to drop, history: %.3f, current: %.3f", historyAvg, currentThroughput)
		delete(statistics, current)
		return true
	}

	return false
}

// streamStopCheck 流式场景停止判断
func streamStopCheck(cfg *config.Config, current int) bool {
	defer saveResult(cfg)

	// 成功率小于0.95
	successRate := float64(statistics[current].Success) / float64(statistics[current].Total)
	if successRate < 0.95 {
		log.Warnf("The success rate is %v", successRate)
		delete(statistics, current)
		last := current - cfg.Increment
		_, ok := statistics[last]
		if ok {
			log.Infof("Max throughput %.3f tokens/s, %.3f req/s, prompt length: %v",
				statistics[last].ServerOutputTokensPerSecond, statistics[last].RequestPerSecond, cfg.InputTokens)
		} else {
			log.Infof("Max throughput is zero, please decrease your concurrency")
		}
		return true
	}

	// 流式场景是否停止
	avgClientOutputTokensPerSecond := statistics[current].ClientOutputTokensPerSecond
	// 这里乘以0.95是容纳一定的波动情况
	if cfg.Stream && (avgClientOutputTokensPerSecond < cfg.MaxStreamSpeed*float64(cfg.StreamThresholds)/100*0.95) {
		log.Warnf("avgClientOutputTokensPerSecond is %v, MaxStreamSpeed is %v, StreamThresholds is %v%%",
			avgClientOutputTokensPerSecond, cfg.MaxStreamSpeed, cfg.StreamThresholds)
		delete(statistics, current)

		last := current - cfg.Increment
		_, ok := statistics[last]
		if ok {
			log.Infof("Max throughput %.3f req/s, prompt length: %v", statistics[last].RequestPerSecond, cfg.InputTokens)
		} else {
			log.Infof("Max throughput is zero, please decrease your concurrency")
		}
		return true
	}
	return false
}
