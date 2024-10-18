package triton

import (
	"encoding/json"
)

type TrtReq struct {
	TextInput   string  `json:"text_input"`
	MaxTokens   int32   `json:"max_tokens"`
	BadWords    string  `json:"bad_words"`
	StopWords   string  `json:"stop_words"`
	Stream      bool    `json:"stream"`
	TopP        float32 `json:"top_p"`
	Temperature float32 `json:"temperature"`
}

type TrtRsp struct {
	ModelName     string `json:"model_name"`
	ModelVersion  string `json:"model_version"`
	SequenceEnd   bool   `json:"sequence_end"`
	SequenceId    int    `json:"sequence_id"`
	SequenceStart bool   `json:"sequence_start"`
	TextOutput    string `json:"text_output"`
}

type VllmReq struct {
	Stream      bool    `json:"stream"`
	Temperature float32 `json:"temperature"`
	MaxTokens   int32   `json:"max_tokens"`
	TextInput   string  `json:"text_input"`
}

type VllmRsp struct {
	ModelName    string          `json:"model_name"`
	ModelVersion string          `json:"model_version"`
	TextOutput   json.RawMessage `json:"text_output"`
}

type ProxyRequest struct {
	Model         string   `json:"model"`
	Prompt        string   `json:"prompt"`
	Stop          []string `json:"stop,omitempty"`
	Temperature   float32  `json:"temperature"`
	MaxTokens     uint32   `json:"max_tokens"`
	UseBeamSearch bool     `json:"use_beam_search"`
	Stream        bool     `json:"stream"`
	IgnoreEos     bool     `json:"ignore_eos"`
}
