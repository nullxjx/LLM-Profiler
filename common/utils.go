package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/nullxjx/LLM-Profiler/config"
	log "github.com/sirupsen/logrus"
)

func IsDirEmpty(dir string) bool {
	// 检查目录是否存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return true
	}

	// 读取目录中的文件信息列表
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Errorf("Error reading directory: %v\n", err)
		return true
	}

	// 判断目录是否为空
	if len(files) == 0 {
		return true
	}
	return false
}

type Message struct {
	MsgType  string `json:"msgtype"`
	Markdown Text   `json:"markdown"`
}

type Text struct {
	Content string `json:"content"`
}

func SendMsg(cfg *config.Config, downloadUrl, dstDir string) {
	// 向企微推送消息
	msg := fmt.Sprintf("## 🥳🤩🥰 Performance Test Done \nDownload statistics result via 👉 [me](%s) 👈 \n"+
		//"\nThis presigned URL is available in **6 hours**. After time expired, please find result in cos:\n"+
		"> bucket: <font color=\"info\">%s</font>\n"+
		"> path: <font color=\"info\">%s</font>\n\n", downloadUrl, cfg.Bucket, dstDir)
	if cfg.User != "" {
		msg += fmt.Sprintf("<@%s>\n", cfg.User)
	}
	SendWebHook(cfg.WebhookUrl, msg)
}

// SendWebHook 通过webhook向企微推送消息
func SendWebHook(webhookURL, content string) {
	msg := Message{
		MsgType: "markdown",
		Markdown: Text{
			Content: content,
		},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Error marshalling message: %v", err)
		return
	}

	// 创建一个自定义的HTTP客户端，禁用TLS证书验证
	customClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := customClient.Post(webhookURL, "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		log.Errorf("Error sending message: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading response body: %v", err)
		return
	}

	log.Infof("企业微信机器人响应: %v", string(body))
}

// Save2Json 保存数据到json文件中，saveDir为保存文件的路径
func Save2Json(v any, saveDir string) {
	jsonData, err := json.Marshal(v)
	if err != nil {
		log.Errorf("Error encoding JSON: %v", err)
		return
	}

	// 获取文件所在的目录
	dir := filepath.Dir(saveDir)

	// 创建目录（如果不存在）
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Errorf("failed to create directories: %v", err)
		return
	}

	file, err := os.Create(saveDir)
	if err != nil {
		log.Errorf("Error creating file: %v", err)
		return
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		log.Errorf("Error writing JSON data to file: %v", err)
		return
	}

	//log.Infof("JSON data saved to file: %v", saveDir)
}

func KeepFinalResult(saveDir string) {
	log.Infof("🤖🤖🤖 clearing unused files...")
	// 读取目录中的所有文件
	files, err := ioutil.ReadDir(saveDir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		return
	}

	// 定义正则表达式来匹配文件名
	pattern := regexp.MustCompile(`^statistics_\d{4}-\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\.json$`)

	// 筛选出符合命名规则的文件
	var matchedFiles []os.FileInfo
	for _, file := range files {
		if pattern.MatchString(file.Name()) {
			matchedFiles = append(matchedFiles, file)
		}
	}

	// 按时间戳排序文件
	sort.Slice(matchedFiles, func(i, j int) bool {
		return matchedFiles[i].ModTime().After(matchedFiles[j].ModTime())
	})

	// 删除除最新文件之外的其他文件
	for i := 1; i < len(matchedFiles); i++ {
		filePath := filepath.Join(saveDir, matchedFiles[i].Name())
		err := os.Remove(filePath)
		if err != nil {
			log.Infof("Error deleting file %s: %v", filePath, err)
		} else {
			log.Infof("Deleted file %s", filePath)
		}
	}
}

func Post(ctx context.Context, url string, rawBody interface{}) ([]byte, error) {
	body, jsonErr := json.Marshal(rawBody)
	if jsonErr != nil {
		return nil, jsonErr
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("infer error, status code: %d, body: %v", resp.StatusCode, respBody))
	}

	return respBody, nil
}

func GenerateRandomStr(length int) string {
	rand.Seed(time.Now().UnixNano())

	// 26个字母和10个数字
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var randomStr string
	for i := 0; i < length; i++ {
		index := rand.Intn(len(characters))
		randomStr += string(characters[index])
	}

	return randomStr
}

type Input struct {
	Prompt string `json:"prompt"`
	Tokens int    `json:"tokens"`
}

// ReadPrompts 从文件中读取给定长度的prompts
// 输入的 promptLength 表示prompt中的token数量
func ReadPrompts(promptLength int) ([]string, error) {
	var result []string
	inputDataPath := fmt.Sprintf("data/ShareGPT_V3_unfiltered_cleaned_split/input_tokens_%d.json", promptLength)
	file, err := os.Open(inputDataPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var inputs []Input
	err = decoder.Decode(&inputs)
	if err != nil {
		return nil, err
	}

	promptPrefix := "Please provide a comprehensive and detailed response based on the following information: "
	for _, input := range inputs {
		result = append(result, promptPrefix+input.Prompt)
	}
	return result, nil
}

// ReadPromptsWithTokens 从文件中读取给定长度的prompts，包含其token信息统计
// 输入的 promptLength 表示prompt中的token数量
func ReadPromptsWithTokens(promptLength int) ([]Input, error) {
	inputDataPath := fmt.Sprintf("data/ShareGPT_V3_unfiltered_cleaned_split/input_tokens_%d.json", promptLength)
	file, err := os.Open(inputDataPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var inputs []Input
	err = decoder.Decode(&inputs)
	if err != nil {
		return nil, err
	}

	promptPrefix := "Please provide a comprehensive and detailed response based on the following information: "
	for _, input := range inputs {
		input.Prompt = promptPrefix + input.Prompt
	}
	return inputs, nil
}

//// ReadPrompts 生成测试需要的prompts
//func ReadPrompts(cfg *config.Config) ([]string, error) {
//	return []string{
//		"55WOf21VqrJ3SuODOfIbmzYbJnleCZQLXKDtrqVACOcvVVikVK1FhbWRrisj1nH2RslfrEwluEqBoarP7OgH8xnV3EcWxEbhaOvsWgS14BC4vWCHVHcNQQRktPpHDlQbts8GzOQGhjLbybySEZ0MJmIBJtaJRaZ49NJbkO4ELemRCQ5IVrwyWF5CGJ7rH98run6wN5BSABTnY63zvW30iWz1AkKy2IlXt5tggN7v5ErB8iDdleo9HvsWR04UHkkODNediEP516hK58uncRVxRHpzDseCPmgsS7MWiyzJoMs4SdVCjpVRHqdqU3iQkkHs1xKrg1QtIUgXCZ4OjOqlHLtTmIS48XJW98LOEXLmzG4Elz0C0SAG1Qt4g39HgrvMhczKx8CMhvHM1iCVYhN34YA5GwAZyGwBBFJ3QKpStVncU4NxPtJYJ5rTCZPWpYQn3xwPoudkz4WxhpsCuigLfNmGw",
//		"CldlsJ5BBktvdY2QdGI7MpuRvpCAO5HBd70YdHerXLroZzAVbdf0MduJYkflLAfifMuSUUftXZqqg2VDsWJQjmTdRKRe5jW6FML7pPs1IwR0cUOuOZnLpSTNX2Epv3WSHRIVGXmGrfFy7QspMIc68vgeO7sw33kEOqvx98CxFysk8sRytReZ8DowppH8bipSe2N3Sw7qvx1rh1u8RiFrIihVPWHeG5C8yWv2vZYrLQzdd9Eb8gonC1SKiVvpOak0TdxArMq4qGFWe2Bb7hLcAaPz8xGmRfw1QVrUXGwm0ieYEgufpQl0nVQLqYD2Fec7jC4DXchf0brVCCp7koG1bX6dnlsS3uAGLMUvZw3SunYDo6n96h3B21h7AnYk3XKm9X2AJYk4vX6DQadam8onaX1J6Ejz647TCglPDK4xQGF4PyLmfyAzuLKy4AAPvYFUxTVPsRphDXjkbm6yCk6pHnd625LvfaxCB",
//	}, nil
//}

func IsClose(a, b, tolerance float64) bool {
	if a == 0 && b == 0 {
		return true
	}
	if a == 0 || b == 0 {
		return false
	}
	relativeError := math.Abs((a - b) / math.Max(math.Abs(a), math.Abs(b)))
	return relativeError <= tolerance
}

func MeanWithoutMinMax(numbers []float64) float64 {
	if len(numbers) < 3 {
		return 0
	}

	min := math.MaxFloat64
	max := -math.MaxFloat64
	sum := 0.0

	for _, num := range numbers {
		sum += num
		if num < min {
			min = num
		}
		if num > max {
			max = num
		}
	}

	sum -= min + max
	mean := sum / float64(len(numbers)-2)
	return mean
}
