# Co-Review v2 MVP TODO

## Project goal

Build a testable MVP of Co-Review v2 that can review GitLab merge requests, generate 4R findings, let a human approve/publish inline comments, and remember accepted repo decisions. GitHub, Telegram, and advanced analytics come after the GitLab MR path is reliable.

## Current-state snapshot

| Area | Current state |
|------|---------------|
| Repository layout | Root contains `docs/`, `packages/`, skill/config folders, and this TODO. |
| Product spec | Main architecture/specification lives in `docs/docs/SPECS.md`. |
| Packages | `packages/` contains `server/` and `web-spa/`; `cli/` is still planned. |
| Web SPA | Vue 3 + Vite + Pinia + Tailwind CSS v4 + Vitest scaffold exists, with minimal routes and placeholder UI. |
| Backend | `packages/server` exists as a Go backend with health/API routing, SQLite migrations, provider abstraction, skill loading, and harness core. |
| CLI | No Go CLI package exists yet. |
| Database | SQLite connection and embedded MVP migrations exist in `packages/server`. |
| MVP priority | GitLab merge request review and inline comments first; GitHub/Telegram later. |

## Intended package layout

```text
packages/
├── server/       # Go backend: REST API, SSE, DB, review engine, providers, platform adapters
├── cli/          # Go CLI: local/remote commands for repos, reviews, publishing, memory, skills
└── web-spa/      # Existing Vue 3 SPA: dashboard, repo setup, review approval/publish UI
```

Future packages can be added only when the MVP needs them. Keep GitHub, Telegram bot, hosted workers, and shared SDK generation out of the first delivery unless they unblock GitLab MR review.

## Phase 0 — Workspace baseline

### Objective

Make the repository ready for multi-package development without changing product behavior.

### Scope

- Document package boundaries and local development commands.
- Add root-level workspace conventions only if needed by later phases.
- Keep the existing `packages/web-spa` scaffold intact.

### Deliverables

- [x] Root development notes or scripts for server, CLI, and web commands.
- [x] Confirmed naming: `packages/server`, `packages/cli`, `packages/web-spa`.
- [x] Root `.gitignore` updated to ignore common Go, Node, DB, and coverage artifacts.
- [x] Documented that `packages/web-spa` already includes `@lucide/vue` for future Vue icon work.

### Verification / tests

- [x] `packages/web-spa`: run type-check script.
- [x] `packages/web-spa`: run unit test script.
- [x] `packages/web-spa`: run build script.
- [x] Confirm no backend/CLI code has been added accidentally in this phase.

### Exit criteria

- [x] A new contributor can identify where backend, CLI, and SPA code will live.
- [x] Existing SPA still builds/tests as before.

## Phase 1 — Server foundation and database migrations

### Objective

Create the Go backend shell with persistence primitives needed by every later feature.

### Scope

- Scaffold `packages/server` as a Go module.
- Add HTTP server bootstrap, config loading, structured error responses, and health endpoint.
- Add SQLite-first DB connection and migration runner.
- Create initial migrations for repos, model configs, skills, reviews, review comments, and repo memory.

### Deliverables

- [x] `packages/server/go.mod` and server entrypoint.
- [x] HTTP routing structure for `/api/v1/*` and `/healthz`.
- [x] Migration files covering MVP tables from `docs/docs/SPECS.md`.
- [x] DB package with migration and test helpers.

### Verification / tests

- [x] Go unit tests for config parsing, health handler, and migration application against `t.TempDir()` SQLite DB.
- [x] Migration test proves schema creates all MVP tables and key indexes.
- [x] `go test ./...` from `packages/server` passes.

### Exit criteria

- [x] Server starts locally and reports healthy.
- [x] Test DB can migrate from empty state deterministically.

## Phase 2 — Provider abstraction and skills/harness core

### Objective

Run deterministic review agents behind a provider interface before touching GitLab.

### Scope

- Implement `ModelProvider` and registry contracts.
- Add a fake/test provider for deterministic tests.
- Load 4R skill markdown files from filesystem.
- Implement harness timeout, retry, JSON schema validation, and structured errors.
- Store skill metadata in DB or expose loaded skills through the server.

### Deliverables

- [x] Provider interfaces and registry for Claude/OpenAI-compatible providers, plus fake provider.
- [x] Skill loader for R1 Risk, R2 Readability, R3 Reliability, and R4 Resilience.
- [x] JSON schemas for agent outputs.
- [x] Harness result model with attempts, duration, token metadata, output, and error.

### Verification / tests

- [x] Table-driven Go tests for provider registry resolution.
- [x] Skill loader tests for valid frontmatter, missing files, and invalid metadata.
- [x] Harness tests for success, timeout, provider error retry, invalid JSON, and invalid schema.
- [x] No real model API calls in unit tests.

### Exit criteria

- [x] A fake provider can produce validated 4R outputs through the harness.
- [x] Harness failures are stored/returned as structured errors, not panics.

## Phase 3 — GitLab platform adapter and MR diff ingestion

### Objective

Fetch enough GitLab MR context to review real merge requests and map findings to inline-comment positions.

### Scope

- Define platform adapter interface around repo inference, branches, MR metadata, diffs, and inline comment publishing.
- Implement GitLab adapter first.
- Defer GitHub adapter to post-MVP unless an interface test needs a stub.
- Add secure token handling via environment variable references, not stored raw secrets.

### Deliverables

- `PlatformClient` interface and GitLab implementation.
- GitLab repo URL inference and MR metadata/diff fetching.
- Internal diff/position model containing `base_sha`, `start_sha`, `head_sha`, file path, and line mapping.
- Fake GitLab server/test fixtures for API contract tests.

### Verification / tests

- Unit tests for GitLab URL parsing and repo inference.
- HTTP fixture tests for GitLab MR metadata, changes, and error responses.
- Position mapping tests for added/changed lines that can receive inline comments.
- `testing.Short()` skips any optional live GitLab integration tests.

### Exit criteria

- Given a GitLab project and MR IID, the server can fetch reviewable diff context without publishing anything.
- Inline comment positions can be computed for generated findings.

## Phase 4 — Review orchestration, persistence, and SSE

### Objective

Generate and persist a complete review while streaming progress to clients.

### Scope

- Implement review creation for GitLab MRs.
- Run 4R harnesses concurrently with cancellation and per-agent status.
- Persist reviews, scores, verdict, generated comments, and harness errors.
- Expose review detail, comments, history, and SSE progress endpoints.

### Deliverables

- `POST /api/v1/reviews` for manual GitLab MR review.
- `GET /api/v1/reviews`, `GET /api/v1/reviews/:id`, `GET /api/v1/reviews/:id/comments`.
- `GET /api/v1/reviews/:id/events` SSE stream.
- Review state transitions: `pending -> running -> generated|awaiting_approval|error`.

### Verification / tests

- Orchestrator tests using fake provider and fake platform client.
- Persistence tests for review/comment state transitions.
- SSE handler test validates event names and JSON payload shape.
- API handler tests for success, invalid MR input, platform failure, provider failure, and cancellation.

### Exit criteria

- A fake GitLab MR can be reviewed end-to-end with stored findings and observable SSE progress.
- Generated comments remain pending until explicit approval/publish action.

## Phase 5 — Repo configuration and repo memory MVP

### Objective

Let users configure repos/models and avoid re-flagging accepted decisions.

### Scope

- Implement repo CRUD and GitLab repo inference.
- Implement active model config per repo.
- Implement local SQLite `repo_memory` as the first memory backend.
- Inject accepted decisions and known patterns into 4R prompts.
- Support marking comments as accepted decisions.

### Deliverables

- `POST/GET/PATCH/DELETE /api/v1/repos`.
- `POST /api/v1/repos/infer` for GitLab URLs.
- `GET/PUT /api/v1/repos/:id/model`.
- `GET/POST/DELETE /api/v1/repos/:id/memory`.
- Comment status update endpoint for `approved`, `accepted_decision`, and `discarded`.

### Verification / tests

- API tests for repo CRUD, model config validation, and GitLab inference fallback behavior.
- Memory tests for accepted-decision TTL rules and prompt-context rendering.
- Regression test proving accepted decisions are injected into later review prompts.

### Exit criteria

- A repo can be configured once, reviewed repeatedly, and use accepted decisions as future context.

## Phase 6 — Approval and GitLab inline publish flow

### Objective

Publish only human-approved review comments to GitLab MRs.

### Scope

- Implement publish modes: all approved comments or selected sequential comments.
- Format platform comments from 4R findings.
- Publish GitLab inline comments through the Discussion API.
- Persist platform comment IDs and publish status.
- Keep GitHub publish support deferred behind the platform interface.

### Deliverables

- `POST /api/v1/reviews/:id/publish`.
- Comment formatter with dimension/severity marker, evidence, rationale, and suggestion.
- GitLab inline publish implementation.
- Idempotency guard against duplicate publishing.

### Verification / tests

- Formatter snapshot/golden tests for published markdown.
- GitLab fixture tests for Discussion API request payloads.
- API tests for publish-all, selected publish, already-published comments, discarded comments, and platform failure.
- Manual smoke test against a test GitLab MR before production use.

### Exit criteria

- A completed review can publish approved inline comments to a GitLab MR exactly once.
- Accepted decisions are saved to repo memory instead of being published.

## Phase 7 — Web SPA MVP for GitLab review workflow

### Objective

Give users a browser UI for configuring GitLab repos, watching reviews, and approving/publishing comments.

### Scope

- Build on existing `packages/web-spa` Vue 3 scaffold.
- Add API client, typed schemas, and feature modules for repos, reviews, skills, providers, and memory.
- Use SSE to show live review progress.
- Implement sequential approval UI for generated comments.
- Keep Telegram/channel management out of MVP UI unless needed for GitLab review.

### Deliverables

- Dashboard and recent reviews.
- Repo list/detail and GitLab repo creation wizard.
- Model provider/model selection UI.
- Review detail page with SSE agent progress.
- Comment approval/publish flow with actions: approve, accept decision, discard, publish.

### Verification / tests

- Vitest unit/component tests for API client, composables, stores, and approval components.
- Mocked `EventSource` tests for review stream state transitions.
- Type-check and production build pass.
- Manual browser smoke test against local server with fake provider/GitLab fixtures.

### Exit criteria

- A user can configure a GitLab repo, start/watch a review, triage comments, and publish approved comments from the SPA.

## Phase 8 — Go CLI MVP for GitLab review workflow

### Objective

Support the same MVP workflow from the terminal, primarily through remote server mode.

### Scope

- Scaffold `packages/cli` as a Go module.
- Implement config file handling for server URL and API token.
- Add commands for repo setup, provider/model inspection, review run/watch/status, publish, and memory management.
- Use Bubbletea/huh only where interactive flows materially help.
- Keep local embedded-engine mode as a later enhancement unless server mode is complete.

### Deliverables

- `review-ctl repo add/list/show/set-model/remove`.
- `review-ctl provider list/models`.
- `review-ctl review run/watch/status/publish/history`.
- `review-ctl memory list/accept/delete`.
- CLI API client shared inside the CLI package.

### Verification / tests

- Table-driven tests for command parsing and config loading using `t.TempDir()`.
- API client tests against `httptest.Server`.
- Watch command tests against a fake SSE stream.
- Optional Bubbletea model tests for interactive state transitions.

### Exit criteria

- A user can run, watch, approve/publish, and inspect GitLab MR reviews from the CLI through the server.

## Phase 9 — Hardening, observability, and MVP release gate

### Objective

Make the GitLab MR review path safe enough to use on real repositories.

### Scope

- Add request logging, correlation IDs, error classification, and basic metrics.
- Add auth/API token protection if the server is not strictly local-only.
- Add rate limits/timeouts for provider and GitLab calls.
- Add fixture-backed integration tests for the full review-to-publish path.
- Document setup and operational constraints.

### Deliverables

- End-to-end test harness using fake provider and fake GitLab API.
- Release checklist for GitLab MVP.
- `.env.example` with secret placeholders only.
- README or docs covering server, SPA, CLI, DB migrations, and test commands.

### Verification / tests

- Full local E2E: configure repo -> fetch MR -> run review -> approve comments -> publish to fake GitLab -> verify DB state.
- Security checks for missing tokens, invalid webhook secrets, duplicate publish, and unsafe inline positions.
- All server, CLI, and SPA test/build commands pass.

### Exit criteria

- GitLab MR review MVP is ready for controlled real-world use.
- Post-MVP work can begin without changing the core review/publish contracts.

## Post-MVP backlog

- GitHub platform adapter and inline publish support.
- Telegram bot and notification channel management.
- Slack/webhook notification channels.
- Optional Engram MCP memory backend per repo.
- Technical-debt trend dashboard and richer analytics.
- Local CLI embedded-engine mode.
- Multi-tenant auth, deployment manifests, and production-grade operations.
