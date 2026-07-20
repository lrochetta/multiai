package bridge

import (
	"encoding/json"
	"fmt"
	"strings"
)

// toOpenAI converts an Anthropic Messages request into an OpenAI
// chat/completions request. maxTokensCap clamps max_tokens when > 0
// (the NVIDIA hosted endpoints reject requests above their per-model cap).
func toOpenAI(req *aRequest, maxTokensCap int) (*oRequest, error) {
	out := &oRequest{
		Model:       req.Model,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.StopSequences,
		Stream:      req.Stream,
	}
	if req.Stream {
		// Without include_usage, spec-compliant backends omit the final
		// usage chunk and the client would see 0/0 tokens forever.
		out.StreamOpts = &oStreamOptions{IncludeUsage: true}
	}
	if req.Metadata != nil {
		out.User = req.Metadata.UserID
	}

	out.MaxTokens = req.MaxTokens
	if maxTokensCap > 0 && (out.MaxTokens <= 0 || out.MaxTokens > maxTokensCap) {
		out.MaxTokens = maxTokensCap
	}

	if sys := flattenSystem(req.System); sys != "" {
		out.Messages = append(out.Messages, oMessage{Role: "system", Content: sys})
	}

	for i := range req.Messages {
		msgs, err := convertMessage(&req.Messages[i])
		if err != nil {
			return nil, fmt.Errorf("message %d: %w", i, err)
		}
		out.Messages = append(out.Messages, msgs...)
	}

	for _, t := range req.Tools {
		params := t.InputSchema
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		out.Tools = append(out.Tools, oTool{
			Type:     "function",
			Function: oToolDef{Name: t.Name, Description: t.Description, Parameters: params},
		})
	}

	if tc := convertToolChoice(req.ToolChoice); tc != nil {
		out.ToolChoice = tc
	}
	return out, nil
}

// flattenSystem accepts the Anthropic system field as a plain string or a
// list of text blocks and returns the concatenated text.
func flattenSystem(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []aBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n\n")
	}
	return ""
}

// convertMessage expands one Anthropic message into one or more OpenAI
// messages. tool_result blocks become role:"tool" messages (emitted first,
// as they answer the PREVIOUS assistant tool_calls); the remaining text
// becomes a user/assistant message. Assistant tool_use blocks become
// tool_calls. Thinking blocks are dropped (backends resend reasoning on
// their own); images are replaced by a placeholder since the NVIDIA text
// endpoints do not accept image parts.
func convertMessage(m *aMessage) ([]oMessage, error) {
	// Plain string content: pass through.
	var plain string
	if err := json.Unmarshal(m.Content, &plain); err == nil {
		return []oMessage{{Role: m.Role, Content: plain}}, nil
	}

	var blocks []aBlock
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		return nil, fmt.Errorf("contenu illisible (ni texte ni blocs): %w", err)
	}

	var out []oMessage
	var texts []string
	var toolCalls []oToolCall

	for _, b := range blocks {
		switch b.Type {
		case "text":
			texts = append(texts, b.Text)
		case "image":
			texts = append(texts, "[image non transmise par le pont multiai]")
		case "thinking", "redacted_thinking":
			// dropped on purpose
		case "tool_use":
			args := "{}"
			if len(b.Input) > 0 {
				args = string(b.Input)
			}
			toolCalls = append(toolCalls, oToolCall{
				ID:       b.ID,
				Type:     "function",
				Function: oFunction{Name: b.Name, Arguments: args},
			})
		case "tool_result":
			content := flattenToolResult(b.Content)
			if b.IsError {
				// OpenAI tool messages have no error flag: encode it in the
				// text so the backend model knows the call failed.
				content = "[tool error] " + content
			}
			out = append(out, oMessage{
				Role:       "tool",
				ToolCallID: b.ToolUseID,
				Content:    content,
			})
		default:
			// Unknown block types are dropped rather than failing the call.
		}
	}

	text := strings.Join(texts, "\n")
	if m.Role == "assistant" {
		if text != "" || len(toolCalls) > 0 {
			out = append(out, oMessage{Role: "assistant", Content: text, ToolCalls: toolCalls})
		}
		return out, nil
	}
	if text != "" {
		out = append(out, oMessage{Role: "user", Content: text})
	}
	return out, nil
}

// flattenToolResult renders a tool_result payload (string or block list)
// as plain text for the OpenAI tool message.
func flattenToolResult(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var blocks []aBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		var parts []string
		for _, b := range blocks {
			if b.Type == "text" {
				parts = append(parts, b.Text)
			}
		}
		return strings.Join(parts, "\n")
	}
	return string(raw)
}

// convertToolChoice maps the Anthropic tool_choice object to its OpenAI
// counterpart; nil means "omit the field".
func convertToolChoice(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var tc struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &tc); err != nil {
		return nil
	}
	switch tc.Type {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "none":
		return "none"
	case "tool":
		return map[string]any{
			"type":     "function",
			"function": map[string]any{"name": tc.Name},
		}
	}
	return nil
}

// mapStopReason converts an OpenAI finish_reason to an Anthropic stop_reason.
func mapStopReason(finish string) string {
	switch finish {
	case "tool_calls", "function_call":
		return "tool_use"
	case "length":
		return "max_tokens"
	case "stop", "content_filter", "":
		return "end_turn"
	}
	return "end_turn"
}

// fromOpenAI converts a non-streaming OpenAI response into the Anthropic
// Messages response document (as generic maps, so block shapes stay exact).
func fromOpenAI(resp *oResponse, model string) map[string]any {
	content := []any{}
	stop := "end_turn"
	if len(resp.Choices) > 0 {
		ch := resp.Choices[0]
		stop = mapStopReason(ch.FinishReason)
		// Several OpenAI-compatible backends report finish_reason "stop"
		// even when tool calls are present; agent clients key the tool
		// loop on stop_reason "tool_use", so derive it from the content.
		if ch.Message != nil && len(ch.Message.ToolCalls) > 0 && stop == "end_turn" {
			stop = "tool_use"
		}
		if ch.Message != nil {
			if ch.Message.ReasoningContent != "" {
				content = append(content, map[string]any{
					"type":     "thinking",
					"thinking": ch.Message.ReasoningContent,
					// A backend reasoning trace has no Anthropic signature;
					// an empty one keeps the block schema-valid for display.
					"signature": "",
				})
			}
			if ch.Message.Content != "" {
				content = append(content, map[string]any{"type": "text", "text": ch.Message.Content})
			}
			for i, tc := range ch.Message.ToolCalls {
				content = append(content, map[string]any{
					"type":  "tool_use",
					"id":    toolCallID(tc.ID, i),
					"name":  tc.Function.Name,
					"input": parseToolArgs(tc.Function.Arguments),
				})
			}
		}
	}

	usage := aUsage{}
	if resp.Usage != nil {
		usage.InputTokens = resp.Usage.PromptTokens
		usage.OutputTokens = resp.Usage.CompletionTokens
	}

	id := resp.ID
	if id == "" {
		id = "msg_bridge"
	} else {
		id = "msg_" + id
	}

	return map[string]any{
		"id":            id,
		"type":          "message",
		"role":          "assistant",
		"model":         model,
		"content":       content,
		"stop_reason":   stop,
		"stop_sequence": nil,
		"usage":         usage,
	}
}

// parseToolArgs decodes an OpenAI arguments JSON string into an object for
// the Anthropic input field; malformed payloads are preserved under _raw so
// the client still sees what the model produced.
func parseToolArgs(args string) map[string]any {
	if strings.TrimSpace(args) == "" {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(args), &m); err != nil || m == nil {
		return map[string]any{"_raw": args}
	}
	return m
}

// toolCallID guarantees a non-empty tool_use id (some backends omit ids in
// tool call deltas).
func toolCallID(id string, index int) string {
	if id != "" {
		return id
	}
	return fmt.Sprintf("toolu_bridge_%d", index)
}
