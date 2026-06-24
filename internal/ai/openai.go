package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func execOpenAI(url, model, key, pType, sysMsg string, msgs []Message, temp *float64, maxTokens int, jsonMode bool, tools []ToolDef) (string, []ToolCall, int64, error) {
	var apiMsgs []OpenAIChatMessage
	if sysMsg != "" {
		apiMsgs = append(apiMsgs, OpenAIChatMessage{
			Role:    "system",
			Content: sysMsg,
		})
	}
	for _, m := range msgs {
		var oCalls []OpenAIToolCall
		for _, tc := range m.ToolCalls {
			oCalls = append(oCalls, OpenAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Name,
					Arguments: tc.Args,
				},
			})
		}

		var content interface{} = m.Content
		if m.Content == "" && len(oCalls) > 0 {
			content = nil
		}

		apiMsgs = append(apiMsgs, OpenAIChatMessage{
			Role:       m.Role,
			Content:    content,
			ToolCalls:  oCalls,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		})
	}

	var apiTools []OpenAITool
	for _, t := range tools {
		apiTools = append(apiTools, OpenAITool{
			Type: "function",
			Function: OpenAIToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	reqBody := OpenAIChatRequest{
		Model:       model,
		Messages:    apiMsgs,
		Temperature: temp,
		MaxTokens:   maxTokens,
		Tools:       apiTools,
	}

	if jsonMode && len(apiTools) == 0 {
		reqBody.ResponseFormat = &OpenAIResponseFormat{Type: "json_object"}
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, 0, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return "", nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	if pType == "openrouter" {
		req.Header.Set("HTTP-Referer", "https://github.com/sativalol/Skyvern")
		req.Header.Set("X-Title", "Skyvern")
	}

	res, err := execWithRetry(req)
	if err != nil {
		return "", nil, 0, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", nil, 0, err
	}

	if res.StatusCode != 200 {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
			return "", nil, 0, fmt.Errorf("api error %d: %s", res.StatusCode, apiErr.Error.Message)
		}
		return "", nil, 0, fmt.Errorf("api error %d: %s", res.StatusCode, string(body))
	}

	var apiRes OpenAIChatResponse
	if err := json.Unmarshal(body, &apiRes); err != nil {
		return "", nil, 0, err
	}

	if apiRes.Error != nil && apiRes.Error.Message != "" {
		return "", nil, 0, fmt.Errorf("api error: %s", apiRes.Error.Message)
	}

	if len(apiRes.Choices) == 0 {
		return "", nil, 0, fmt.Errorf("empty api choices")
	}

	var tokens int64
	if apiRes.Usage != nil {
		tokens = int64(apiRes.Usage.TotalTokens)
	}

	var toolCalls []ToolCall
	for _, tc := range apiRes.Choices[0].Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:   tc.ID,
			Name: tc.Function.Name,
			Args: tc.Function.Arguments,
		})
	}

	return apiRes.Choices[0].Message.Content, toolCalls, tokens, nil
}
