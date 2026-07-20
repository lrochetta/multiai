package bridge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// streamState drives the OpenAI-SSE -> Anthropic-SSE translation. Anthropic
// content blocks are strictly sequential (start, deltas, stop, next start),
// so the translator closes the current block whenever the backend switches
// between reasoning, text and tool-call deltas.
//
// Tool-call blocks are opened lazily: some OpenAI-compatible backends split
// id and function.name across deltas, and the Anthropic content_block_start
// must carry both. Metadata is accumulated in pendTool* until the first
// argument fragment (or the end of the call) forces the block out.
type streamState struct {
	w       io.Writer
	flusher http.Flusher

	started    bool   // message_start emitted
	blockIndex int    // index of the open block; -1 when none
	blockType  string // "", "thinking", "text", "tool"

	// Current tool call identity. toolIdxKnown marks whether the backend
	// provided a numeric index; currentToolID holds the last seen id.
	openaiToolIdx int
	toolIdxKnown  bool
	currentToolID string
	toolStarted   bool // content_block_start emitted for the current tool
	pendToolID    string
	pendToolName  string

	nextIndex    int
	model        string
	stopReason   string
	sawFinish    bool // a finish_reason chunk arrived
	sawToolBlock bool
	errored      bool // an error event was emitted; suppress finish()
	usage        *oUsage
	inputTokens  int
}

// translateStream reads an OpenAI SSE body and writes the Anthropic SSE
// event sequence. A stream that ends without [DONE] and without any
// finish_reason — or with a read error, or with an in-stream error payload —
// is reported to the client as an Anthropic `error` event instead of being
// silently finalized as a complete answer.
func translateStream(w io.Writer, flusher http.Flusher, body io.Reader, model string) error {
	st := &streamState{w: w, flusher: flusher, blockIndex: -1, model: model}

	sawDone := false
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			sawDone = true
			break
		}
		// In-stream backend error payloads ({"error": {...}}) must surface
		// as Anthropic error events, not vanish as unparseable chunks.
		var errChk struct {
			Error json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal([]byte(payload), &errChk); err == nil &&
			len(errChk.Error) > 0 && string(errChk.Error) != "null" {
			st.emitError("api_error", "erreur backend en cours de flux: "+backendErrMessage(errChk.Error))
			return nil
		}
		var chunk oResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue // tolerate malformed keep-alive lines
		}
		st.consume(&chunk)
	}

	if err := scanner.Err(); err != nil {
		st.emitError("api_error", "flux backend interrompu: "+err.Error())
		return err
	}
	if !sawDone && !st.sawFinish {
		// Backend closed the connection mid-generation: the answer is
		// truncated, never pretend it completed.
		st.emitError("api_error", "flux backend termine prematurement (reponse tronquee)")
		return nil
	}
	st.finish()
	return nil
}

// backendErrMessage extracts a human message from a backend error payload.
func backendErrMessage(raw json.RawMessage) string {
	var obj struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Message != "" {
		return obj.Message
	}
	s := string(raw)
	if len(s) > 512 {
		s = s[:512]
	}
	return s
}

// consume applies one backend chunk to the state machine.
func (st *streamState) consume(chunk *oResponse) {
	if chunk.Usage != nil {
		st.usage = chunk.Usage
	}
	if len(chunk.Choices) == 0 {
		return
	}
	ch := chunk.Choices[0]
	if ch.FinishReason != "" {
		st.sawFinish = true
		st.stopReason = mapStopReason(ch.FinishReason)
	}
	d := ch.Delta
	if d == nil {
		return
	}

	st.ensureMessageStart()

	if d.ReasoningContent != "" {
		st.ensureSimpleBlock("thinking", map[string]any{"type": "thinking", "thinking": "", "signature": ""})
		st.event("content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": st.blockIndex,
			"delta": map[string]any{"type": "thinking_delta", "thinking": d.ReasoningContent},
		})
	}

	if d.Content != "" {
		st.ensureSimpleBlock("text", map[string]any{"type": "text", "text": ""})
		st.event("content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": st.blockIndex,
			"delta": map[string]any{"type": "text_delta", "text": d.Content},
		})
	}

	for i := range d.ToolCalls {
		st.consumeToolDelta(&d.ToolCalls[i])
	}
}

// consumeToolDelta routes one tool_calls delta entry: it either starts a
// new tool_use block or continues the current one, merging late id/name
// metadata and streaming argument fragments.
func (st *streamState) consumeToolDelta(tc *oToolCall) {
	if st.isNewToolCall(tc) {
		st.closeBlock()
		st.blockType = "tool"
		st.blockIndex = st.nextIndex
		st.nextIndex++
		st.sawToolBlock = true
		st.toolStarted = false
		st.pendToolID = tc.ID
		st.pendToolName = tc.Function.Name
		st.currentToolID = tc.ID
		st.toolIdxKnown = tc.Index != nil
		if tc.Index != nil {
			st.openaiToolIdx = *tc.Index
		}
	} else if !st.toolStarted {
		// Late metadata for a block not yet emitted.
		if tc.ID != "" {
			st.pendToolID = tc.ID
			st.currentToolID = tc.ID
		}
		if tc.Function.Name != "" {
			st.pendToolName = tc.Function.Name
		}
		if tc.Index != nil {
			st.openaiToolIdx = *tc.Index
			st.toolIdxKnown = true
		}
	}

	if tc.Function.Arguments != "" {
		st.openToolBlock()
		st.event("content_block_delta", map[string]any{
			"type":  "content_block_delta",
			"index": st.blockIndex,
			"delta": map[string]any{"type": "input_json_delta", "partial_json": tc.Function.Arguments},
		})
	}
}

// isNewToolCall decides whether a tool_calls delta entry belongs to the
// current tool block or starts a new call. Backends with numeric indexes
// are distinguished by index; index-less backends by a changing id.
func (st *streamState) isNewToolCall(tc *oToolCall) bool {
	if st.blockType != "tool" {
		return true
	}
	if tc.Index != nil && st.toolIdxKnown && *tc.Index != st.openaiToolIdx {
		return true
	}
	if tc.ID != "" && st.currentToolID != "" && tc.ID != st.currentToolID {
		return true
	}
	return false
}

// openToolBlock emits the deferred content_block_start for the current
// tool call, with whatever id/name metadata has been accumulated.
func (st *streamState) openToolBlock() {
	if st.toolStarted {
		return
	}
	st.toolStarted = true
	st.event("content_block_start", map[string]any{
		"type":  "content_block_start",
		"index": st.blockIndex,
		"content_block": map[string]any{
			"type":  "tool_use",
			"id":    toolCallID(st.pendToolID, st.blockIndex),
			"name":  st.pendToolName,
			"input": map[string]any{},
		},
	})
}

// ensureMessageStart emits message_start once.
func (st *streamState) ensureMessageStart() {
	if st.started {
		return
	}
	st.started = true
	st.event("message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            "msg_bridge_stream",
			"type":          "message",
			"role":          "assistant",
			"model":         st.model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         aUsage{InputTokens: st.inputTokens},
		},
	})
}

// ensureSimpleBlock switches the open content block to a text or thinking
// block, closing the previous one when needed.
func (st *streamState) ensureSimpleBlock(typ string, contentBlock map[string]any) {
	if st.blockType == typ {
		return
	}
	st.closeBlock()
	st.blockIndex = st.nextIndex
	st.nextIndex++
	st.blockType = typ
	st.event("content_block_start", map[string]any{
		"type":          "content_block_start",
		"index":         st.blockIndex,
		"content_block": contentBlock,
	})
}

// closeBlock emits content_block_stop for the open block, if any. A tool
// block whose start was still deferred (a call with no arguments, e.g. a
// zero-parameter tool) is flushed first so the client sees a valid pair.
func (st *streamState) closeBlock() {
	if st.blockIndex < 0 {
		return
	}
	if st.blockType == "tool" && !st.toolStarted {
		st.openToolBlock()
	}
	st.event("content_block_stop", map[string]any{
		"type":  "content_block_stop",
		"index": st.blockIndex,
	})
	st.blockIndex = -1
	st.blockType = ""
	st.toolIdxKnown = false
	st.currentToolID = ""
	st.toolStarted = false
	st.pendToolID = ""
	st.pendToolName = ""
}

// finish closes the message: last block stop, message_delta with the stop
// reason and usage, then message_stop. A message that streamed a tool_use
// block ends with stop_reason "tool_use" even when the backend reported
// finish_reason "stop" (a documented quirk of several OpenAI-compatible
// servers): agent clients key the tool-execution loop on this value.
func (st *streamState) finish() {
	if st.errored {
		return
	}
	st.ensureMessageStart() // empty backend stream: still emit a valid message
	st.closeBlock()
	stop := st.stopReason
	if stop == "" {
		stop = "end_turn"
	}
	if st.sawToolBlock && stop == "end_turn" {
		stop = "tool_use"
	}
	usage := aUsage{}
	if st.usage != nil {
		usage.InputTokens = st.usage.PromptTokens
		usage.OutputTokens = st.usage.CompletionTokens
	}
	st.event("message_delta", map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": stop, "stop_sequence": nil},
		"usage": usage,
	})
	st.event("message_stop", map[string]any{"type": "message_stop"})
}

// emitError sends an Anthropic error SSE event and marks the stream as
// terminated: no message_delta/message_stop will follow, so the client
// treats the turn as failed instead of truncated-but-complete.
func (st *streamState) emitError(typ, msg string) {
	st.errored = true
	st.event("error", map[string]any{
		"type":  "error",
		"error": map[string]any{"type": typ, "message": msg},
	})
}

// event writes one SSE event and flushes it immediately.
func (st *streamState) event(name string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(st.w, "event: %s\ndata: %s\n\n", name, data)
	if st.flusher != nil {
		st.flusher.Flush()
	}
}
