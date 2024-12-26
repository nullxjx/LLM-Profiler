package cmd

import (
	"fmt"
	"github.com/nullxjx/llm_profiler/internal/perf/speed"
	"github.com/nullxjx/llm_profiler/internal/perf/throughput"
	"os"

	"github.com/nullxjx/llm_profiler/config"
	"github.com/nullxjx/llm_profiler/internal/utils"
	logformat "github.com/nullxjx/llm_profiler/pkg/log"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var configPath string

var customCmd = &cobra.Command{
	Use:   "custom",
	Short: "使用配置文件测试模型吞吐量和延迟",
	Long:  "使用配置文件测试模型吞吐量和延迟",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		defer func() {
			if err != nil {
				fmt.Printf("custom perf test err: %v", err.Error())
				os.Exit(1)
			}
		}()

		customTest()
	},
}

func init() {
	rootCmd.AddCommand(customCmd)
	customCmd.Flags().StringVarP(&configPath, "config_path", "c", "config/config_local.yml", "配置文件路径")
}

func customTest() {
	cfg, err := config.ReadConf(configPath)
	if err != nil {
		fmt.Printf("read config error: %v\n", err)
		return
	}

	// 判断saveDir是否为空，不为空直接退出
	if !utils.IsDirEmpty(cfg.SaveDir) {
		log.Errorf("local save dir: %s is not empty", cfg.SaveDir)
		return
	}
	if err := logformat.SetLogFile(cfg.SaveDir + "/test.log"); err != nil {
		return
	}

	log.Infof("Begin performance testing on the model %v at %v:%v.", cfg.Model.Name, cfg.ServerIp, cfg.Port)
	log.Infof("Concurrency from %v to %v, Increment: %v, Duration: %v min, Estimated time: %v min",
		cfg.StartConcurrency, cfg.EndConcurrency, cfg.Increment, cfg.Duration,
		(cfg.EndConcurrency-cfg.StartConcurrency)/cfg.Increment*cfg.Duration)

	if cfg.Stream {
		// 先测出只有一条请求的时的速度（每秒token数），可以使用多条输入数据测试几次取均值
		log.Infof("calculate max stream speed...")
		s, err := speed.CalStreamSpeed(cfg)
		if err != nil {
			log.Errorf("calculate max speed error: %v", err)
			return
		}
		log.Infof("max stream speed: %.1f tokens/s, first_token: %.1f ms", s.TokensPerSecond, s.FirstTokenTime)
		cfg.MaxStreamSpeed = s.TokensPerSecond
	}
	throughput.StartTest(cfg)
	log.Infof("Done")
}
