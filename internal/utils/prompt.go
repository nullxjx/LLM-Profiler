package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

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
