<p align="center">
  <img src="docs/assets/banner-3.jpg" alt="Co-Review Banner" width="100%">
</p>

# Co-Review v2

> An intelligent, multi-agent AI code review companion.

Co-Review v2 is an AI-powered code review system designed to streamline GitLab merge requests. It fetches MR context, generates highly focused **4R findings** (Relevance, Risk, Remediability, Relation), provides an interactive UI for approving comments, publishes inline reviews, and learns from accepted repository decisions.

---

## 🚀 Key Features

- 🤖 **4R Review Engine**: Evaluates code changes along four structured dimensions: Relevance, Risk, Remediability, and Relation to existing patterns.
- 🦊 **GitLab MR Workflow**: Smooth integration to fetch merge requests, analyze diffs, and publish inline review comments.
- 🧠 **Repository Memory**: Remembers and enforces architectural decisions, coding styles, and project-specific guidelines.
- 💻 **Interactive Approval UI**: Clean, real-time Vue 3 dashboard to preview, edit, and approve AI findings before they reach GitLab.
- 🛠️ **Extensible Architecture**: Modularity across a Go-powered backend, a Go-based CLI tool, and a lightweight web frontend.

---

## 📂 Repository Layout

Co-Review is organized as a packages-based workspace. Please keep code bounded strictly within these packages:

| Package | Path | Status | Purpose & Boundary |
|:---|:---|:---|:---|
| **Server** | [`packages/server`](file:///home/webcloster-dev/Development/Repos/PRODUCTIVITY/ai-review-4/packages/server) | Existing | Go backend for REST API, SSE, database migrations, review engine, model providers, and repo memory. |
| **Web SPA** | [`packages/web-spa`](file:///home/webcloster-dev/Development/Repos/PRODUCTIVITY/ai-review-4/packages/web-spa) | Existing | Vue 3 Single Page Application for dashboard, repo setup, and review approval/publish UI. |
| **CLI** | `packages/cli` | Planned | Go CLI for local/remote review workflows, publishing, and local repository memory commands. |

*Note: Do not place package-specific implementation code at the repository root. `packages/cli` is not scaffolded yet and will be added during its respective implementation phase.*

---

## 🛠️ Local Development Commands

Use the root [Makefile](file:///home/webcloster-dev/Development/Repos/PRODUCTIVITY/ai-review-4/Makefile) for workspace-wide commands, or execute package-specific scripts inside their respective folders.

```bash
make help  # View all available make targets
```

### Development & Verification Commands

| Scope | Command | Notes / Description |
|:---|:---|:---|
| **Workspace** | `make dev` | Starts the local dev servers. |
| **Workspace** | `make check` | Runs full workspace check (SPA checks + server tests). |
| **Workspace** | `make build` | Compiles and builds all active packages. |
| **Web SPA** | `make install` or `make spa-install` | Installs frontend dependencies using `packages/web-spa/bun.lock`. |
| **Web SPA** | `make spa-dev` | Starts the Vite development server. |
| **Web SPA** | `make spa-type-check` | Type-checks the Vue application using `vue-tsc --build`. |
| **Web SPA** | `make spa-test` | Runs frontend unit tests via Vitest. |
| **Web SPA** | `make spa-build` | Runs type-checks and builds the production frontend. |
| **Web SPA** | `make spa-check` | Runs type-checks, unit tests, and the production build. |
| **Go Server** | `make server-run` | Starts the Go REST/SSE backend. |
| **Go Server** | `make server-test` | Executes backend tests (`go test ./...`). |
| **Go Server** | `make server-build` | Builds the Go binary at `packages/server/bin/server`. |

---

## 🎨 Frontend Guidelines

- Icons: [`packages/web-spa`](file:///home/webcloster-dev/Development/Repos/PRODUCTIVITY/ai-review-4/packages/web-spa) has `@lucide/vue` installed (`^1.21.0`) for vector iconography.
- Vue Style: Keep codebase uniform by using **Vue 3 Composition API** with `<script setup lang="ts">` for all new SFCs.

---

## 📖 Project Reference & Specification

- **Architecture Details & Specifications**: See the complete architectural spec in [`docs/docs/SPECS.md`](file:///home/webcloster-dev/Development/Repos/PRODUCTIVITY/ai-review-4/docs/docs/SPECS.md).
- **Implementation Status & Roadmap**: Track active development progress in [`TODO.md`](file:///home/webcloster-dev/Development/Repos/PRODUCTIVITY/ai-review-4/TODO.md).
