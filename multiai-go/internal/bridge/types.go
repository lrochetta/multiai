package bridge

import "encoding/json"

// ── Anthropic Messages API wire types (subset used by Claude Code) ──────────

// aRequest mirrors POST /v1/messages.
type aRequest struct {
	Model         string          `json:"model"`
	MaxTokens     int             `json:"max_tokens"`
	Messages      []aMessage      `json:"messages"`
	System        json.RawMessage `json:"system,omitempty"` // string or []block
	Temperature   *float64        `json:"temperature,omitempty"`
	TopP          *float64        `json:"top_p,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	Tools         []aTool         `json:"tools,omitempty"`
	ToolChoice    json.RawMessage `json:"tool_choice,omitempty"`
	Metadata      *aMetadata      `json:"metadata,omitempty"`
}

type aMetadata struct {
	UserID string `json:"user_id,omitempty"`
}

type aMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // string or []aBlock
}

// aBlock is one content block of a request message. Fields overlap across
// block types (text, image, tool_use, tool_result, thinking); Type selects
// which are meaningful.
type aBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"` // tool_result payload: string or []aBlock
	IsError   bool            `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
}

type aTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema,omitempty"`
}

// aUsage is the Anthropic usage envelope.
type aUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ── OpenAI chat/completions wire types (subset) ─────────────────────────────

type oRequest struct {
	Model       string          `json:"model"`
	Messages    []oMessage      `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	StreamOpts  *oStreamOptions `json:"stream_options,omitempty"`
	Tools       []oTool         `json:"tools,omitempty"`
	ToolChoice  any             `json:"tool_choice,omitempty"`
	User        string          `json:"user,omitempty"`
}

// oStreamOptions requests the final usage chunk in streaming mode; without
// it, spec-compliant OpenAI backends omit usage entirely.
type oStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type oMessage struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []oToolCall `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

type oToolCall struct {
	// Index identifies the call in streaming deltas; pointer so the field
	// is absent from non-streaming requests we build.
	Index    *int      `json:"index,omitempty"`
	ID       string    `json:"id,omitempty"`
	Type     string    `json:"type,omitempty"`
	Function oFunction `json:"function"`
}

type oFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type oTool struct {
	Type     string   `json:"type"` // "function"
	Function oToolDef `json:"function"`
}

type oToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

// oResponse covers both the full response and one streaming chunk.
type oResponse struct {
	ID      string    `json:"id"`
	Model   string    `json:"model"`
	Choices []oChoice `json:"choices"`
	Usage   *oUsage   `json:"usage,omitempty"`
}

type oChoice struct {
	Index        int       `json:"index"`
	Message      *oRespMsg `json:"message,omitempty"` // non-streaming
	Delta        *oRespMsg `json:"delta,omitempty"`   // streaming
	FinishReason string    `json:"finish_reason,omitempty"`
}

type oRespMsg struct {
	Role             string      `json:"role,omitempty"`
	Content          string      `json:"content,omitempty"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
	ToolCalls        []oToolCall `json:"tool_calls,omitempty"`
}

type oUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}
