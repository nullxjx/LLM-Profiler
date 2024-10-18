package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nullxjx/LLM-Profiler/common"
	"github.com/nullxjx/LLM-Profiler/config"
	"github.com/nullxjx/LLM-Profiler/perf/throughput"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type Result struct {
	Data   map[string]map[string]map[string]string `json:"data"`
	Config map[string]string                       `json:"config"`
}

var (
	ip      string
	port    int
	model   string
	backend string
	user    string
	prompt  int
)

const (
	WebhookUrl string = "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx"
	Bucket     string = "ai-file-xxx"
)

var peakCmd = &cobra.Command{
	Use:   "peak",
	Short: "测试模型峰值吞吐量（无延迟限制）",
	Long:  "测试模型峰值吞吐量（无延迟限制）",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		defer func() {
			if err != nil {
				fmt.Printf("perf test err: %v", err.Error())
				os.Exit(1)
			}
		}()

		peakTest()
	},
}

func init() {
	rootCmd.AddCommand(peakCmd)
	peakCmd.Flags().StringVarP(&ip, "ip", "i", "127.0.0.1", "模型IP")
	peakCmd.Flags().IntVarP(&port, "port", "p", 8000, "模型端口")
	peakCmd.Flags().StringVarP(&model, "model", "m", "codellama", "模型名字")
	peakCmd.Flags().StringVarP(&backend, "backend", "b", "vllm", "部署模型用的框架，当前支持vllm、tgi、triton-vllm、triton-trt")
	peakCmd.Flags().StringVarP(&user, "user", "u", "nullxjx", "你的企微id")
}

func readPeakEnvs() ([]int, []int) {
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
		maxNewTokens = []int{8, 16, 32, 64, 128, 256}
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
		inputTokens = []int{500, 1000, 1500, 2000}
	}

	return maxNewTokens, inputTokens
}

func peakTest() {
	maxNewTokens, inputTokens := readPeakEnvs()
	rootSaveDir := fmt.Sprintf("%v/auto_%v", user, time.Now().Format("2006-01-02-15-04-05"))
	cfg := &config.Config{
		Model:            config.ModelConfig{Name: model, Version: "1"},
		ServerIp:         ip,
		Port:             port,
		RequestTimeout:   1200 * 1000, // 1200秒，20分钟
		Backend:          backend,
		MaxTokens:        0,
		InputTokens:      0,
		StartConcurrency: 100,
		EndConcurrency:   5000,
		Increment:        100,
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
	cosPathMap := make(map[string]map[string]map[string]string)
	// 判断saveDir是否为空，不为空直接退出
	if !common.IsDirEmpty(cfg.SaveDir) {
		log.Errorf("Local save dir: %s is not empty", cfg.SaveDir)
		return
	}
	if err := common.SetLogFile(rootSaveDir + "/test.log"); err != nil {
		return
	}

	log.Infof("Begin performance testing on the model %v at %v:%v.", cfg.Model.Name, cfg.ServerIp, cfg.Port)
	log.Infof("max_new_tokens: %v, input_tokens: %v", maxNewTokens, inputTokens)
	log.Infof("Total estimated time: %v min",
		(cfg.EndConcurrency-cfg.StartConcurrency)/cfg.Increment*cfg.Duration*len(maxNewTokens)*len(inputTokens))

	start := time.Now()
	for _, i := range inputTokens {
		m := make(map[string]map[string]string)
		for _, n := range maxNewTokens {
			log.Debugf("Configuration of this iteration: input_tokens:%v max_new_tokens:%v", i, n)

			cfg.MaxTokens = uint32(n)
			cfg.InputTokens = i
			cfg.SaveDir = fmt.Sprintf("%v/input_tokens_%v/output_tokens_%v", rootSaveDir, i, n)

			downloadUrl, dstDir := throughput.ThroughputTest(cfg)
			data := make(map[string]string)
			data["cos_url"] = downloadUrl
			if !cfg.Save2Cos {
				absolutePath, err := filepath.Abs(cfg.SaveDir)
				if err != nil {
					log.Errorf("convert to abs path error: %v", err)
					return
				}
				data["cos_url"] = absolutePath
			}
			data["cos_path"] = dstDir
			m[fmt.Sprintf("%d", n)] = data

			time.Sleep(30 * time.Second)
		}
		cosPathMap[fmt.Sprintf("%d", i)] = m
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

	result := &Result{
		Data:   cosPathMap,
		Config: cfg_,
	}
	common.Save2Json(result, "visualizer/peak_results.json")

	log.Infof("Done")
}
