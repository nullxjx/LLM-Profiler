package cmd

import (
	"fmt"
	"github.com/nullxjx/llm_profiler/internal/perf/speed"
	"os"

	logformat "github.com/nullxjx/llm_profiler/pkg/log"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var speedCmd = &cobra.Command{
	Use:   "speed",
	Short: "测试模型单条速度",
	Long:  "测试模型单条速度",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		defer func() {
			if err != nil {
				fmt.Printf("perf test err: %v", err.Error())
				os.Exit(1)
			}
		}()

		speedTest()
	},
}

func init() {
	rootCmd.AddCommand(speedCmd)
	speedCmd.Flags().StringVarP(&ip, "ip", "i", "127.0.0.1", "模型IP")
	speedCmd.Flags().IntVarP(&port, "port", "p", 8000, "模型端口")
	speedCmd.Flags().StringVarP(&model, "model", "m", "codellama", "模型名字")
	speedCmd.Flags().StringVarP(&backend, "backend", "b", "vllm", "部署模型用的框架，当前支持vllm、tgi、trt")
	speedCmd.Flags().IntVarP(&prompt, "prompt", "l", 0, "prompt长度，0表示采用默认较短的prompt")
	speedCmd.Flags().Float32VarP(&temperature, "temperature", "t", 1, "温度，默认为1")
}

func speedTest() {
	if err := logformat.SetLogFile(user + "/test.log"); err != nil {
		log.Errorf("set log file failed: %v", err)
		return
	}
	speed.SpeedTest(ip, model, backend, port, prompt, temperature)

	log.Infof("Done")
}
