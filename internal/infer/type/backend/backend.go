package backend

type BackendType string

// BackendType 的枚举值
const (
	VLLM BackendType = "vllm"
	TRT  BackendType = "trt"
	TGI  BackendType = "tgi"
)
