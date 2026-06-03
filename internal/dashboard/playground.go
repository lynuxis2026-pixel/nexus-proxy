// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 NEXUS contributors

package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lynuxis2026-pixel/nexus-proxy/internal/config"
	"github.com/lynuxis2026-pixel/nexus-proxy/internal/providers"
)

// PlaygroundModel describes one selectable model in the picker.
type PlaygroundModel struct {
	Provider string `json:"provider"`
	Tier     string `json:"tier"`
	Model    string `json:"model"`
	Label    string `json:"label"`
}

// ── GET /api/playground/models ────────────────────────────────────────────
// Returns the full set of (provider, model) pairs the user can pick from,
// built from the configured providers + their default model list. Only shows
// providers the user has actually configured — matches the user's request.
func (s *Server) handlePlaygroundModels(w http.ResponseWriter, r *http.Request) {
	cfg := loadCfg()
	out := []PlaygroundModel{}
	for _, p := range cfg.Providers {
		name := strings.ToLower(p.Name)
		// Honour a user's explicit Models list; fall back to the built-in defaults.
		models := p.Models
		if len(models) == 0 {
			models = providers.DefaultModels(name)
		}
		tier := p.Tier
		if tier == "" {
			tier = providers.DefaultTier(name)
		}
		for _, m := range models {
			label := fmt.Sprintf("%s · %s", name, m)
			out = append(out, PlaygroundModel{Provider: name, Tier: tier, Model: m, Label: label})
		}
	}
	writeJSON(w, map[string]interface{}{
		"models":     out,
		"configured": len(cfg.Providers) > 0,
	})
}

// ── POST /api/playground/chat ─────────────────────────────────────────────
// Body matches the Anthropic /v1/messages shape. The handler:
//   1. forces stream=true (the UI is built around streaming),
//   2. forwards the request to the local proxy at proxyPort/v1/messages,
//   3. pipes the SSE response straight back to the browser.
// Every request flows through the normal proxy pipeline — cache, cascade,
// privacy firewall, cost tracking, the live feed — for free.
func (s *Server) handlePlaygroundChat(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Parse just enough to ensure stream=true; pass the rest through unchanged.
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	payload["stream"] = true
	if _, ok := payload["max_tokens"]; !ok {
		payload["max_tokens"] = 4096
	}
	upstreamBody, _ := json.Marshal(payload)

	// Resolve the proxy port: explicit field wins, else loaded config, else 3000.
	proxyPort := s.proxyPort
	if proxyPort == 0 {
		proxyPort = loadCfg().Proxy.Port
	}
	if proxyPort == 0 {
		proxyPort = 3000
	}
	url := fmt.Sprintf("http://localhost:%d/v1/messages", proxyPort)

	upstreamReq, err := http.NewRequestWithContext(r.Context(), "POST", url, bytes.NewReader(upstreamBody))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	// Forward optional pinning headers (so the UI can override routing per call).
	for _, h := range []string{"X-Nexus-Provider", "X-Nexus-Tier"} {
		if v := r.Header.Get(h); v != "" {
			upstreamReq.Header.Set(h, v)
		}
	}
	// "nexus-local" is the standard local-only token the proxy accepts; it never
	// leaves the machine. Real provider auth is done by the proxy from config.
	upstreamReq.Header.Set("X-API-Key", "nexus-local")

	resp, err := http.DefaultClient.Do(upstreamReq)
	if err != nil {
		http.Error(w, "proxy unreachable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Pipe through. We do NOT buffer — we want the user to see tokens as they
	// arrive, not after the whole response is collected.
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no") // hint for any nginx in front
	w.WriteHeader(resp.StatusCode)
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, werr := w.Write(buf[:n])
			if werr != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			return
		}
	}
}

// ensureKnownProvider reports whether name is a known built-in provider; used
// by tests and the playground's input validation.
func ensureKnownProvider(name string) bool {
	_, err := providers.FromConfig(name, "test", "", nil)
	return err == nil
}

// Silence unused-import warnings in builds that strip the model assertion.
var _ = config.Provider{}
