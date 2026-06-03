// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 NEXUS contributors

package dashboard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
)

func TestPlaygroundModelsEmpty(t *testing.T) {
	isolateNexusHome(t)
	s := NewServer(0, 0, nil, nil)
	rec := httptest.NewRecorder()
	s.handlePlaygroundModels(rec, httptest.NewRequest("GET", "/api/playground/models", nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["configured"] != false {
		t.Errorf("configured = %v, want false on a fresh home", got["configured"])
	}
	if models, _ := got["models"].([]interface{}); len(models) != 0 {
		t.Errorf("models = %d, want 0", len(models))
	}
}

func TestPlaygroundModelsListsConfigured(t *testing.T) {
	isolateNexusHome(t)
	// Save a small config with two providers.
	cfg := &config.Config{Providers: []config.Provider{
		{Name: "groq", APIKey: "gsk_x"},
		{Name: "deepseek", APIKey: "sk_y"},
	}}
	cfg.Proxy.Port = 3000
	cfg.Dashboard.Port = 2222
	cfg.Routing.Strategy = "auto"
	if err := config.Save("", cfg); err != nil {
		t.Fatal(err)
	}

	s := NewServer(0, 0, nil, nil)
	rec := httptest.NewRecorder()
	s.handlePlaygroundModels(rec, httptest.NewRequest("GET", "/api/playground/models", nil))
	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var got map[string]interface{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["configured"] != true {
		t.Errorf("configured = %v, want true", got["configured"])
	}
	models, _ := got["models"].([]interface{})
	if len(models) < 2 {
		t.Fatalf("models = %d, want at least 2 (groq + deepseek defaults)", len(models))
	}
	// Look for at least one groq and one deepseek entry, with the right tier.
	var sawGroq, sawDeepSeek bool
	for _, mAny := range models {
		m := mAny.(map[string]interface{})
		if m["provider"] == "groq" && m["tier"] == "free" {
			sawGroq = true
		}
		if m["provider"] == "deepseek" && m["tier"] == "standard" {
			sawDeepSeek = true
		}
	}
	if !sawGroq {
		t.Error("missing groq (free tier)")
	}
	if !sawDeepSeek {
		t.Error("missing deepseek (standard tier)")
	}
}

// TestPlaygroundChatForwardsToProxy stubs a fake proxy on a random port and
// verifies the chat endpoint forwards stream=true Anthropic SSE through.
func TestPlaygroundChatForwardsToProxy(t *testing.T) {
	isolateNexusHome(t)

	// Fake upstream proxy that confirms stream=true and answers with SSE.
	var gotBody []byte
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\"}\n\n"))
		_, _ = w.Write([]byte("event: content_block_delta\ndata: {\"delta\":{\"text\":\"hi\"}}\n\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}))
	defer upstream.Close()

	// Server must point at the fake upstream's port. httptest URLs are
	// http://127.0.0.1:<port> so we parse the port.
	parts := strings.Split(strings.TrimPrefix(upstream.URL, "http://"), ":")
	if len(parts) != 2 {
		t.Fatalf("bad upstream URL %q", upstream.URL)
	}
	var port int
	if _, err := jsonScanInt(parts[1], &port); err != nil {
		t.Fatal(err)
	}
	s := NewServer(0, port, nil, nil)

	body := []byte(`{"model":"claude-haiku-4-5","messages":[{"role":"user","content":"hi"}]}`)
	rec := httptest.NewRecorder()
	s.handlePlaygroundChat(rec, httptest.NewRequest("POST", "/api/playground/chat", bytes.NewReader(body)))

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q, want event-stream", ct)
	}
	if !strings.Contains(rec.Body.String(), "content_block_delta") {
		t.Errorf("body missing delta: %q", rec.Body.String())
	}
	// Stream=true must have been added by the shim.
	if !strings.Contains(string(gotBody), `"stream":true`) {
		t.Errorf("upstream body missing stream=true: %s", gotBody)
	}
}

func TestPlaygroundChatRejectsBadJSON(t *testing.T) {
	isolateNexusHome(t)
	s := NewServer(0, 0, nil, nil)
	rec := httptest.NewRecorder()
	s.handlePlaygroundChat(rec, httptest.NewRequest("POST", "/api/playground/chat", bytes.NewBufferString("not json")))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestEnsureKnownProvider(t *testing.T) {
	if !ensureKnownProvider("groq") {
		t.Error("groq should be known")
	}
	if ensureKnownProvider("definitely-not-real") {
		t.Error("unknown provider should NOT pass")
	}
}

// jsonScanInt parses an unsigned port number — tiny helper so the test stays
// in one file without dragging in strconv.Atoi just for this.
func jsonScanInt(s string, out *int) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n, io.EOF
		}
		n = n*10 + int(c-'0')
	}
	*out = n
	return n, nil
}
