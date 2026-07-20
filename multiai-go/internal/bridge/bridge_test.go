package bridge

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── request translation ─────────────────────────────────────────────────────

func TestToOpenAI_SystemAndText(t *testing.T) {
	req := &aRequest{
		Model:     "z-ai/glm-5.2",
		MaxTokens: 100,
		System:    json.RawMessage(`"tu es concis"`),
		Messages: []aMessage{
			{Role: "user", Content: json.RawMessage(`"salut"`)},
		},
	}
	out, err := toOpenAI(req, 32768)
	if err != nil {
		t.Fatalf("toOpenAI: %v", err)
	}
	if len(out.Messages) != 2 || out.Messages[0].Role != "system" || out.Messages[0].Content != "tu es concis" {
		t.Errorf("system message wrong: %+v", out.Messages)
	}
	if out.Messages[1].Role != "user" || out.Messages[1].Content != "salut" {
		t.Errorf("user message wrong: %+v", out.Messages[1])
	}
	if out.MaxTokens != 100 {
		t.Errorf("max_tokens = %d, want 100", out.MaxTokens)
	}
}

func TestToOpenAI_SystemBlocksAndCap(t *testing.T) {
	req := &aRequest{
		Model:     "m",
		MaxTokens: 999999,
		System:    json.RawMessage(`[{"type":"text","text":"a"},{"type":"text","text":"b"}]`),
		Messages:  []aMessage{{Role: "user", Content: json.RawMessage(`"x"`)}},
	}
	out, err := toOpenAI(req, 32768)
	if err != nil {
		t.Fatalf("toOpenAI: %v", err)
	}
	if out.Messages[0].Content != "a\n\nb" {
		t.Errorf("system blocks = %q", out.Messages[0].Content)
	}
	if out.MaxTokens != 32768 {
		t.Errorf("max_tokens not capped: %d", out.MaxTokens)
	}
}

func TestToOpenAI_ToolRoundTrip(t *testing.T) {
	req := &aRequest{
		Model:     "m",
		MaxTokens: 10,
		Messages: []aMessage{
			{Role: "user", Content: json.RawMessage(`"liste les fichiers"`)},
			{Role: "assistant", Content: json.RawMessage(`[{"type":"text","text":"ok"},{"type":"tool_use","id":"toolu_1","name":"ls","input":{"dir":"/tmp"}}]`)},
			{Role: "user", Content: json.RawMessage(`[{"type":"tool_result","tool_use_id":"toolu_1","content":"a.txt\nb.txt"},{"type":"text","text":"continue"}]`)},
		},
		Tools: []aTool{{Name: "ls", Description: "liste", InputSchema: json.RawMessage(`{"type":"object","properties":{"dir":{"type":"string"}}}`)}},
	}
	out, err := toOpenAI(req, 0)
	if err != nil {
		t.Fatalf("toOpenAI: %v", err)
	}
	// user, assistant(+tool_calls), tool, user
	if len(out.Messages) != 4 {
		t.Fatalf("messages = %d, want 4: %+v", len(out.Messages), out.Messages)
	}
	asst := out.Messages[1]
	if asst.Role != "assistant" || len(asst.ToolCalls) != 1 || asst.ToolCalls[0].Function.Name != "ls" {
		t.Errorf("assistant tool_calls wrong: %+v", asst)
	}
	if asst.ToolCalls[0].Function.Arguments != `{"dir":"/tmp"}` {
		t.Errorf("arguments = %q", asst.ToolCalls[0].Function.Arguments)
	}
	toolMsg := out.Messages[2]
	if toolMsg.Role != "tool" || toolMsg.ToolCallID != "toolu_1" || toolMsg.Content != "a.txt\nb.txt" {
		t.Errorf("tool message wrong: %+v", toolMsg)
	}
	if out.Messages[3].Role != "user" || out.Messages[3].Content != "continue" {
		t.Errorf("trailing user text wrong: %+v", out.Messages[3])
	}
	if len(out.Tools) != 1 || out.Tools[0].Function.Name != "ls" || out.Tools[0].Type != "function" {
		t.Errorf("tools wrong: %+v", out.Tools)
	}
}

func TestToOpenAI_ToolChoice(t *testing.T) {
	tests := []struct {
		raw  string
		want string // JSON of expected value
	}{
		{`{"type":"auto"}`, `"auto"`},
		{`{"type":"any"}`, `"required"`},
		{`{"type":"tool","name":"ls"}`, `{"function":{"name":"ls"},"type":"function"}`},
	}
	for _, tt := range tests {
		got := convertToolChoice(json.RawMessage(tt.raw))
		data, _ := json.Marshal(got)
		if string(data) != tt.want {
			t.Errorf("tool_choice %s -> %s, want %s", tt.raw, data, tt.want)
		}
	}
}

func TestFromOpenAI_TextToolReasoning(t *testing.T) {
	resp := &oResponse{
		ID: "abc",
		Choices: []oChoice{{
			FinishReason: "tool_calls",
			Message: &oRespMsg{
				Content:          "je lance ls",
				ReasoningContent: "il faut lister",
				ToolCalls:        []oToolCall{{ID: "call_1", Function: oFunction{Name: "ls", Arguments: `{"dir":"."}`}}},
			},
		}},
		Usage: &oUsage{PromptTokens: 11, CompletionTokens: 7},
	}
	out := fromOpenAI(resp, "z-ai/glm-5.2")
	content := out["content"].([]any)
	if len(content) != 3 {
		t.Fatalf("content blocks = %d, want 3 (thinking, text, tool_use)", len(content))
	}
	if content[0].(map[string]any)["type"] != "thinking" {
		t.Errorf("block 0 = %v", content[0])
	}
	if content[1].(map[string]any)["text"] != "je lance ls" {
		t.Errorf("block 1 = %v", content[1])
	}
	tu := content[2].(map[string]any)
	if tu["type"] != "tool_use" || tu["name"] != "ls" || tu["input"].(map[string]any)["dir"] != "." {
		t.Errorf("block 2 = %v", tu)
	}
	if out["stop_reason"] != "tool_use" {
		t.Errorf("stop_reason = %v", out["stop_reason"])
	}
	if u := out["usage"].(aUsage); u.InputTokens != 11 || u.OutputTokens != 7 {
		t.Errorf("usage = %+v", u)
	}
}

// ── stream translation ──────────────────────────────────────────────────────

// sseChunk builds one backend SSE line.
func sseChunk(t *testing.T, chunk any) string {
	t.Helper()
	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatal(err)
	}
	return "data: " + string(data) + "\n\n"
}

type sseEvent struct {
	Name string
	Data map[string]any
}

func parseAnthropicSSE(t *testing.T, raw string) []sseEvent {
	t.Helper()
	var events []sseEvent
	var current sseEvent
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			current = sseEvent{Name: strings.TrimPrefix(line, "event: ")}
		case strings.HasPrefix(line, "data: "):
			var d map[string]any
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &d); err != nil {
				t.Fatalf("bad event data: %v", err)
			}
			current.Data = d
			events = append(events, current)
		}
	}
	return events
}

func TestTranslateStream_FullSequence(t *testing.T) {
	idx0, idx1 := 0, 1
	var backend strings.Builder
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ReasoningContent: "reflexion"}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{Content: "Bon"}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{Content: "jour"}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{{Index: &idx0, ID: "call_a", Function: oFunction{Name: "ls", Arguments: `{"d`}}}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{{Index: &idx0, Function: oFunction{Arguments: `ir":"."}`}}}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{{Index: &idx1, ID: "call_b", Function: oFunction{Name: "cat", Arguments: `{}`}}}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{FinishReason: "tool_calls"}}, Usage: &oUsage{PromptTokens: 5, CompletionTokens: 9}}))
	backend.WriteString("data: [DONE]\n\n")

	var out bytes.Buffer
	if err := translateStream(&out, nil, strings.NewReader(backend.String()), "z-ai/glm-5.2"); err != nil {
		t.Fatalf("translateStream: %v", err)
	}
	events := parseAnthropicSSE(t, out.String())

	var names []string
	for _, e := range events {
		names = append(names, e.Name)
	}
	want := []string{
		"message_start",
		"content_block_start", "content_block_delta", "content_block_stop", // thinking
		"content_block_start", "content_block_delta", "content_block_delta", "content_block_stop", // text
		"content_block_start", "content_block_delta", "content_block_delta", "content_block_stop", // tool ls
		"content_block_start", "content_block_delta", "content_block_stop", // tool cat
		"message_delta", "message_stop",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("event sequence:\n got %v\nwant %v", names, want)
	}

	// Thinking block content.
	if d := events[2].Data["delta"].(map[string]any); d["thinking"] != "reflexion" {
		t.Errorf("thinking delta = %v", d)
	}
	// Tool block start carries id + name; deltas carry partial_json.
	start := events[8].Data["content_block"].(map[string]any)
	if start["id"] != "call_a" || start["name"] != "ls" {
		t.Errorf("tool block start = %v", start)
	}
	pj1 := events[9].Data["delta"].(map[string]any)["partial_json"].(string)
	pj2 := events[10].Data["delta"].(map[string]any)["partial_json"].(string)
	if pj1+pj2 != `{"dir":"."}` {
		t.Errorf("partial_json = %q + %q", pj1, pj2)
	}
	// Final message_delta: stop_reason + usage.
	md := events[len(events)-2].Data
	if md["delta"].(map[string]any)["stop_reason"] != "tool_use" {
		t.Errorf("stop_reason = %v", md)
	}
	if md["usage"].(map[string]any)["output_tokens"].(float64) != 9 {
		t.Errorf("usage = %v", md["usage"])
	}
	// Indexes must be sequential per block.
	if events[1].Data["index"].(float64) != 0 || events[4].Data["index"].(float64) != 1 || events[8].Data["index"].(float64) != 2 || events[12].Data["index"].(float64) != 3 {
		t.Errorf("block indexes not sequential")
	}
}

func TestTranslateStream_EmptyBackend(t *testing.T) {
	var out bytes.Buffer
	if err := translateStream(&out, nil, strings.NewReader("data: [DONE]\n\n"), "m"); err != nil {
		t.Fatalf("translateStream: %v", err)
	}
	events := parseAnthropicSSE(t, out.String())
	var names []string
	for _, e := range events {
		names = append(names, e.Name)
	}
	if strings.Join(names, ",") != "message_start,message_delta,message_stop" {
		t.Errorf("empty stream sequence = %v", names)
	}
}

func TestToOpenAI_StreamRequestsUsage(t *testing.T) {
	req := &aRequest{Model: "m", MaxTokens: 8, Stream: true,
		Messages: []aMessage{{Role: "user", Content: json.RawMessage(`"x"`)}}}
	out, err := toOpenAI(req, 0)
	if err != nil {
		t.Fatalf("toOpenAI: %v", err)
	}
	if out.StreamOpts == nil || !out.StreamOpts.IncludeUsage {
		t.Error("streaming request must set stream_options.include_usage")
	}
	req.Stream = false
	out, _ = toOpenAI(req, 0)
	if out.StreamOpts != nil {
		t.Error("non-streaming request must not set stream_options")
	}
}

func TestToOpenAI_ToolResultErrorFlag(t *testing.T) {
	req := &aRequest{Model: "m", MaxTokens: 8, Messages: []aMessage{
		{Role: "user", Content: json.RawMessage(`[{"type":"tool_result","tool_use_id":"t1","is_error":true,"content":"boom"}]`)},
	}}
	out, err := toOpenAI(req, 0)
	if err != nil {
		t.Fatalf("toOpenAI: %v", err)
	}
	if out.Messages[0].Content != "[tool error] boom" {
		t.Errorf("is_error not encoded: %q", out.Messages[0].Content)
	}
}

func TestFromOpenAI_ToolCallsWithFinishStop(t *testing.T) {
	resp := &oResponse{Choices: []oChoice{{
		FinishReason: "stop", // quirky backend: tool calls but finish "stop"
		Message:      &oRespMsg{ToolCalls: []oToolCall{{ID: "c1", Function: oFunction{Name: "ls", Arguments: "{}"}}}},
	}}}
	out := fromOpenAI(resp, "m")
	if out["stop_reason"] != "tool_use" {
		t.Errorf("stop_reason = %v, want tool_use (derived from tool calls)", out["stop_reason"])
	}
}

func TestTranslateStream_NoIndexParallelToolCalls(t *testing.T) {
	// Backend sends two complete tool calls in one delta, without index.
	var backend strings.Builder
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{
		{ID: "call_a", Function: oFunction{Name: "ls", Arguments: `{"d":1}`}},
		{ID: "call_b", Function: oFunction{Name: "cat", Arguments: `{"f":2}`}},
	}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{FinishReason: "tool_calls"}}}))
	backend.WriteString("data: [DONE]\n\n")

	var out bytes.Buffer
	if err := translateStream(&out, nil, strings.NewReader(backend.String()), "m"); err != nil {
		t.Fatalf("translateStream: %v", err)
	}
	events := parseAnthropicSSE(t, out.String())
	var starts []string
	for _, e := range events {
		if e.Name == "content_block_start" {
			starts = append(starts, e.Data["content_block"].(map[string]any)["name"].(string))
		}
	}
	if len(starts) != 2 || starts[0] != "ls" || starts[1] != "cat" {
		t.Fatalf("expected two distinct tool blocks (ls, cat), got %v", starts)
	}
}

func TestTranslateStream_LateToolName(t *testing.T) {
	idx0 := 0
	var backend strings.Builder
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{{Index: &idx0, ID: "call_x"}}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{{Index: &idx0, Function: oFunction{Name: "Read"}}}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{{Index: &idx0, Function: oFunction{Arguments: `{"p":"x"}`}}}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{FinishReason: "tool_calls"}}}))
	backend.WriteString("data: [DONE]\n\n")

	var out bytes.Buffer
	if err := translateStream(&out, nil, strings.NewReader(backend.String()), "m"); err != nil {
		t.Fatalf("translateStream: %v", err)
	}
	events := parseAnthropicSSE(t, out.String())
	for _, e := range events {
		if e.Name == "content_block_start" {
			cb := e.Data["content_block"].(map[string]any)
			if cb["name"] != "Read" || cb["id"] != "call_x" {
				t.Fatalf("tool block start lost late metadata: %v", cb)
			}
			return
		}
	}
	t.Fatal("no content_block_start emitted")
}

func TestTranslateStream_ToolBlockForcesToolUseStop(t *testing.T) {
	idx0 := 0
	var backend strings.Builder
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{ToolCalls: []oToolCall{{Index: &idx0, ID: "c", Function: oFunction{Name: "ls", Arguments: "{}"}}}}}}}))
	backend.WriteString(sseChunk(t, oResponse{Choices: []oChoice{{FinishReason: "stop"}}})) // quirky backend
	backend.WriteString("data: [DONE]\n\n")

	var out bytes.Buffer
	_ = translateStream(&out, nil, strings.NewReader(backend.String()), "m")
	events := parseAnthropicSSE(t, out.String())
	for _, e := range events {
		if e.Name == "message_delta" {
			if sr := e.Data["delta"].(map[string]any)["stop_reason"]; sr != "tool_use" {
				t.Fatalf("stop_reason = %v, want tool_use", sr)
			}
			return
		}
	}
	t.Fatal("no message_delta emitted")
}

func TestTranslateStream_BackendErrorPayload(t *testing.T) {
	backend := "data: {\"error\":{\"message\":\"quota exceeded\",\"code\":429}}\n\n"
	var out bytes.Buffer
	if err := translateStream(&out, nil, strings.NewReader(backend), "m"); err != nil {
		t.Fatalf("translateStream: %v", err)
	}
	events := parseAnthropicSSE(t, out.String())
	last := events[len(events)-1]
	if last.Name != "error" {
		t.Fatalf("last event = %s, want error (events: %v)", last.Name, events)
	}
	if !strings.Contains(last.Data["error"].(map[string]any)["message"].(string), "quota exceeded") {
		t.Errorf("error message lost: %v", last.Data)
	}
	for _, e := range events {
		if e.Name == "message_stop" {
			t.Error("message_stop must not follow an error event")
		}
	}
}

func TestTranslateStream_TruncatedStreamIsError(t *testing.T) {
	// Stream dies mid-generation: no finish_reason, no [DONE].
	backend := sseChunk(t, oResponse{Choices: []oChoice{{Delta: &oRespMsg{Content: "partial"}}}})
	var out bytes.Buffer
	if err := translateStream(&out, nil, strings.NewReader(backend), "m"); err != nil {
		t.Fatalf("translateStream: %v", err)
	}
	events := parseAnthropicSSE(t, out.String())
	last := events[len(events)-1]
	if last.Name != "error" {
		t.Fatalf("truncated stream must end with an error event, got %s", last.Name)
	}
}

// ── server end-to-end (fake backend) ────────────────────────────────────────

func newFakeBackend(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"z-ai/glm-5.2","object":"model","created":1,"owned_by":"z-ai"}]}`))
		case "/chat/completions":
			if r.Header.Get("Authorization") != "Bearer nvapi-test" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"bad key"}`))
				return
			}
			body, _ := io.ReadAll(r.Body)
			var req oRequest
			_ = json.Unmarshal(body, &req)
			if req.Stream {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"pong\"}}]}\n\n")
				fmt.Fprint(w, "data: {\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1}}\n\n")
				fmt.Fprint(w, "data: [DONE]\n\n")
				return
			}
			_, _ = w.Write([]byte(`{"id":"x","choices":[{"message":{"role":"assistant","content":"pong"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":1}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func startTestBridge(t *testing.T, backendURL string) *Server {
	t.Helper()
	srv, err := Start(Config{Target: backendURL, APIKey: "nvapi-test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(srv.Shutdown)
	return srv
}

func TestServerNonStreaming(t *testing.T) {
	backend := newFakeBackend(t)
	defer backend.Close()
	srv := startTestBridge(t, backend.URL)

	resp, err := http.Post(srv.URL()+"/v1/messages", "application/json",
		strings.NewReader(`{"model":"z-ai/glm-5.2","max_tokens":32,"messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var doc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["type"] != "message" || doc["role"] != "assistant" || doc["stop_reason"] != "end_turn" {
		t.Errorf("envelope wrong: %v", doc)
	}
	text := doc["content"].([]any)[0].(map[string]any)["text"]
	if text != "pong" {
		t.Errorf("text = %v", text)
	}
}

func TestServerStreaming(t *testing.T) {
	backend := newFakeBackend(t)
	defer backend.Close()
	srv := startTestBridge(t, backend.URL)

	resp, err := http.Post(srv.URL()+"/v1/messages", "application/json",
		strings.NewReader(`{"model":"z-ai/glm-5.2","max_tokens":32,"stream":true,"messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %s", ct)
	}
	raw, _ := io.ReadAll(resp.Body)
	events := parseAnthropicSSE(t, string(raw))
	var names []string
	for _, e := range events {
		names = append(names, e.Name)
	}
	want := "message_start,content_block_start,content_block_delta,content_block_stop,message_delta,message_stop"
	if strings.Join(names, ",") != want {
		t.Fatalf("sequence = %v", names)
	}
}

func TestServerBackendErrorMapped(t *testing.T) {
	backend := newFakeBackend(t)
	defer backend.Close()
	srv, err := Start(Config{Target: backend.URL, APIKey: "wrong-key"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Shutdown()

	resp, err := http.Post(srv.URL()+"/v1/messages", "application/json",
		strings.NewReader(`{"model":"m","max_tokens":8,"messages":[{"role":"user","content":"x"}]}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	var doc map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&doc)
	if doc["type"] != "error" || doc["error"].(map[string]any)["type"] != "authentication_error" {
		t.Errorf("error doc = %v", doc)
	}
}

func TestServerCountTokens(t *testing.T) {
	backend := newFakeBackend(t)
	defer backend.Close()
	srv := startTestBridge(t, backend.URL)

	resp, err := http.Post(srv.URL()+"/v1/messages/count_tokens", "application/json",
		strings.NewReader(`{"model":"m","messages":[{"role":"user","content":"`+strings.Repeat("a", 400)+`"}]}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	var doc map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&doc)
	n, ok := doc["input_tokens"].(float64)
	if !ok || n < 50 {
		t.Errorf("input_tokens = %v", doc)
	}
}

func TestServerModelsList(t *testing.T) {
	backend := newFakeBackend(t)
	defer backend.Close()
	srv := startTestBridge(t, backend.URL)

	resp, err := http.Get(srv.URL() + "/v1/models")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	var doc struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(doc.Data) != 1 || doc.Data[0]["id"] != "z-ai/glm-5.2" || doc.Data[0]["type"] != "model" {
		t.Errorf("models = %v", doc.Data)
	}
}

func TestStartRejectsBadTarget(t *testing.T) {
	if _, err := Start(Config{Target: "ftp://nope", APIKey: "k"}); err == nil {
		t.Error("expected error for non-http target")
	}
}
