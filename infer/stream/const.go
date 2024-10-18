package stream

// StreamFormat 后端推理框架流式请求的格式
type StreamFormat string

// StreamFormat 的枚举值
const (
	Starts     StreamFormat = "data: "
	NewLine    StreamFormat = "\n"
	EOF        StreamFormat = "data: [DONE]\n"
	ErrorEvent StreamFormat = "event: {error: %s}\n"
)

// InferType 推理类型
type InferType string

// InferType 的枚举值
const (
	Completion InferType = "completion"
	Chat       InferType = "chat"
)

// FinishReason 推理结束的原因
type FinishReason string

// FinishReason 的枚举值
const (
	Length FinishReason = "length" // 因上下文长度导致的停止或报错，用这个来标识
	Stop   FinishReason = "stop"   // 命中stop策略，或后端非上下文长度报错，用这个来标识
)
