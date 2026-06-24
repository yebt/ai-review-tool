# Co-Review v2 Development Notes

Co-Review v2 is organized as a packages-based workspace. The MVP priority is the GitLab merge request review workflow: fetch MR context, generate 4R findings, approve comments, publish inline comments, and remember accepted repo decisions.

## Package boundaries

| Package | Status | Boundary |
|---------|--------|----------|
| `packages/server` | Existing | Go backend for REST API, SSE, database migrations, review engine, model providers, platform adapters, and repo memory. Do not place server code at the repository root. |
| `packages/cli` | Planned | Go CLI for local/remote review workflows, publishing, repo configuration, memory, and skills. Do not place CLI code at the repository root. |
| `packages/web-spa` | Existing | Vue 3 SPA for dashboard, repo setup, review approval, and publish UI. |

`packages/cli` is intentionally not scaffolded yet. Add it only in its implementation phase.

## Local commands

Use the root `Makefile` for common workspace commands, or run package commands directly when you need package-specific options.

```bash
make help
make dev
make check
```

| Area | Command | Notes |
|------|---------|-------|
| Web setup | `make install` or `make spa-install` | Uses the existing `packages/web-spa/bun.lock`. |
| Web dev server | `make dev` or `make spa-dev` | Starts the Vite development server. |
| Web type-check | `make type-check` or `make spa-type-check` | Uses `vue-tsc --build`. |
| Web unit tests | `make test` or `make spa-test` | Uses Vitest. |
| Workspace check | `make check` | Runs SPA checks and server tests. |
| Workspace build | `make build` | Builds available packages. |
| Web build | `make spa-build` | Runs type-check and production build. |
| Web full check | `make spa-check` | Runs type-check, unit tests, and build. |
| Server test | `make server-test` | Runs `go test ./...` from `packages/server`. |
| Server build | `make server-build` | Builds `packages/server/bin/server`. |
| Server run | `make server-run` | Starts the local Go server. Use `SERVER_HOST`, `SERVER_PORT`, and `DATABASE_URL` to override defaults. |
| CLI | Not available yet | Future package: `packages/cli`. Expected verification after scaffolding: `go test ./...` from that package. |

## Frontend notes

- `packages/web-spa` already includes `@lucide/vue` (`^1.21.0`) for future Vue icon work.
- Keep Vue code in Vue 3 Composition API style with `<script setup lang="ts">` unless the package establishes a different convention.

## Source of truth

- Architecture/specification: `docs/docs/SPECS.md`.
- Implementation plan: `TODO.md`.
