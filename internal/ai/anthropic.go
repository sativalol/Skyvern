package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func execAnthropic(url, model, key, sysMsg string, msgs []Message, temp *float64, maxTokens int, tools []ToolDef) (string, []ToolCall, int64, error) {
	var apiMsgs []AnthropicMessage
	for _, m := range msgs {
		role := m.Role
		if role == "system" {
			continue
		}

		if len(m.ToolCalls) > 0 {
			var blocks []interface{}
			if m.Content != "" {
				blocks = append(blocks, map[string]interface{}{
					"type": "text",
					"text": m.Content,
				})
			}
			for _, tc := range m.ToolCalls {
				var inputMap map[string]interface{}
				_ = json.Unmarshal([]byte(tc.Args), &inputMap)
				blocks = append(blocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Name,
					"input": inputMap,
				})
			}
			apiMsgs = append(apiMsgs, AnthropicMessage{
				Role:    role,
				Content: blocks,
			})
			continue
		}

		if m.ToolCallID != "" {
			blocks := []interface{}{
				map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": m.ToolCallID,
					"content":     m.Content,
				},
			}
			apiMsgs = append(apiMsgs, AnthropicMessage{
				Role:    "user",
				Content: blocks,
			})
			continue
		}

		apiMsgs = append(apiMsgs, AnthropicMessage{
			Role:    role,
			Content: m.Content,
		})
	}

	var apiTools []interface{}
	for _, t := range tools {
		apiTools = append(apiTools, map[string]interface{}{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": t.Parameters,
		})
	}

	reqBody := AnthropicRequest{
		Model:       model,
		MaxTokens:   maxTokens,
		System:      sysMsg,
		Temperature: temp,
		Messages:    apiMsgs,
		Tools:       apiTools,
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
	req.Header.Set("x-api-key", key)
	req.Header.Set("anthropic-version", "2023-06-01")

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
			return "", nil, 0, fmt.Errorf("anthropic error %d: %s", res.StatusCode, apiErr.Error.Message)
		}
		return "", nil, 0, fmt.Errorf("anthropic error %d: %s", res.StatusCode, string(body))
	}

	var apiRes AnthropicResponse
	if err := json.Unmarshal(body, &apiRes); err != nil {
		return "", nil, 0, err
	}

	if apiRes.Error != nil && apiRes.Error.Message != "" {
		return "", nil, 0, fmt.Errorf("anthropic error: %s", apiRes.Error.Message)
	}

	if len(apiRes.Content) == 0 {
		return "", nil, 0, fmt.Errorf("empty anthropic content")
	}

	var tokens int64
	if apiRes.Usage != nil {
		tokens = int64(apiRes.Usage.InputTokens + apiRes.Usage.OutputTokens)
	}

	var textParts []string
	var toolCalls []ToolCall

	for _, item := range apiRes.Content {
		if item.Type == "text" {
			textParts = append(textParts, item.Text)
		} else if item.Type == "tool_use" {
			toolCalls = append(toolCalls, ToolCall{
				ID:   item.ID,
				Name: item.Name,
				Args: string(item.Input),
			})
		}
	}

	var text string
	if len(textParts) > 0 {
		text = textParts[0] // or join them if multiple text blocks return
	}

	return text, toolCalls, tokens, nil
}
