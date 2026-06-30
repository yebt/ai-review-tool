# Co-Review v2 MVP TODO

## Project goal

Build a testable MVP of Co-Review v2 that can review GitLab merge requests, generate 4R findings, let a human approve/publish inline comments, and remember accepted repo decisions. GitHub, Telegram, and advanced analytics come after the GitLab MR path is reliable.

## Current-state snapshot

| Area | Current state |
|------|---------------|
| Repository layout | Root contains `docs/`, `packages/`, skill/config folders, and this TODO. |
| Product spec | Main architecture/specification lives in `docs/docs/SPECS.md`. |
| Packages | `packages/` contains `server/` and `web-spa/`; `cli/` is still planned. |
| Web SPA | Vue 3 + Vite + Pinia + Tailwind CSS v4 + Vitest exists, with brutalist Phase 4.5 route separation for `/`, `/health`, `/skills`, `/reviews`, and `/repos`. |
| Backend | `packages/server` exists as a Go backend with health/API routing, SQLite migrations, provider abstraction, skill loading, harness core, and GitLab MR diff ingestion. |
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

Phase 3 starts the platform boundary for GitLab only: parse GitLab project/MR inputs, fetch merge request metadata and changed files, and produce an internal diff context that later phases can review. It must not run review orchestration or publish comments yet.

### Non-goals

- No review orchestration, harness fan-out, persistence workflow, or SSE progress; those belong to Phase 4.
- No publishing comments or GitLab Discussion API writes; publishing belongs to Phase 6.
- No GitHub adapter yet, except for interface shape notes if they prevent GitLab-specific coupling.
- No real tokens, secrets, SDK lock-in, or live API dependency in normal unit tests.

### Scope

- Define platform adapter interface around repo inference, branches, MR metadata, diffs, and inline comment publishing.
- Implement GitLab adapter first.
- Defer GitHub adapter to post-MVP unless an interface test needs a stub.
- Add secure token handling via environment variable references, not stored raw secrets.

### Deliverables

- [x] `PlatformClient` interface and GitLab implementation.
- [x] GitLab repo URL inference and MR metadata/diff fetching.
- [x] Internal diff/position model containing `base_sha`, `start_sha`, `head_sha`, file path, and line mapping.
- [x] Fake GitLab server/test fixtures for API contract tests.

### Verification / tests

- [x] Unit tests for GitLab URL parsing and repo inference.
- [x] HTTP fixture tests for GitLab MR metadata, changes, and error responses.
- [x] Position mapping tests for added/changed lines that can receive inline comments.
- [x] `testing.Short()` skips any optional live GitLab integration tests. No live tests were added in Phase 3.

### Phase 3 execution notes

#### Test strategy

| Area | Expected test shape |
|------|---------------------|
| URL parsing | Table-driven tests for HTTPS/SSH GitLab project URLs, namespace paths, `.git` suffixes, and invalid inputs. |
| MR metadata and changes | `httptest.Server` fake GitLab API with JSON fixtures for happy path, missing MR, unauthorized, and malformed response cases. |
| Diff position mapping | Fixture tests that map changed lines to `base_sha`, `start_sha`, `head_sha`, old/new paths, and inline-commentable line positions. |
| Live GitLab smoke tests | Optional external tests only; skip under `testing.Short()` and when env vars are absent. |

Future live-test configuration names, if live tests are added:

- `CO_REVIEW_GITLAB_BASE_URL` — GitLab instance URL, defaulting to `https://gitlab.com` in tests only when explicitly enabled.
- `CO_REVIEW_GITLAB_TOKEN` — personal/project access token for a disposable test project.
- `CO_REVIEW_GITLAB_PROJECT_ID` — project ID or URL-escaped path for fixture MR access.
- `CO_REVIEW_GITLAB_MR_IID` — merge request IID in that disposable project.

Do not store real values for these variables in the repo.

#### Start checklist

- [x] Define the smallest `PlatformClient` contract needed to fetch MR context without publish operations.
- [x] Add GitLab URL/project inference tests before implementation.
- [x] Add fake GitLab API fixtures with `httptest.Server`; normal tests must not call the network.
- [x] Model diff context separately from provider/harness output so Phase 4 can consume it without GitLab-specific types.
- [x] Implement line-position mapping tests before wiring API handlers.
- [x] Keep publish/comment-write methods out of Phase 3 implementation.

### Exit criteria

- [x] Given a GitLab project and MR IID, the server can fetch reviewable diff context without publishing anything.
- [x] Inline comment positions can be computed for generated findings.

## Phase 3.5 — Backend smoke UI

### Objective

Expose the currently available backend endpoints in the browser so API shape issues can be caught before the full review UI exists.

### Scope

- Build a temporary minimal brutalist dashboard in `packages/web-spa`.
- Test only existing backend endpoints: `/healthz` and `/api/v1/skills`.
- Keep real GitLab review workflow UI out of scope until Phase 7.

### Deliverables

- [x] Smoke dashboard module with focused Vue components and `useSmokeChecks()` composable.
- [x] Vite dev proxy for same-origin calls to the Go server.
- [x] Zod validation for `/healthz` and `/api/v1/skills` response shapes.
- [x] Metadata-only skills rendering without prompt body or internal file path leakage.
- [x] Brutalist UI states for idle, loading, success, error, and refresh.
- [x] Removed tracked Python `.pyc` cache files from web-spa skill copies and ignored future cache artifacts.

### Verification / tests

- [x] Vitest component tests mock `fetch` and cover success, endpoint error, refresh, and validation failure.
- [x] `bun run lint` passes in `packages/web-spa`.
- [x] `bun run type-check` passes in `packages/web-spa`.
- [x] `bun run test:unit` passes in `packages/web-spa`.
- [x] `bun run build` passes in `packages/web-spa`.
- [x] `make spa-test` and `make spa-build` pass from the repo root.

### Exit criteria

- [x] A developer can run `make server-run` and `make spa-dev` to inspect backend health and loaded skills in the browser.
- [x] The smoke UI clearly states that full review workflow features are not available yet.

## Phase 4 — Review orchestration, persistence, and SSE

### Objective

Generate and persist a complete review while streaming progress to clients.

### Scope

- [x] Implement review creation for GitLab MRs.
- [x] Run 4R harnesses concurrently with cancellation and per-agent status.
- [x] Persist reviews, scores, verdict, generated comments, and harness errors.
- [x] Expose review detail, comments, history, and SSE progress endpoints.

### Deliverables

- [x] `POST /api/v1/reviews` for manual GitLab MR review.
- [x] `GET /api/v1/reviews`, `GET /api/v1/reviews/:id`, `GET /api/v1/reviews/:id/comments`.
- [x] `GET /api/v1/reviews/:id/events` SSE stream.
- [x] Review state transitions: `pending -> running -> generated|awaiting_approval|error`.

### Verification / tests

- [x] Orchestrator tests using fake provider and fake platform client.
- [x] Persistence tests for review/comment state transitions.
- [x] SSE handler test validates event names and JSON payload shape.
- [x] API handler tests for success, invalid MR input, platform failure, and provider failure. Cancellation remains covered through request context propagation and harness timeout behavior rather than a dedicated API test.

### Exit criteria

- [x] A fake GitLab MR can be reviewed end-to-end with stored findings and observable SSE progress.
- [x] Generated comments remain pending until explicit approval/publish action.

## Phase 4.5 — Review operations smoke UI

### Objective

Expose the Phase 4 backend review actions in the SPA for manual testing before the final Phase 7 product UI.

### Scope

- [x] Add a manual GitLab MR review form using `POST /api/v1/reviews`.
- [x] List review history from `GET /api/v1/reviews`.
- [x] Select a review and load `GET /api/v1/reviews/:id` plus generated comments from `GET /api/v1/reviews/:id/comments`.
- [x] Open an EventSource stream for `GET /api/v1/reviews/:id/events` and display review/agent events.
- [x] Keep current limitations visible: no publish/approval UI, no repo CRUD, no repo memory UI, and backend provider behavior may be deterministic/fake.
- [x] Separate Phase 4.5 actions into independent file-based routes: `/` capability map, `/health`, `/skills`, `/reviews`, and `/repos` Phase 5 placeholder.

### Verification / tests

- [x] Vitest tests mock `fetch` and `EventSource`; no Go server is required for unit tests.
- [x] Response shapes are validated with Zod for review creation, list, detail, and comments.
- [x] Route/page tests prove home is a static capability map, health and skills load independently, and `/repos` remains a placeholder.

### Exit criteria

- [x] A developer can start the local server and SPA, create a fake/deterministic GitLab MR review, inspect comments, and watch selected review events without touching final Phase 7 flows.
- [x] Review operations live on `/reviews` instead of being mixed into the home page.

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
