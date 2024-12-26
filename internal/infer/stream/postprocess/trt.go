package postprocess

import (
	"context"
	"regexp"

	"github.com/nullxjx/llm_profiler/internal/infer/type/stream"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TrtStreamHandler 处理 trt 流式请求的handler，简化参数调用
type TrtStreamHandler struct {
	Type  stream.InferType
	Model string
}

var ErrorPattern = regexp.MustCompile(`data: {"error":"[^"]+"}`)

// TrtChunk trt流式输出chunk结构
type TrtChunk struct {
	BatchIndex       int       `json:"batch_index"`
	ContextLogits    float64   `json:"context_logits"`
	CumLogProbs      float64   `json:"cum_log_probs"`
	GenerationLogits float64   `json:"generation_logits"`
	ModelName        string    `json:"model_name"`
	ModelVersion     string    `json:"model_version"`
	OutputLogProbs   []float64 `json:"output_log_probs"`
	SequenceEnd      bool      `json:"sequence_end"`
	SequenceID       int       `json:"sequence_id"`
	SequenceStart    bool      `json:"sequence_start"`
	TextOutput       string    `json:"text_output"`
}

// Handle 处理流式请求的返回结果，如果返回结果是错误，则关闭channel
func (s *TrtStreamHandler) Handle(ctx context.Context, out chan []byte, in <-chan []byte) error {
	defer close(out)
	for {
		data, ok := <-in
		if !ok {
			break
		}

		match := ErrorPattern.FindString(string(data))
		if match != "" {
			log.Errorf("trt stream api return error: %v", match)
			FinishCompletion(out, s.Model, string(stream.Length))
			return errors.New("trt stream api return error")
		}
		out <- data
	}
	return nil
}
