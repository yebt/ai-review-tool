# Co-Review v2 Development Notes

Co-Review v2 is organized as a packages-based workspace. The MVP priority is the GitLab merge request review workflow: fetch MR context, generate 4R findings, approve comments, publish inline comments, and remember accepted repo decisions.

## Package boundaries

| Package | Status | Boundary |
|---------|--------|----------|
| `packages/server` | Planned | Go backend for REST API, SSE, database migrations, review engine, model providers, platform adapters, and repo memory. Do not place server code at the repository root. |
| `packages/cli` | Planned | Go CLI for local/remote review workflows, publishing, repo configuration, memory, and skills. Do not place CLI code at the repository root. |
| `packages/web-spa` | Existing | Vue 3 SPA for dashboard, repo setup, review approval, and publish UI. |

`packages/server` and `packages/cli` are intentionally not scaffolded yet. Add them only in their implementation phases.

## Local commands

Run commands from the package that owns the code.

| Area | Command | Notes |
|------|---------|-------|
| Web setup | `cd packages/web-spa && bun install` | Use the existing `bun.lock`. Do not install from the repository root. |
| Web dev server | `cd packages/web-spa && bun run dev` | Starts the Vite development server. |
| Web type-check | `cd packages/web-spa && bun run type-check` | Uses `vue-tsc --build`. |
| Web unit tests | `cd packages/web-spa && bun run test:unit` | Uses Vitest. |
| Web build | `cd packages/web-spa && bun run build` | Runs type-check and production build. |
| Server | Not available yet | Future package: `packages/server`. Expected verification after scaffolding: `go test ./...` from that package. |
| CLI | Not available yet | Future package: `packages/cli`. Expected verification after scaffolding: `go test ./...` from that package. |

## Frontend notes

- `packages/web-spa` already includes `@lucide/vue` (`^1.21.0`) for future Vue icon work.
- Keep Vue code in Vue 3 Composition API style with `<script setup lang="ts">` unless the package establishes a different convention.

## Source of truth

- Architecture/specification: `docs/docs/SPECS.md`.
- Implementation plan: `TODO.md`.
