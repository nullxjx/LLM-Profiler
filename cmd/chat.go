package cmd

import (
	"fmt"
	"os"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/config"
	"github.com/nullxjx/LLM-Profiler/perf/speed"
	"github.com/nullxjx/LLM-Profiler/perf/throughput"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "测试流式对话场景的吞吐量",
	Long:  "测试流式对话场景的吞吐量",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		defer func() {
			if err != nil {
				fmt.Printf("perf test err: %v", err.Error())
				os.Exit(1)
			}
		}()

		chatTest()
	},
}

func init() {
	rootCmd.AddCommand(chatCmd)
	chatCmd.Flags().StringVarP(&configPath, "config_path", "c", "config/config_local.yml", "配置文件路径")
}

func chatTest() {
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

	// 先测出只有一条请求的时的速度（每秒token数），可以使用多条输入数据测试几次取均值
	log.Infof("calculate max stream speed...")
	maxSpeed, err := speed.CalculateStreamSpeed(cfg)
	if err != nil {
		log.Errorf("calculate max speed error: %v", err)
		return
	}
	log.Infof("max stream speed: %.1ftokens/s", maxSpeed)

	cfg.MaxStreamSpeed = maxSpeed
	throughput.ThroughputTest(cfg)
	log.Infof("Done")
}
