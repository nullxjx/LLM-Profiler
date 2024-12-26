package postprocess

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/nullxjx/llm_profiler/internal/infer/type/stream"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// VllmStreamHandler 处理vllm流式请求的handler，简化参数调用
type VllmStreamHandler struct {
	Type  stream.InferType
	Model string
}

// InferErrRsp 代码补全报错结果
type InferErrRsp struct {
	Object  string      `json:"object"`
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Param   interface{} `json:"param"`
	Code    interface{} `json:"code"`
}

var VllmErrorPattern = regexp.MustCompile(`event: {error: [^"]+}`)

// Handle 处理流式请求的返回结果，如果返回结果是错误，则关闭channel
// 如果返回结果不是错误，则透传出去
func (s *VllmStreamHandler) Handle(ctx context.Context, out chan []byte, in <-chan []byte) error {
	defer close(out)
	for {
		data, ok := <-in
		if !ok {
			break
		}
		// 尝试解析成errRsp，解析没有问题说明vllm报错了，把channel关掉
		// 如果解析出错，说明不是vllm的错误，直接透传出去
		var errRsp InferErrRsp
		if err := json.Unmarshal(data, &errRsp); err == nil {
			log.Errorf("Call vLLM stream API error: %v", errRsp)
			FinishCompletion(out, s.Model, string(stream.Length))
			return err
		}
		match := VllmErrorPattern.FindString(string(data))
		if match != "" {
			FinishCompletion(out, s.Model, string(stream.Length))
			if strings.Contains(match, "EOF") {
				break
			}
			return errors.New("vllm stream api return error")
		}
		out <- data
	}
	return nil
}
