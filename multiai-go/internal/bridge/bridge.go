// Package bridge embeds a local Anthropic->OpenAI translation proxy in the
// multiai binary, so Claude Code can use OpenAI-compatible-only backends
// (NVIDIA build.nvidia.com in particular) without any external proxy.
//
// The server binds to loopback only, is started automatically by the
// launcher for profiles declaring BRIDGE=anthropic-openai, and is torn down
// when the child CLI exits. Endpoints served:
//
//	POST /v1/messages               Anthropic Messages (stream + non-stream)
//	POST /v1/messages/count_tokens  rough estimation (chars/4)
//	GET  /v1/models                 backend catalog, Anthropic list shape
package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DefaultNvidiaTarget is the hosted NVIDIA OpenAI-compatible endpoint.
const DefaultNvidiaTarget = "https://integrate.api.nvidia.com/v1"

// DefaultMaxTokensCap clamps max_tokens: the NVIDIA hosted endpoints reject
// values above the per-model cap (32768 for z-ai/glm-5.2).
const DefaultMaxTokensCap = 32768

// Config parameterizes a bridge instance.
type Config struct {
	Target       string // OpenAI-compatible base URL, must end with /v1
	APIKey       string // backend Bearer key (never exposed to the client side)
	Addr         string // listen address; default "127.0.0.1:0" (ephemeral)
	MaxTokensCap int    // 0 = DefaultMaxTokensCap; negative = no clamp
}

// Server is a running bridge.
type Server struct {
	cfg     Config
	ln      net.Listener
	httpSrv *http.Server
	client  *http.Client
}

// Start validates the config, binds the listener and serves in background.
func Start(cfg Config) (*Server, error) {
	cfg.Target = strings.TrimRight(cfg.Target, "/")
	if !strings.HasPrefix(cfg.Target, "http://") && !strings.HasPrefix(cfg.Target, "https://") {
		return nil, fmt.Errorf("cible du pont invalide (http(s) attendu): %q", cfg.Target)
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:0"
	}
	if cfg.MaxTokensCap == 0 {
		cfg.MaxTokensCap = DefaultMaxTokensCap
	}

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("pont: ecoute impossible sur %s: %w", cfg.Addr, err)
	}

	s := &Server{
		cfg: cfg,
		ln:  ln,
		// No global timeout: reasoning models on the free tier can queue for
		// minutes before the first token. The incoming request context still
		// cancels the backend call when the client gives up.
		client: &http.Client{Timeout: 0},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", s.handleMessages)
	mux.HandleFunc("/v1/messages/count_tokens", s.handleCountTokens)
	mux.HandleFunc("/v1/models", s.handleModels)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeAnthropicError(w, http.StatusNotFound, "not_found_error",
			fmt.Sprintf("endpoint non gere par le pont multiai: %s %s", r.Method, r.URL.Path))
	})

	s.httpSrv = &http.Server{Handler: mux}
	go func() { _ = s.httpSrv.Serve(ln) }()
	return s, nil
}

// URL returns the base URL clients must use as ANTHROPIC_BASE_URL.
func (s *Server) URL() string {
	return "http://" + s.ln.Addr().String()
}

// Target returns the backend base URL (for display).
func (s *Server) Target() string { return s.cfg.Target }

// Shutdown stops the server, waiting briefly for in-flight requests.
func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.httpSrv.Shutdown(ctx)
}

// handleMessages implements POST /v1/messages.
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAnthropicError(w, http.StatusMethodNotAllowed, "invalid_request_error", "methode non autorisee")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "corps illisible")
		return
	}
	var areq aRequest
	if err := json.Unmarshal(body, &areq); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "JSON invalide: "+err.Error())
		return
	}

	oreq, err := toOpenAI(&areq, s.cfg.MaxTokensCap)
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	payload, err := json.Marshal(oreq)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", err.Error())
		return
	}

	backendReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		s.cfg.Target+"/chat/completions", strings.NewReader(string(payload)))
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", err.Error())
		return
	}
	backendReq.Header.Set("Content-Type", "application/json")
	backendReq.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	if areq.Stream {
		backendReq.Header.Set("Accept", "text/event-stream")
	}

	resp, err := s.client.Do(backendReq)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "backend inaccessible: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		writeAnthropicError(w, resp.StatusCode, errorTypeForStatus(resp.StatusCode),
			fmt.Sprintf("backend %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet))))
		return
	}

	if areq.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, _ := w.(http.Flusher)
		_ = translateStream(w, flusher, resp.Body, areq.Model)
		return
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "reponse backend illisible: "+err.Error())
		return
	}
	var oresp oResponse
	if err := json.Unmarshal(respBody, &oresp); err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "reponse backend invalide: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, fromOpenAI(&oresp, areq.Model))
}

// handleCountTokens implements POST /v1/messages/count_tokens with a rough
// chars/4 estimation: the backend has no equivalent endpoint, and Claude
// Code only uses this for budgeting.
func (s *Server) handleCountTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAnthropicError(w, http.StatusMethodNotAllowed, "invalid_request_error", "methode non autorisee")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "corps illisible")
		return
	}
	var areq aRequest
	if err := json.Unmarshal(body, &areq); err != nil {
		writeAnthropicError(w, http.StatusBadRequest, "invalid_request_error", "JSON invalide: "+err.Error())
		return
	}
	chars := len(flattenSystem(areq.System))
	for _, m := range areq.Messages {
		chars += len(m.Content)
	}
	for _, t := range areq.Tools {
		chars += len(t.Name) + len(t.Description) + len(t.InputSchema)
	}
	writeJSON(w, http.StatusOK, map[string]any{"input_tokens": chars/4 + 1})
}

// handleModels proxies GET /v1/models and reshapes the OpenAI list into the
// Anthropic models list, so Claude Code gateway model discovery works.
func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	backendReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, s.cfg.Target+"/models", nil)
	if err != nil {
		writeAnthropicError(w, http.StatusInternalServerError, "api_error", err.Error())
		return
	}
	if s.cfg.APIKey != "" {
		backendReq.Header.Set("Authorization", "Bearer "+s.cfg.APIKey)
	}
	resp, err := s.client.Do(backendReq)
	if err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "backend inaccessible: "+err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		writeAnthropicError(w, resp.StatusCode, errorTypeForStatus(resp.StatusCode), "backend /models en erreur")
		return
	}
	var list struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&list); err != nil {
		writeAnthropicError(w, http.StatusBadGateway, "api_error", "liste backend invalide: "+err.Error())
		return
	}
	models := make([]map[string]any, 0, len(list.Data))
	for _, m := range list.Data {
		models = append(models, map[string]any{
			"type":         "model",
			"id":           m.ID,
			"display_name": m.ID,
			"created_at":   time.Unix(maxInt64(m.Created, 0), 0).UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": models, "first_id": firstID(models), "last_id": lastID(models), "has_more": false,
	})
}

func maxInt64(v, floor int64) int64 {
	if v < floor {
		return floor
	}
	return v
}

func firstID(models []map[string]any) any {
	if len(models) == 0 {
		return nil
	}
	return models[0]["id"]
}

func lastID(models []map[string]any) any {
	if len(models) == 0 {
		return nil
	}
	return models[len(models)-1]["id"]
}

// errorTypeForStatus maps an HTTP status to the Anthropic error type slug.
func errorTypeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid_request_error"
	case http.StatusUnauthorized:
		return "authentication_error"
	case http.StatusForbidden:
		return "permission_error"
	case http.StatusNotFound:
		return "not_found_error"
	case http.StatusTooManyRequests:
		return "rate_limit_error"
	case http.StatusServiceUnavailable, 529:
		return "overloaded_error"
	}
	return "api_error"
}

// writeAnthropicError writes an Anthropic-format error document.
func writeAnthropicError(w http.ResponseWriter, status int, typ, msg string) {
	writeJSON(w, status, map[string]any{
		"type":  "error",
		"error": map[string]any{"type": typ, "message": msg},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
