package stream

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nullxjx/LLM-Profiler/infer/param"
)

// FinishCompletion 结束流式补全请求的channel，并把finish_reason透传回前端
func FinishCompletion(out chan<- []byte, model, finishReason string) {
	lastResp := GenCompletionStreamResp(model, finishReason)
	sb := strings.Builder{}
	sb.WriteString(FormatStreamData(lastResp))
	sb.WriteString(string(NewLine))
	sb.WriteString(string(EOF))
	sb.WriteString(string(NewLine))
	WriteLinesToChannel(out, sb.String())
}

// GenCompletionStreamResp 生成默认流式补全请求的返回结果
func GenCompletionStreamResp(model, finishReason string) param.InferRsp {
	choice := &param.Choice{
		Index:        0,
		Text:         "",
		Logprobs:     nil,
		FinishReason: finishReason,
	}
	rspData := param.InferRsp{
		Object:  "text_completion",
		Model:   model,
		Choices: []*param.Choice{choice},
	}
	return rspData
}

// FormatStreamData 将传入的数据格式化为流式数据格式，并返回，注意这里是单个换行
// 格式为：data: [data]\n
func FormatStreamData(data interface{}) string {
	bytes, _ := json.Marshal(data)
	return fmt.Sprintf(string(Starts+"%s"+NewLine), string(bytes))
}

// WriteLinesToChannel 将格式化后的数据按行写入channel，并根据EOF判断是否继续写入
// 如果输出了data: [DONE]\n，补充一个newline后，关闭channel，终止流式请求
// 返回值表示是否继续写入
func WriteLinesToChannel(out chan<- []byte, formattedData string) bool {
	lines := strings.SplitAfter(formattedData, "\n")
	for _, line := range lines {
		if len(line) > 0 {
			out <- []byte(line)
		}
		// 如果输出了data: [DONE]\n，补充一个newline后，关闭channel，终止流式请求
		if line == string(EOF) {
			out <- []byte(NewLine)
			return false
		}
	}
	return true
}
