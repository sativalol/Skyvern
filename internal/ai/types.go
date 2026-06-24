package ai

import (
	"encoding/json"
	"time"
)

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Args string `json:"args"`
}

type ToolDef struct {
	Name        string                                            `json:"name"`
	Description string                                            `json:"description"`
	Parameters  interface{}                                       `json:"parameters"`
	Handler     func(args map[string]interface{}) (string, error) `json:"-"`
}

type GenOpts struct {
	SystemMsg   string            `json:"system_msg"`
	Messages    []Message         `json:"messages"`
	UserMsg     string            `json:"user_msg"`
	Temperature float64           `json:"temperature"`
	MaxTokens   int               `json:"max_tokens"`
	JSONMode    bool              `json:"json_mode"`
	Variables   map[string]string `json:"variables"`
	FallbackID  string            `json:"fallback_id"`
	Tools       []ToolDef         `json:"tools,omitempty"`
}

type Result struct {
	Text       string        `json:"text"`
	Model      string        `json:"model"`
	ProviderID string        `json:"provider_id"`
	Duration   time.Duration `json:"duration"`
	ToolCalls  []ToolCall    `json:"tool_calls,omitempty"`
}

type OpenAIChatMessage struct {
	Role       string            `json:"role"`
	Content    interface{}       `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Name       string            `json:"name,omitempty"`
}

type OpenAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type OpenAIResponseFormat struct {
	Type string `json:"type"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAITool struct {
	Type     string             `json:"type"`
	Function OpenAIToolFunction `json:"function"`
}

type OpenAIToolFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type OpenAIChatRequest struct {
	Model          string                `json:"model"`
	Messages       []OpenAIChatMessage   `json:"messages"`
	Temperature    *float64              `json:"temperature,omitempty"`
	MaxTokens      int                   `json:"max_tokens,omitempty"`
	ResponseFormat *OpenAIResponseFormat `json:"response_format,omitempty"`
	Tools          []OpenAITool          `json:"tools,omitempty"`
}

type OpenAIChatResponse struct {
	Choices []struct {
		Message struct {
			Role      string           `json:"role"`
			Content   string           `json:"content"`
			ToolCalls []OpenAIToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
	} `json:"choices"`
	Usage *OpenAIUsage `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // can be string or list of content blocks
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []AnthropicMessage `json:"messages"`
	Temperature *float64           `json:"temperature,omitempty"`
	Tools       []interface{}      `json:"tools,omitempty"`
}

type AnthropicResponse struct {
	Content []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text,omitempty"`
		ID    string          `json:"id,omitempty"`
		Name  string          `json:"name,omitempty"`
		Input json.RawMessage `json:"input,omitempty"`
	} `json:"content"`
	Usage *AnthropicUsage `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}
