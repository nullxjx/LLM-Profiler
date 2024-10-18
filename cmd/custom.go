package cmd

import (
	"fmt"
	"os"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/config"
	"github.com/nullxjx/LLM-Profiler/perf/throughput"
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
	if !common.IsDirEmpty(cfg.SaveDir) {
		log.Errorf("local save dir: %s is not empty", cfg.SaveDir)
		return
	}
	if err := common.SetLogFile(cfg.SaveDir + "/test.log"); err != nil {
		return
	}

	log.Infof("Begin performance testing on the model %v at %v:%v.", cfg.Model.Name, cfg.ServerIp, cfg.Port)
	log.Infof("Concurrency from %v to %v, Increment: %v, Duration: %v min, Estimated time: %v min",
		cfg.StartConcurrency, cfg.EndConcurrency, cfg.Increment, cfg.Duration,
		(cfg.EndConcurrency-cfg.StartConcurrency)/cfg.Increment*cfg.Duration)
	throughput.ThroughputTest(cfg)

	log.Infof("Done")
}
