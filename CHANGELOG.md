# Changelog

High-level summary of each release. Full, commit-level notes are on the
[GitHub releases page](https://github.com/lynuxis2026-pixel/nexus-proxy/releases).

## Unreleased

- **Playground (chat UI):** a new "💬 Playground" overlay in the dashboard. Pick
  any model from any configured provider (Claude Opus, GPT-4o, DeepSeek, Groq,
  …), type a question, watch the answer stream back in real time. Every
  Playground call flows through the normal proxy pipeline so cache, cascade,
  privacy firewall and cost tracking apply for free, and the call shows up in
  the live feed. Streaming via `fetch` + `ReadableStream`, Anthropic SSE
  format. New endpoints `GET /api/playground/models` and
  `POST /api/playground/chat`. Five backend tests; verified end-to-end on the
  running binary.
- **First-run setup wizard:** open the dashboard after a fresh install and a
  4-step in-browser onboarding takes over (Welcome → Providers → Test → Connect).
  Auto-detects providers already in your env vars, recommends a sensible starter
  set, validates each key with a live `HealthCheck`, and prints a
  platform-aware (Win / macOS / Linux) copyable `ANTHROPIC_BASE_URL` + key
  snippet. Saves to `~/.nexus/config.toml`; writes a `setup-done` marker so it
  never re-appears. Skippable. New endpoints under `/api/setup/*`. Verified
  end-to-end on the running binary.
- **Licensing layer:** moved the project licence from MIT to **Apache-2.0** for
  trademark + patent-grant protection (forks may use the code but not the
  "NEXUS" name); added [`NOTICE`](NOTICE), a runtime
  [`internal/license`](internal/license/) package that's surfaced in
  `nexus version`, SPDX headers on the package entry points, and a full
  strategy doc at [`docs/LICENSING.md`](docs/LICENSING.md) — including a
  ready-to-drop BSL 1.1 template if the project ever wants newer versions to be
  source-available. Today every feature is unlocked; the seam is in place to
  gate later without touching call sites.

## v0.5.0

- **6 more providers (24 → 30):** Baseten, Featherless, kluster.ai, Venice
  (privacy-focused), Friendli, Chutes — all OpenAI-compatible, with env
  auto-discovery.
- **Smarter complexity classifier:** an explainable policy (security/architecture
  keywords, large-context, Opus → premium; trivial tool-less → free; ordinary
  coding/tool-use → cheap Standard) that reads intent from *user* text only.
  Routes the bulk of agentic coding to the cheap tier; full test matrix.
- **Dashboard:** a privacy "secrets masked · 0 leaked" card and a "routing mix"
  bar showing how the classifier split recent traffic.
- **Windows CI:** the test suite now runs on an ubuntu + windows matrix, plus a
  spaced-path DB-open guard.

## v0.4.1

- **Off-peak-aware pricing** — prices requests at a provider's off-peak rate
  during its UTC discount window (e.g. DeepSeek), so cost + savings reflect reality.
- **`X-Nexus-Tier` header** — pin a single call to a tier (premium for
  architecture/security, free for lint/format); lets an agent harness route per skill.
- **Agent-harness integration (ECC)** — stack-them guide, a `nexus mcp` config, and
  a proposal/PR to affaan-m/ECC.
- Launch playbook rewritten around the positioning.

## v0.4.0

- **`nexus bench`** — benchmark every provider on your *own* captured traffic
  (cost × latency × agreement) with a Provider Report Card + recommendation.
- **Adaptive routing** (`--adaptive`) — learns which provider wins each of your
  task types from real outcomes and reorders routing automatically.
- **Team mode** — per-user attribution + a shared cache + a savings leaderboard.
- **Rules engine** — declarative routing overrides in config.
- **Cost guardrail** (`--max-request-usd`) — downgrade a pricey single request.
- **Trust & Savings report** (`nexus report`) — "$ saved + N secrets masked · 0 leaked".
- Repositioned around Private · Proven · Self-learning · Local.

## v0.3.0

- **Cache-aware cost engine** — captures provider prompt-cache tokens and prices
  them at the real discount; shows a live "cache saved $X".
- **Prefix-normalized + opt-in semantic cache** (`--semantic-cache`).
- **Cheap-first cascade with verification** (`--cascade`).
- **Free-tier API key rotation / pools** with 429 cooldown.
- **Privacy firewall** (`--redact`) — mask secrets/PII before they leave; restore in
  the response.
- **Inspect & replay** (`--inspect`) — compare a captured request across providers.
- **CLI:** `nexus code`, `nexus doctor` (+ env auto-discovery), `nexus top`, `nexus mcp`.
- Budget spend alerts via Slack/Discord/generic webhook.

## v0.2.0 – v0.2.1

- 24 providers built in + a config-driven custom provider; Azure/Bedrock/Vertex
  (incl. AWS SigV4) with offline integration tests.
- **Universal gateway** — also speaks the OpenAI API (`/v1/chat/completions`).
- True OpenAI-compatible streaming, automatic failover, daily budget cap,
  normalized response cache, background health checks.
- Hardened + tested every route (httptest integration suite); fixes.

## v0.1.0

- Initial public release: single-binary proxy + live Svelte dashboard for Claude
  Code, intelligent complexity-based routing, SQLite request log + stats, and the
  first set of providers. GoReleaser cross-platform binaries.
