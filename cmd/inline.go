package cmd

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/config"
	"github.com/nullxjx/LLM-Profiler/perf/throughput"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type IterationData struct {
	TimeoutSeconds         int     `json:"timeoutSeconds"`
	InputTokens            int     `json:"inputTokens"`
	OutputTokens           int     `json:"outputTokens"`
	OutputTokensPerSeconds float64 `json:"outputTokensPerSeconds"`
	InputTokensPerSeconds  float64 `json:"inputTokensPerSeconds"`
	RequestPerSeconds      float64 `json:"requestPerSeconds"`
}

type InlineResult struct {
	Data   []*IterationData  `json:"data"`
	Config map[string]string `json:"config"`
}

var autoCmd = &cobra.Command{
	Use:   "inline",
	Short: "测试模型给定延迟下的吞吐量",
	Long:  "测试模型给定延迟下的吞吐量",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		defer func() {
			if err != nil {
				fmt.Printf("perf test err: %v", err.Error())
				os.Exit(1)
			}
		}()

		inlineTest()
	},
}

func init() {
	rootCmd.AddCommand(autoCmd)
	autoCmd.Flags().StringVarP(&ip, "ip", "i", "127.0.0.1", "模型IP")
	autoCmd.Flags().IntVarP(&port, "port", "p", 8000, "模型端口")
	autoCmd.Flags().StringVarP(&model, "model", "m", "codellama", "模型名字")
	autoCmd.Flags().StringVarP(&backend, "backend", "b", "vllm", "部署模型用的框架，当前支持vllm、tgi、trt")
	autoCmd.Flags().StringVarP(&user, "user", "u", "nullxjx", "你的企微id")
}

func readInlineEnvs() ([]int, []int, []int) {
	envVar := os.Getenv("MAX_NEW_TOKENS")
	var maxNewTokens []int
	if envVar != "" {
		// 使用逗号分割字符串
		strSlice := strings.Split(envVar, ",")
		// 将字符串切片转换为整数切片
		maxNewTokens = make([]int, len(strSlice))
		for i, str := range strSlice {
			value, err := strconv.Atoi(str)
			if err != nil {
				fmt.Println("Error converting string to int:", err)
				break
			}
			maxNewTokens[i] = value
		}
	}
	if len(maxNewTokens) == 0 {
		maxNewTokens = []int{16, 32}
	}

	envVar = os.Getenv("INPUT_TOKENS")
	var inputTokens []int
	if envVar != "" {
		// 使用逗号分割字符串
		strSlice := strings.Split(envVar, ",")
		// 将字符串切片转换为整数切片
		inputTokens = make([]int, len(strSlice))
		for i, str := range strSlice {
			value, err := strconv.Atoi(str)
			if err != nil {
				fmt.Println("Error converting string to int:", err)
				break
			}
			inputTokens[i] = value
		}
	}
	if len(inputTokens) == 0 {
		inputTokens = []int{100, 200, 400, 600, 800, 1000}
		//inputTokens = []int{100, 200, 400}
	}

	envVar = os.Getenv("TIME")
	var timeoutSeconds []int
	if envVar != "" {
		// 使用逗号分割字符串
		strSlice := strings.Split(envVar, ",")
		// 将字符串切片转换为整数切片
		timeoutSeconds = make([]int, len(strSlice))
		for i, str := range strSlice {
			value, err := strconv.Atoi(str)
			if err != nil {
				fmt.Println("Error converting string to int:", err)
				break
			}
			timeoutSeconds[i] = value
		}
	}
	if len(timeoutSeconds) == 0 {
		timeoutSeconds = []int{1, 2}
	}

	return timeoutSeconds, inputTokens, maxNewTokens
}

func inlineTest() {
	timeoutSeconds, inputTokens, maxNewTokens := readInlineEnvs()
	rootSaveDir := fmt.Sprintf("%v/auto_%v", user, time.Now().Format("2006-01-02-15-04-05"))
	cfg := &config.Config{
		Model:            config.ModelConfig{Name: model, Version: "1"},
		ServerIp:         ip,
		Port:             port,
		RequestTimeout:   0,
		Backend:          backend,
		MaxTokens:        0,
		InputTokens:      0,
		StartConcurrency: 30,
		EndConcurrency:   5000,
		Increment:        30,
		Duration:         1,
		SaveDir:          rootSaveDir,
		Bucket:           Bucket,
		Region:           "ap-shanghai",
		SubFolder:        "perf_analyzer",
		WebhookUrl:       WebhookUrl,
		User:             user,
		Auto:             true,
		Save2Cos:         true,
	}

	// 判断saveDir是否为空，不为空直接退出
	if !common.IsDirEmpty(cfg.SaveDir) {
		log.Errorf("Local save dir: %s is not empty", cfg.SaveDir)
		return
	}
	if err := common.SetLogFile(rootSaveDir + "/test.log"); err != nil {
		return
	}

	log.Infof("Begin performance testing on the model %v at %v:%v.", cfg.Model.Name, cfg.ServerIp, cfg.Port)
	log.Infof("timeout: %v, max_new_tokens: %v, input_tokens: %v", timeoutSeconds, maxNewTokens, inputTokens)
	log.Infof("Total estimated time: %v min", (cfg.EndConcurrency-cfg.StartConcurrency)/
		cfg.Increment*cfg.Duration*len(maxNewTokens)*len(inputTokens)*len(timeoutSeconds))

	start := time.Now()
	var data []*IterationData
	for _, t := range timeoutSeconds {
		for _, i := range inputTokens {
			for _, n := range maxNewTokens {
				log.Debugf("Configuration of this iteration: timeout:%vs | input_tokens:%v | max_new_tokens:%v", t, i, n)

				cfg.MaxTokens = uint32(n)
				cfg.InputTokens = i
				cfg.RequestTimeout = t
				cfg.SaveDir = fmt.Sprintf("%v/timeout_%vs/input_tokens_%v/output_tokens_%v", rootSaveDir, t, i, n)

				throughput.ThroughputTest(cfg)
				maxRequestPerSecond, maxInputTokensPerSecond, maxOutputTokensPerSecond := throughput.GetMaxThroughput()
				log.Infof("Max throughput [req/s:%v, input_tokens/s:%v, output_tokens/s:%v], prompt length: %v",
					maxRequestPerSecond, maxInputTokensPerSecond, maxOutputTokensPerSecond, cfg.InputTokens)
				data = append(data, &IterationData{
					TimeoutSeconds:         t,
					InputTokens:            i,
					OutputTokens:           n,
					OutputTokensPerSeconds: math.Round(maxOutputTokensPerSecond*100) / 100, // 保留2位小数
					InputTokensPerSeconds:  math.Round(maxInputTokensPerSecond*100) / 100,
					RequestPerSeconds:      math.Round(maxRequestPerSecond*100) / 100,
				})

				time.Sleep(15 * time.Second)
			}
		}
	}

	cfg_ := make(map[string]string)
	cfg_["user"] = cfg.User
	cfg_["model"] = cfg.Model.Name
	cfg_["backend"] = cfg.Backend
	cfg_["url"] = fmt.Sprintf("%v:%v", ip, port)
	cfg_["time"] = fmt.Sprintf("%.3f min", time.Since(start).Minutes())
	cfg_["save"] = "cos"
	if !cfg.Save2Cos {
		cfg_["save"] = "local"
	}
	result := &InlineResult{
		Config: cfg_,
		Data:   data,
	}
	common.Save2Json(result, "visualizer/inline_results.json")

	log.Infof("Done")
}
