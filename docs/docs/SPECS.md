# Co-Review v2 — Arquitectura completa

> Go server · Vue 3 UI · CLI · Telegram bot · Framework 4R · Harness system · Memoria por repo · Comentarios inline aprobables

---

## Índice

1. [Por qué Go (y no PHP)](#1-por-qué-go-y-no-php)
2. [Visión general del sistema](#2-visión-general-del-sistema)
3. [Stack tecnológico](#3-stack-tecnológico)
4. [Skills 4R — definición completa](#4-skills-4r--definición-completa)
5. [Harness system](#5-harness-system)
6. [Model Provider — AI SDK intercambiable](#6-model-provider--ai-sdk-intercambiable)
7. [Memoria por repo](#7-memoria-por-repo)
8. [Flujo de review y publicación](#8-flujo-de-review-y-publicación)
9. [Comentarios inline en MR](#9-comentarios-inline-en-mr)
10. [Schema de base de datos](#10-schema-de-base-de-datos)
11. [API REST del server](#11-api-rest-del-server)
12. [SSE — eventos en tiempo real](#12-sse--eventos-en-tiempo-real)
13. [Vue 3 UI — diseño y UX](#13-vue-3-ui--diseño-y-ux)
14. [CLI — diseño y modos](#14-cli--diseño-y-modos)
15. [Telegram bot](#15-telegram-bot)
16. [UX flows especiales](#16-ux-flows-especiales)
17. [Estructura del proyecto](#17-estructura-del-proyecto)
18. [Roadmap de implementación](#18-roadmap-de-implementación)

---

## 1. Por qué Go (y no PHP)

PHP moderno (Octane, Swoole) puede hacer async, pero es ciudadano de segunda clase. Para este sistema los blockers concretos son:

| Necesidad | PHP | Go |
|-----------|-----|----|
| SSE / streaming nativo | Workaround con Swoole | `http.Flusher` nativo |
| Goroutines para 4 subagentes en paralelo | Procesos/colas externas | `sync.WaitGroup` + channels |
| Binario único sin runtime | No | Sí — `go build` |
| SDKs de IA (Anthropic, OpenAI, Gemini) | Wrappers HTTP manuales | SDKs oficiales o bien mantenidos |
| Webhook concurrentes sin bloqueo | Necesita FPM tuning | Trivial |
| CLI incluido en el mismo repo | Proyecto separado | `cmd/cli` con Cobra, mismo módulo |

**Decisión: Go para el server y CLI. Bun para scripts de tooling si se necesitan.**

---

## 2. Visión general del sistema

```
┌─────────────────────────────────────────────────────────────┐
│                      CLIENTES                               │
│  Vue 3 UI (web)  │  CLI (local/remoto)  │  Telegram bot    │
└────────┬─────────────────┬──────────────────────┬──────────┘
         │                 │                      │
         └─────────────────┴──────────────────────┘
                           │ HTTP + SSE
                    ┌──────▼──────┐
                    │  Go Server  │
                    │  (API REST) │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        Config DB     Review Engine   Notifier
        (SQLite/PG)        │
                    ┌──────┴──────┐
                    │ Orchestrator│
                    └──────┬──────┘
                           │ (paralelo via goroutines)
          ┌────────┬────────┼────────┬────────┐
          ▼        ▼        ▼        ▼
      Harness  Harness  Harness  Harness
      [R1Risk] [R2Read] [R3Rel]  [R4Res]
          │        │        │        │
          └────────┴────────┴────────┘
                    │ ModelProvider (intercambiable)
          ┌─────────┼─────────┐
          ▼         ▼         ▼
       Claude    OpenAI    Gemini / Groq / Ollama

                    │ Memoria
              ┌─────▼──────┐
              │  Engram /  │
              │  SQLite    │
              │ (por repo) │
              └────────────┘
```

---

## 3. Stack tecnológico

| Capa | Tecnología | Justificación |
|------|-----------|---------------|
| Server | Go 1.23 + `net/http` + Chi router | Rendimiento, goroutines, binario único |
| AI SDK | `github.com/anthropics/anthropic-sdk-go` + `github.com/openai/openai-go` | SDKs oficiales; Gemini/Groq vía OpenAI-compat |
| Base de datos | SQLite (dev) → Postgres (prod) vía `pgx` | `modernc.org/sqlite` sin CGO |
| Migraciones | `golang-migrate` | SQL puro, reversible |
| Memoria | SQLite por repo + opción Engram MCP | Ver sección 7 |
| UI web | Vue 3 + Vite + Pinia + Tailwind | Reactivo, ligero, SSE-friendly |
| CLI | Go + Cobra + `charmbracelet/bubbletea` | Interactividad, spinners, tablas |
| Telegram | `go-telegram-bot-api/telegram-bot-api` | Bien mantenido, async |
| SSE | Go nativo (`http.Flusher`) | Sin dependencias adicionales |
| Validación output AI | `santhosh-tekuri/jsonschema` | Validar JSON del modelo antes de procesar |

---

## 4. Skills 4R — definición completa

Las skills viven como archivos `.md` en `skills/` y se cargan al arrancar el server. El formato es el que ya tienes, con campos adicionales para el harness.

### Campos del frontmatter extendido

```yaml
---
name: review-risk
description: "..."
model: inherit          # inherit = usa el modelo del repo; o forzar uno: "claude-sonnet-4-6"
readonly: true
background: false
harness:
  timeout_seconds: 45
  max_retries: 2
  output_schema: risk   # referencia al JSON schema de validación
  require_evidence: true
  min_findings_quality: strict  # strict | lenient
memory:
  inject_context: true          # inyectar memoria del repo en el prompt
  save_findings: true           # guardar hallazgos en memoria
---
```

### R1 Risk — ampliado

```markdown
---
name: review-risk
description: R1 Risk — security, privilege boundaries, data exposure, dependency risks, merge-blocking vulnerabilities.
model: inherit
harness:
  timeout_seconds: 45
  max_retries: 2
  output_schema: risk
  require_evidence: true
---

You are **R1 Risk**, a read-only reviewer. Find security risks; do not fix them.

## Context injection (from repo memory)
{MEMORY_CONTEXT}
<!-- Si existe memoria del repo, aquí se inyectan decisiones aceptadas,
     riesgos previamente aprobados, y patrones conocidos del codebase -->

## Review rules

- Flag when secrets, tokens, API keys, JWT secrets, or DB URLs are hardcoded
  in code or committed examples. Include the exact line as evidence.
- Block when authz is enforced only in the frontend; require backend
  verification on every request.
- Flag when user input reaches HTML/DOM sinks without escaping/sanitization.
- Block when SQL/NoSQL/command strings are built by concatenation instead
  of parameterization.
- Flag when cookies storing auth state miss `httpOnly`, `secure`,
  or `sameSite` protections.
- Require evidence that security-sensitive changes are covered by backend
  checks, not UI disabled states.
- Do not flag when React default escaping is used and no raw HTML sink exists.
- Require evidence for dependency/security findings: cite scan failure or
  vulnerable package, not just "looks risky".
- Do NOT re-flag findings listed in {ACCEPTED_DECISIONS} from repo memory.
- Flag when `.env.example` or documentation contains real credential values.
- Block when CORS is configured with wildcard origin + credentials: true.
- Flag when JWT verification skips signature check (`alg: none` or
  ignored `verify` flag).
- Flag when file uploads have no MIME type or size validation.

## Inline comment format

For each finding that references specific code, produce an inline comment:
- file: exact path
- line_start / line_end: affected range
- suggestion_snippet: corrected code snippet (short, focused on the issue)

## Output contract

Each finding must include:
- severity: BLOCKER | CRITICAL | WARNING | SUGGESTION
- file, line_start, line_end
- evidence: exact code excerpt causing the finding
- why: one sentence explaining the risk
- suggestion_snippet: corrected or safe version (≤ 15 lines)
- inline_comment: true | false (whether to post as MR line comment)

If clean: `No findings.`
```

### R2 Readability — ampliado

```markdown
---
name: review-readability
description: R2 Readability — naming, complexity, intention, maintainability, review size, context clarity.
model: inherit
harness:
  timeout_seconds: 40
  max_retries: 2
  output_schema: readability
  require_evidence: true
---

You are **R2 Readability**, a read-only reviewer. Find clarity problems; do not fix them.

## Context injection
{MEMORY_CONTEXT}

## Review rules

- Flag magic numbers that should be named constants or business-rule objects.
- Flag long parameter lists (> 4 params) that should be parameter objects.
- Flag duplicated logic across components/hooks/modules; cite both locations.
- Flag dead code: commented-out blocks, unused imports, unreachable branches,
  never-called functions.
- Flag naming that hides intent or requires comment-heavy explanation.
- Flag PR/context explanation that is too vague to review safely.
- Require evidence for "too complex" claims: cite exact function, branch count,
  or repeated pattern with file:line.
- Do not flag a small helper or inline constant that is clear and self-explanatory.
- Do NOT re-flag patterns listed as accepted in {ACCEPTED_DECISIONS}.
- Flag functions longer than 50 lines without clear decomposition rationale.
- Flag boolean parameters that hide branching logic (prefer enum or strategy).
- Flag inconsistent naming conventions within the same module.
- Flag missing JSDoc/GoDoc on exported functions with non-obvious contracts.

## Output contract

Each finding must include:
- severity: BLOCKER | CRITICAL | WARNING | SUGGESTION
- file, line_start, line_end
- evidence: exact code excerpt
- why: one sentence on the clarity problem
- suggestion_snippet: clearer version (≤ 15 lines)
- inline_comment: true | false
```

### R3 Reliability — ampliado

```markdown
---
name: review-reliability
description: R3 Reliability — behavior-first tests, coverage value, edge cases, determinism, contracts, regressions.
model: inherit
harness:
  timeout_seconds: 50
  max_retries: 2
  output_schema: reliability
  require_evidence: true
---

You are **R3 Reliability**, a read-only reviewer. Find test and behavior risks; do not fix them.

## Context injection
{MEMORY_CONTEXT}

## Review rules

- Block behavior changes without tests asserting externally visible contract.
- Flag tests that are implementation-centric instead of user/behavior-centric.
- Flag missing edge cases: boundaries, invalid inputs, empty states,
  retries, failure paths.
- Block when CI can pass with `test.only`; require `forbidOnly` in CI configs.
- Flag misallocated test coverage: too much E2E where unit/integration suffices.
- Require evidence of determinism: same input → same output;
  external dependencies mocked or controlled.
- Flag weak selectors in UI tests; prefer semantic/user-visible queries.
- Do not flag intentional reliance on built-in async waiting over custom polling.
- Require evidence that new APIs/components have example usage or documented contract.
- Do NOT re-flag patterns listed as accepted in {ACCEPTED_DECISIONS}.
- Flag tests with hardcoded timestamps or UUIDs without seed/mock.
- Block when error paths are untested in critical flows (auth, payments, data mutations).
- Flag when mocks return unrealistic success responses that hide real contracts.
- Flag absence of contract tests when a public API changes its response shape.

## Output contract

Each finding must include:
- severity: BLOCKER | CRITICAL | WARNING | SUGGESTION
- file, line_start, line_end
- evidence: exact code or test excerpt
- why: one sentence on the behavior/test risk
- suggestion_snippet: example test or fix (≤ 20 lines)
- inline_comment: true | false
```

### R4 Resilience — ampliado

```markdown
---
name: review-resilience
description: R4 Resilience — fallbacks, retry/backoff, graceful degradation, observability, load, rollback, SLO risks.
model: inherit
harness:
  timeout_seconds: 45
  max_retries: 2
  output_schema: resilience
  require_evidence: true
---

You are **R4 Resilience**, a read-only reviewer. Find operational failure risks; do not fix them.

## Context injection
{MEMORY_CONTEXT}

## Review rules

- Flag failures with no fallback, retry, or graceful-degradation path.
- Block when production error-rate or build/test thresholds are ignored.
  Anchors: test success < 95%, build success < 95%,
  prod error rate > 1% investigate, > 2% emergency, > 5% all hands.
- Flag releases that can regress without alerting/observability hooks.
- Require evidence for rollback/fix-forward readiness.
- Flag performance regressions exceeding user-visible budgets (LCP > 2.5s,
  API p95 > 500ms) or lacking measurement.
- Block when there is no production visibility for error/performance issues.
- Do not flag low-impact expected issues isolated by alert grouping.
- Require evidence of SLO/latency/load impact; not generic "might be slow".
- Do NOT re-flag decisions listed in {ACCEPTED_DECISIONS}.
- Flag when HTTP calls have no timeout configured.
- Flag when retry logic uses fixed delay instead of exponential backoff + jitter.
- Block when a new external dependency has no circuit breaker or fallback.
- Flag when health check endpoints don't validate downstream dependency status.
- Flag when structured logs are missing correlation IDs for distributed tracing.

## Output contract

Each finding must include:
- severity: BLOCKER | CRITICAL | WARNING | SUGGESTION
- file, line_start, line_end
- evidence: exact code excerpt
- why: one sentence on the operational risk
- suggestion_snippet: resilient version (≤ 15 lines)
- inline_comment: true | false
```

---

## 5. Harness system

El harness envuelve cada subagente. Es la capa entre el orquestador y el model provider.

### Responsabilidades

1. **Control de ejecución:** timeout, reintentos con backoff, logging de duración
2. **Validación de output:** el JSON del modelo se valida contra un JSON Schema antes de procesarse
3. **Fallback:** si el modelo falla después de reintentos, produce un resultado de error estructurado (no panic)
4. **Métricas:** emite eventos internos de duración, tokens, reintentos para el dashboard

### Interfaz en Go

```go
// internal/harness/harness.go

type HarnessConfig struct {
    TimeoutSeconds  int
    MaxRetries      int
    OutputSchema    string // nombre del schema en schemas/
    RequireEvidence bool
}

type HarnessResult struct {
    Dimension string
    Output    *AgentOutput  // nil si error
    Error     *HarnessError
    Attempts  int
    Duration  time.Duration
    Tokens    int
}

type HarnessError struct {
    Code    string // TIMEOUT | INVALID_OUTPUT | PROVIDER_ERROR | MAX_RETRIES
    Message string
    Raw     string // respuesta cruda del modelo para debug
}

func Run(
    ctx context.Context,
    cfg HarnessConfig,
    provider ModelProvider,
    prompt AgentPrompt,
) HarnessResult {
    var lastErr error
    start := time.Now()

    for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
        timeoutCtx, cancel := context.WithTimeout(
            ctx, time.Duration(cfg.TimeoutSeconds)*time.Second,
        )
        defer cancel()

        raw, tokens, err := provider.Complete(timeoutCtx, prompt.System, prompt.User)
        if err != nil {
            lastErr = err
            // backoff: 1s, 2s, 4s...
            time.Sleep(time.Duration(math.Pow(2, float64(attempt-1))) * time.Second)
            continue
        }

        // Validar JSON schema
        output, validErr := validateAndParse(raw, cfg.OutputSchema)
        if validErr != nil {
            lastErr = validErr
            // prompt de corrección al modelo (1 retry adicional)
            prompt.User = buildCorrectionPrompt(raw, validErr)
            continue
        }

        return HarnessResult{
            Output:   output,
            Attempts: attempt,
            Duration: time.Since(start),
            Tokens:   tokens,
        }
    }

    return HarnessResult{
        Error: &HarnessError{
            Code:    classifyError(lastErr),
            Message: lastErr.Error(),
        },
        Attempts: cfg.MaxRetries,
        Duration: time.Since(start),
    }
}
```

### JSON Schema de validación (ejemplo: risk)

```json
// schemas/risk.json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["dimension", "findings", "summary", "verdict"],
  "properties": {
    "dimension": { "type": "string", "enum": ["risk"] },
    "score": { "type": "integer", "minimum": 0, "maximum": 100 },
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["severity", "file", "evidence", "why", "inline_comment"],
        "properties": {
          "severity": { "enum": ["BLOCKER", "CRITICAL", "WARNING", "SUGGESTION"] },
          "file": { "type": "string" },
          "line_start": { "type": "integer" },
          "line_end": { "type": "integer" },
          "evidence": { "type": "string", "minLength": 5 },
          "why": { "type": "string", "minLength": 10 },
          "suggestion_snippet": { "type": "string" },
          "inline_comment": { "type": "boolean" }
        }
      }
    },
    "summary": { "type": "string" },
    "verdict": { "enum": ["pass", "needs_changes", "block"] }
  }
}
```

---

## 6. Model Provider — AI SDK intercambiable

### Interfaz en Go

```go
// internal/provider/provider.go

type CompletionRequest struct {
    System    string
    User      string
    MaxTokens int
}

type CompletionResponse struct {
    Content    string
    ModelUsed  string
    InputTokens  int
    OutputTokens int
}

type ModelProvider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
    Name() string
    SupportedModels() []ModelInfo
}

type ModelInfo struct {
    ID          string // "claude-sonnet-4-6"
    DisplayName string // "Claude Sonnet 4.6"
    ContextWindow int
    CostPer1kInput  float64 // USD
    CostPer1kOutput float64
}
```

### Implementaciones

```go
// internal/provider/claude.go
type ClaudeProvider struct {
    client *anthropic.Client
    model  string
}

func NewClaude(apiKey, model string) *ClaudeProvider {
    return &ClaudeProvider{
        client: anthropic.NewClient(option.WithAPIKey(apiKey)),
        model:  model,
    }
}

func (p *ClaudeProvider) Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error) {
    msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.Model(p.model),
        MaxTokens: int64(req.MaxTokens),
        System:    []anthropic.TextBlockParam{{Text: req.System}},
        Messages:  []anthropic.MessageParam{anthropic.NewUserMessage(req.User)},
    })
    if err != nil {
        return CompletionResponse{}, err
    }
    return CompletionResponse{
        Content:      msg.Content[0].Text,
        ModelUsed:    p.model,
        InputTokens:  int(msg.Usage.InputTokens),
        OutputTokens: int(msg.Usage.OutputTokens),
    }, nil
}

func (p *ClaudeProvider) SupportedModels() []ModelInfo {
    return []ModelInfo{
        {ID: "claude-opus-4-6", DisplayName: "Claude Opus 4.6", ContextWindow: 200000},
        {ID: "claude-sonnet-4-6", DisplayName: "Claude Sonnet 4.6", ContextWindow: 200000},
        {ID: "claude-haiku-4-5-20251001", DisplayName: "Claude Haiku 4.5", ContextWindow: 200000},
    }
}
```

```go
// internal/provider/openai_compat.go
// Sirve para OpenAI, Groq, Ollama, Mistral — todos son OpenAI-compat

type OpenAICompatProvider struct {
    client  *openai.Client
    model   string
    name    string
    models  []ModelInfo
}

func NewOpenAICompat(name, baseURL, apiKey, model string, models []ModelInfo) *OpenAICompatProvider {
    return &OpenAICompatProvider{
        client: openai.NewClient(
            option.WithAPIKey(apiKey),
            option.WithBaseURL(baseURL),
        ),
        model:  model,
        name:   name,
        models: models,
    }
}
// Ejemplos de instanciación:
//
// Groq:   NewOpenAICompat("groq", "https://api.groq.com/openai/v1", key, model, groqModels)
// Ollama: NewOpenAICompat("ollama", "http://localhost:11434/v1", "ollama", model, ollamaModels)
// Mistral:NewOpenAICompat("mistral", "https://api.mistral.ai/v1", key, model, mistralModels)
```

### Registry y resolución dinámica

```go
// internal/provider/registry.go

type ProviderRegistry struct {
    providers map[string]func(cfg ProviderConfig) (ModelProvider, error)
}

func DefaultRegistry() *ProviderRegistry {
    r := &ProviderRegistry{providers: make(map[string]func(ProviderConfig) (ModelProvider, error))}

    r.Register("claude", func(cfg ProviderConfig) (ModelProvider, error) {
        return NewClaude(cfg.APIKey, cfg.ModelName), nil
    })
    r.Register("openai", func(cfg ProviderConfig) (ModelProvider, error) {
        return NewOpenAICompat("openai", "https://api.openai.com/v1", cfg.APIKey, cfg.ModelName, openAIModels()), nil
    })
    r.Register("groq", func(cfg ProviderConfig) (ModelProvider, error) {
        return NewOpenAICompat("groq", "https://api.groq.com/openai/v1", cfg.APIKey, cfg.ModelName, groqModels()), nil
    })
    r.Register("ollama", func(cfg ProviderConfig) (ModelProvider, error) {
        baseURL := cfg.ExtraParams["base_url"]
        if baseURL == "" { baseURL = "http://localhost:11434/v1" }
        return NewOpenAICompat("ollama", baseURL, "ollama", cfg.ModelName, ollamaModels(cfg)), nil
    })
    r.Register("gemini", func(cfg ProviderConfig) (ModelProvider, error) {
        return NewOpenAICompat("gemini", "https://generativelanguage.googleapis.com/v1beta/openai", cfg.APIKey, cfg.ModelName, geminiModels()), nil
    })

    return r
}
```

---

## 7. Memoria por repo

### Qué se guarda

| Tipo | Descripción | TTL |
|------|-------------|-----|
| `accepted_decision` | Riesgo o hallazgo explícitamente aprobado por el equipo | Indefinido (manual) |
| `codebase_pattern` | Convención detectada automáticamente (naming, estructura) | 30 días sin actualización |
| `finding_history` | Cada hallazgo de cada review con su estado | 90 días |
| `tech_debt_trend` | Acumulación de hallazgos por dimensión a lo largo del tiempo | Permanente (para métricas) |

### Motor de memoria: opción recomendada

**Opción A — SQLite local (incluida, zero-config):**
Una tabla `repo_memory` con embeddings simples (búsqueda por keywords). Funciona sin servicios externos. Suficiente para la mayoría de casos.

**Opción B — Engram MCP:**
Engram es un servidor MCP que provee memoria semántica persistente. Se conecta al server de Go como tool call durante la construcción del prompt. Ventaja: búsqueda por similitud semántica, no solo keywords. Desventaja: servicio adicional a levantar.

**Recomendación: Implementar Opción A primero. Agregar Opción B como plugin activable por repo.**

```sql
-- Tabla de memoria por repo
CREATE TABLE repo_memory (
    id           TEXT PRIMARY KEY,
    repo_id      TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    type         TEXT NOT NULL, -- accepted_decision | codebase_pattern | finding_history | tech_debt_trend
    key          TEXT NOT NULL, -- identificador semántico breve
    content      TEXT NOT NULL, -- texto completo para inyectar en prompt
    dimension    TEXT,          -- risk | readability | reliability | resilience | null (global)
    source_mr    TEXT,          -- MR donde se originó
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at   TEXT           -- null = indefinido
);

CREATE INDEX idx_repo_memory_repo_type ON repo_memory(repo_id, type);
```

### Inyección en el prompt

```go
// internal/memory/injector.go

func BuildContextForPrompt(repoID, dimension string) string {
    decisions := getAcceptedDecisions(repoID)
    patterns := getCodebasePatterns(repoID, dimension)

    if len(decisions) == 0 && len(patterns) == 0 {
        return ""
    }

    var sb strings.Builder
    sb.WriteString("## Repo memory context\n\n")

    if len(decisions) > 0 {
        sb.WriteString("### Accepted decisions (do NOT re-flag these):\n")
        for _, d := range decisions {
            sb.WriteString(fmt.Sprintf("- %s\n", d.Content))
        }
    }

    if len(patterns) > 0 {
        sb.WriteString("\n### Known codebase patterns (use as context):\n")
        for _, p := range patterns {
            sb.WriteString(fmt.Sprintf("- %s\n", p.Content))
        }
    }

    return sb.String()
}
```

### Flujo de "marcar como aceptado"

Desde la UI o CLI, al revisar un hallazgo antes de publicar:
1. El usuario presiona "Marcar como decisión aceptada" en un finding
2. Se guarda en `repo_memory` con `type = accepted_decision`
3. Las próximas reviews de ese repo inyectarán esa decisión en el prompt de los 4 subagentes
4. El finding no volverá a aparecer a menos que el código cambie significativamente

---

## 8. Flujo de review y publicación

### Estados de una review

```
PENDING → RUNNING → GENERATED → [PUBLISHING] → PUBLISHED
                        │
                        └──(si auto_publish=false)──►
                                AWAITING_APPROVAL
                                   │
                        ┌──────────┼──────────┐
                        ▼          ▼          ▼
                   PUBLISH_ALL  SEQUENTIAL  DISCARD
                   (todo)       (uno a uno)
```

### Publicación secuencial de comentarios

Cuando `auto_publish = false` y el usuario elige modo secuencial:

1. El server agrupa los comentarios por severidad: BLOCKER → CRITICAL → WARNING → SUGGESTION
2. La UI (o el CLI interactivo) muestra cada comentario uno a uno:
   - Previsualización del snippet de código afectado
   - La sugerencia del modelo
   - Botones: **Publicar** / **Marcar como aceptado** / **Descartar**
3. Los comentarios aprobados se acumulan y se publican en batch al finalizar la sesión
4. Los marcados como "aceptados" se guardan en `repo_memory`

### API de publicación

```
POST /reviews/{id}/publish
Body: {
  "mode": "all" | "sequential",
  "comment_ids": ["id1", "id2"]  // solo para mode=sequential, los aprobados
}
```

---

## 9. Comentarios inline en MR

### Formato interno (JSON)

```json
{
  "id": "c_abc123",
  "review_id": "r_xyz",
  "dimension": "risk",
  "severity": "BLOCKER",
  "file": "src/auth/login.go",
  "line_start": 42,
  "line_end": 44,
  "evidence": "db.Query(\"SELECT * FROM users WHERE id=\" + userId)",
  "why": "SQL injection vulnerability: user input concatenated directly into query.",
  "suggestion_snippet": "// Safe version:\nrow := db.QueryRow(\"SELECT * FROM users WHERE id = $1\", userId)",
  "inline_comment": true,
  "status": "pending"  // pending | approved | accepted_decision | discarded
}
```

### Formato del comentario publicado en GitLab/GitHub

El comentario se formatea en Markdown para ser legible directo en la plataforma:

```markdown
<!-- R1 Risk — BLOCKER -->
## ⛔ SQL Injection — R1 Risk

**Evidencia:**
```go
db.Query("SELECT * FROM users WHERE id=" + userId)
```

**Por qué importa:** User input concatenated directly into SQL query — allows arbitrary query execution.

**Sugerencia:**
```go
row := db.QueryRow("SELECT * FROM users WHERE id = $1", userId)
```

---
*Co-Review · R1 Risk · [Ver reporte completo](https://your-server/reviews/r_xyz)*
```

### Adaptadores por plataforma

```go
// internal/platform/gitlab.go

func (g *GitLabClient) PostInlineComment(ctx context.Context, req InlineCommentRequest) error {
    // Usa GitLab Discussion API para comentarios en línea de código
    // POST /projects/:id/merge_requests/:mr_iid/discussions
    body := map[string]any{
        "body": formatComment(req),
        "position": map[string]any{
            "position_type": "text",
            "new_path":      req.File,
            "new_line":      req.LineStart,
            "base_sha":      req.BaseSHA,
            "head_sha":      req.HeadSHA,
            "start_sha":     req.StartSHA,
        },
    }
    // ...
}

// internal/platform/github.go

func (g *GitHubClient) PostInlineComment(ctx context.Context, req InlineCommentRequest) error {
    // Usa GitHub Pull Request Review Comments API
    // POST /repos/:owner/:repo/pulls/:pull_number/comments
    // ...
}
```

---

## 10. Schema de base de datos

```sql
-- repos
CREATE TABLE repos (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    url             TEXT NOT NULL,
    platform        TEXT NOT NULL,       -- gitlab | github
    default_branch  TEXT NOT NULL DEFAULT 'main',
    webhook_secret  TEXT,
    auto_publish    INTEGER NOT NULL DEFAULT 0,
    publish_mode    TEXT NOT NULL DEFAULT 'sequential', -- all | sequential
    active          INTEGER NOT NULL DEFAULT 1,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

-- model_configs
CREATE TABLE model_configs (
    id           TEXT PRIMARY KEY,
    repo_id      TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL,   -- claude | openai | gemini | groq | ollama | mistral
    model_name   TEXT NOT NULL,
    api_key_env  TEXT,            -- nombre de la var de entorno (no el valor)
    extra_params TEXT,            -- JSON: base_url, temperature, etc.
    is_active    INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

-- channels
CREATE TABLE channels (
    id     TEXT PRIMARY KEY,
    type   TEXT NOT NULL, -- telegram | slack | email | webhook
    name   TEXT NOT NULL UNIQUE,
    config TEXT NOT NULL, -- JSON según tipo
    active INTEGER NOT NULL DEFAULT 1
);

-- repo_channels
CREATE TABLE repo_channels (
    repo_id    TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    events     TEXT NOT NULL DEFAULT '["review.generated","review.published"]',
    PRIMARY KEY (repo_id, channel_id)
);

-- skills
CREATE TABLE skills (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,  -- review-risk | review-readability | etc.
    dimension   TEXT NOT NULL,
    file_path   TEXT NOT NULL,         -- ruta al .md en el filesystem
    active      INTEGER NOT NULL DEFAULT 1,
    loaded_at   TEXT                   -- última vez que se cargó del archivo
);

-- repo_skills (override por repo)
CREATE TABLE repo_skills (
    repo_id   TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    skill_id  TEXT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    priority  INTEGER NOT NULL DEFAULT 100,
    PRIMARY KEY (repo_id, skill_id)
);

-- reviews
CREATE TABLE reviews (
    id           TEXT PRIMARY KEY,
    repo_id      TEXT NOT NULL REFERENCES repos(id),
    mr_id        TEXT NOT NULL,
    mr_url       TEXT,
    mr_title     TEXT,
    base_sha     TEXT,
    head_sha     TEXT,
    start_sha    TEXT,
    model_used   TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    -- pending | running | generated | awaiting_approval | publishing | published | error
    auto_publish INTEGER NOT NULL DEFAULT 0,
    scores       TEXT,  -- JSON: {risk, readability, reliability, resilience}
    verdict      TEXT,  -- approved | changes_requested | blocked | error
    created_at   TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT
);

-- review_comments (comentarios inline generados)
CREATE TABLE review_comments (
    id               TEXT PRIMARY KEY,
    review_id        TEXT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    dimension        TEXT NOT NULL,
    severity         TEXT NOT NULL,
    file             TEXT NOT NULL,
    line_start       INTEGER,
    line_end         INTEGER,
    evidence         TEXT NOT NULL,
    why              TEXT NOT NULL,
    suggestion_snippet TEXT,
    status           TEXT NOT NULL DEFAULT 'pending',
    -- pending | approved | accepted_decision | discarded | published
    platform_comment_id TEXT, -- ID del comentario en GitLab/GitHub tras publicar
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    published_at     TEXT
);

-- repo_memory
CREATE TABLE repo_memory (
    id          TEXT PRIMARY KEY,
    repo_id     TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    key         TEXT NOT NULL,
    content     TEXT NOT NULL,
    dimension   TEXT,
    source_mr   TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    expires_at  TEXT
);
```

---

## 11. API REST del server

```
# Providers (para UI de selección)
GET  /api/v1/providers                    Lista providers soportados
GET  /api/v1/providers/:name/models       Lista modelos del provider

# Repos
POST   /api/v1/repos                      Crear repo
GET    /api/v1/repos                      Listar repos
GET    /api/v1/repos/:id                  Detalle
PATCH  /api/v1/repos/:id                  Actualizar
DELETE /api/v1/repos/:id                  Eliminar
POST   /api/v1/repos/infer                Inferir datos desde URL (nombre, rama, plataforma)
GET    /api/v1/repos/:id/branches         Listar ramas del repo (via plataforma API)

# Model config por repo
GET    /api/v1/repos/:id/model            Config de modelo activo
PUT    /api/v1/repos/:id/model            Cambiar modelo

# Channels
POST   /api/v1/channels                   Crear canal
GET    /api/v1/channels                   Listar canales
PATCH  /api/v1/channels/:id               Actualizar canal
DELETE /api/v1/channels/:id               Eliminar canal
POST   /api/v1/channels/telegram/discover Descubrir chats/grupos del bot (token requerido)

# Repo ↔ Channel
POST   /api/v1/repos/:id/channels/:ch_id  Asignar canal
PATCH  /api/v1/repos/:id/channels/:ch_id  Actualizar eventos
DELETE /api/v1/repos/:id/channels/:ch_id  Desasignar canal

# Skills
GET    /api/v1/skills                     Listar skills (filtro: ?dimension=)
POST   /api/v1/skills/reload              Recargar skills desde filesystem

# Reviews
POST   /api/v1/reviews                    Iniciar review manual
GET    /api/v1/reviews                    Historial (?repo_id=&limit=)
GET    /api/v1/reviews/:id                Detalle completo
GET    /api/v1/reviews/:id/events         SSE stream de progreso
GET    /api/v1/reviews/:id/comments       Lista de comentarios generados
POST   /api/v1/reviews/:id/publish        Publicar (body: mode, comment_ids)
PATCH  /api/v1/reviews/:id/comments/:cid  Actualizar status de un comentario

# Memoria por repo
GET    /api/v1/repos/:id/memory           Listar entradas de memoria
POST   /api/v1/repos/:id/memory           Agregar entrada manual
DELETE /api/v1/repos/:id/memory/:mid      Eliminar entrada

# Webhooks (recepción de eventos del repo)
POST   /webhooks/gitlab                   Webhook GitLab (valida X-Gitlab-Token)
POST   /webhooks/github                   Webhook GitHub (valida X-Hub-Signature-256)
```

---

## 12. SSE — eventos en tiempo real

El endpoint `GET /api/v1/reviews/:id/events` mantiene una conexión SSE abierta durante la review.

### Eventos emitidos

```
event: review.started
data: {"review_id":"r_abc","repo":"backend","mr_id":"42","model":"claude-sonnet-4-6"}

event: agent.started
data: {"dimension":"risk","attempt":1}

event: agent.completed
data: {"dimension":"risk","score":72,"findings_count":3,"duration_ms":8420,"tokens":1240}

event: agent.started
data: {"dimension":"readability","attempt":1}

event: agent.error
data: {"dimension":"reliability","error":"TIMEOUT","attempt":1,"retrying":true}

event: agent.completed
data: {"dimension":"reliability","score":88,"findings_count":1,"duration_ms":12100,"tokens":980}

event: review.generated
data: {"review_id":"r_abc","verdict":"changes_requested","scores":{"risk":72,"readability":85,"reliability":88,"resilience":91},"comments_count":7,"auto_publish":false}

event: review.published
data: {"review_id":"r_abc","published_comments":5,"discarded":2}
```

### En Vue 3 (UI)

```typescript
// composables/useReviewStream.ts
export function useReviewStream(reviewId: string) {
  const status = ref<ReviewStatus>('pending')
  const agents = ref<AgentProgress[]>([])
  const review = ref<Review | null>(null)

  const es = new EventSource(`/api/v1/reviews/${reviewId}/events`)

  es.addEventListener('agent.started', (e) => {
    const data = JSON.parse(e.data)
    agents.value.push({ dimension: data.dimension, status: 'running', attempt: data.attempt })
  })

  es.addEventListener('agent.completed', (e) => {
    const data = JSON.parse(e.data)
    const agent = agents.value.find(a => a.dimension === data.dimension)
    if (agent) Object.assign(agent, { status: 'done', score: data.score, duration: data.duration_ms })
  })

  es.addEventListener('review.generated', (e) => {
    review.value = JSON.parse(e.data)
    status.value = 'generated'
    es.close()
  })

  return { status, agents, review }
}
```

### En CLI (polling async)

El CLI no bloquea. Al disparar una review:

```bash
$ review-ctl review run --repo backend --mr 42
✓ Review iniciada: r_abc123
  Consultar progreso: review-ctl review status r_abc123
  O seguir en tiempo real: review-ctl review watch r_abc123
```

```bash
$ review-ctl review watch r_abc123
⟳ R1 Risk        [■■■□□] running...
✓ R2 Readability [■■■■■] done — score 85, 2 findings (8.4s)
⟳ R3 Reliability [■■□□□] running...
✓ R4 Resilience  [■■■■■] done — score 91, 1 finding (6.1s)
✓ R1 Risk        [■■■■■] done — score 72, 3 findings (11.2s)
✓ R3 Reliability [■■■■■] done — score 88, 1 finding (12.1s)

Review generada — veredicto: CHANGES REQUESTED
7 comentarios pendientes de aprobación.
Publicar: review-ctl review publish r_abc123
```

---

## 13. Vue 3 UI — diseño y UX

### Páginas principales

```
/ ─── Dashboard (repos activos, reviews recientes, métricas de deuda técnica)
/repos ─── Lista de repos
/repos/new ─── Wizard de creación de repo
/repos/:id ─── Detalle repo (config, historial, memoria)
/repos/:id/model ─── Configurar modelo
/channels ─── Gestión de canales
/channels/new ─── Wizard de canal
/skills ─── Vista de skills cargadas (solo lectura, desde filesystem)
/reviews ─── Historial global
/reviews/:id ─── Detalle de review + flujo de aprobación de comentarios
/settings ─── Config global (proveedores, API keys)
```

### UX flow: agregar repositorio

```
Paso 1: Ingresar URL
  [ https://gitlab.com/company/backend-api          ]
  [Inferir información ↵]
         │
         ▼ (llama a POST /api/v1/repos/infer)
Paso 2: Confirmar datos inferidos (editables)
  Nombre:    backend-api          [editable]
  Plataforma: GitLab              [editable]
  Rama:      main                 [dropdown desde API del repo]
  Token:     [ingresado o heredado de config global]

Paso 3: Seleccionar modelo
  Proveedor: [Claude ▼]           [dropdown: Claude | OpenAI | Gemini | Groq | Ollama]
  Modelo:    [Claude Sonnet 4.6 ▼] [carga dinámicamente según proveedor]

Paso 4: Canales de notificación
  [+ Asignar canal existente] o [+ Crear canal nuevo]

Paso 5: Opciones de publicación
  ○ Auto-publicar al generar
  ● Requiere aprobación
     ○ Publicar todos a la vez
     ● Revisar uno a uno (secuencial)
```

### UX flow: agregar canal Telegram

```
Tipo: Telegram

Token del bot: [BotFather token ____________]
               [Buscar chats y grupos ↵]
                      │
                      ▼ (llama a POST /api/v1/channels/telegram/discover)
┌──────────────────────────────────────────┐
│ Chats y grupos disponibles               │
│                                          │
│ ● Mi grupo de desarrollo    -1001234567  │
│ ○ Canal de alertas          -1009876543  │
│ ○ Chat con el bot           123456789   │
└──────────────────────────────────────────┘
         │ [Seleccionar]
         ▼
Chat ID: -1001234567  [autocompletado]
Thread ID: _________  [opcional, para supergrupos con topics]
Nombre: mi-grupo-dev  [sugerido, editable]
```

### UX flow: review en progreso

```
┌─────────────────────────────────────────────────┐
│  Review en progreso — MR !42 "Add user auth"    │
│  backend-api · claude-sonnet-4-6                │
│                                                 │
│  R1 Risk         ████████░░  running...  [~12s] │
│  R2 Readability  ██████████  ✓ 85 pts   [8.4s] │
│  R3 Reliability  █████░░░░░  running...  [~20s] │
│  R4 Resilience   ██████████  ✓ 91 pts   [6.1s] │
│                                                 │
│  2 de 4 agentes completados                     │
└─────────────────────────────────────────────────┘
```

### UX flow: aprobación secuencial de comentarios

```
Aprobando comentarios — 7 pendientes (5 restantes)

Comentario 2 de 7
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⛔ BLOCKER — R1 Risk
src/auth/login.go  líneas 42–44

Evidencia:
  42│ query := "SELECT * FROM users WHERE id=" + userId
  43│ rows, err := db.Query(query)
  44│ if err != nil { ... }

Por qué: SQL injection — input de usuario concatenado en la query.

Sugerencia:
  42│ rows, err := db.QueryRow(
  43│     "SELECT * FROM users WHERE id = $1", userId,
  44│ )

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[Publicar]  [Marcar como decisión aceptada]  [Descartar]  [Publicar todos restantes]
```

---

## 14. CLI — diseño y modos

### Modos de conexión

```toml
# ~/.config/review-ctl/config.toml

[connection]
mode = "local"    # local | remote

[local]
db_path = "~/.local/share/co-review/co_review.db"
skills_dir = "~/.config/co-review/skills"

[remote]
server_url = "https://co-review.mycompany.com"
api_token   = "eyJ..."
```

En modo `local`, el CLI levanta el engine directamente (importa los paquetes de Go del server como librería). En modo `remote`, todo va por HTTP al server. El CLI detecta si el server está corriendo; si no, en modo `local` lo puede iniciar como proceso background.

### Comandos principales

```bash
# Config
review-ctl config set-mode local|remote
review-ctl config set-server https://...

# Repos
review-ctl repo add --url https://gitlab.com/org/repo    # wizard interactivo
review-ctl repo list
review-ctl repo show <name>
review-ctl repo set-model <name> --provider claude --model claude-sonnet-4-6
review-ctl repo remove <name>

# Proveedores (UI interactiva con bubbletea)
review-ctl provider list                 # tabla de providers disponibles
review-ctl provider models claude        # lista modelos de un provider

# Canales
review-ctl channel add                   # wizard interactivo por tipo
review-ctl channel list
review-ctl channel telegram discover --token <bot_token>  # lista chats
review-ctl channel remove <name>

# Reviews
review-ctl review run --repo <name> --mr <id>   # dispara y retorna ID
review-ctl review watch <id>                     # sigue el progreso en tiempo real
review-ctl review status <id>                    # estado puntual
review-ctl review publish <id>                   # inicia flujo de publicación
review-ctl review publish <id> --all             # publica todo sin aprobación
review-ctl review history --repo <name>

# Memoria
review-ctl memory list --repo <name>
review-ctl memory accept --repo <name> --comment <comment_id>
review-ctl memory delete --repo <name> --id <memory_id>

# Skills
review-ctl skill list
review-ctl skill reload   # recarga desde filesystem
```

### Interactividad con Bubbletea

Para flujos complejos (add repo, add channel, publish secuencial), el CLI usa `charmbracelet/bubbletea` + `charmbracelet/huh` para forms interactivos en terminal. El resultado es idéntico al UX de la UI pero en texto.

---

## 15. Telegram bot

El bot interactúa con el server vía API REST. No tiene lógica propia de review.

### Comandos

```
/start          — bienvenida e instrucciones
/review         — disparar review: /review backend 42
/status         — estado de la última review de un repo: /status backend
/watch          — seguir review en progreso (mensajes de actualización)
/publish        — publicar review generada: /publish r_abc123
/publish_all    — publicar todos los comentarios sin revisión
/list_repos     — listar repos activos
/list_reviews   — últimas reviews
/memory         — gestionar memoria: /memory list backend
/help           — ayuda

# Comandos de configuración (solo desde chat privado con el bot)
/add_repo       — wizard de texto para agregar repo
/set_model      — cambiar modelo: /set_model backend claude claude-sonnet-4-6
```

### Notificaciones automáticas (outgoing)

Cuando una review termina, el bot envía:

```
⚠️ Review generada — MR !42
Repo: backend-api
Modelo: claude-sonnet-4-6

Risk        72 pts ⛔ BLOCKER (1)
Readability 85 pts ⚠️ WARNING (2)
Reliability 88 pts ⚠️ WARNING (1)
Resilience  91 pts ✅ OK

Veredicto: CHANGES REQUESTED
7 comentarios pendientes

[Ver reporte] [Publicar todo] [Revisar uno a uno]
```

Los botones inline de Telegram usan `callback_query` para disparar las acciones contra el server.

### El bot funciona sin el PC encendido

El bot corre en el server (o en un VPS/proceso cloud). Cuando mandas `/review backend 42` desde el móvil:
1. El bot recibe el comando
2. Llama a `POST /api/v1/reviews` en el server
3. El server ejecuta la review (los 4 subagentes en paralelo)
4. Cuando termina, el bot te manda el resumen con los botones de acción
5. Pulsas "Publicar todo" → el bot llama a `POST /api/v1/reviews/:id/publish`
6. El server publica los comentarios en GitLab/GitHub

Todo sin abrir el PC.

---

## 16. UX flows especiales

### Selección de proveedor y modelo (web y CLI)

**Web:**
```
Proveedor [Claude          ▼]
  └── Claude
  └── OpenAI
  └── Gemini
  └── Groq
  └── Ollama (local)
  └── Mistral

(al seleccionar Claude)
Modelo    [Claude Sonnet 4.6 ▼]
  └── Claude Opus 4.6        (200k ctx)
  └── Claude Sonnet 4.6      (200k ctx)  ← recomendado
  └── Claude Haiku 4.5       (200k ctx)  ← más rápido
```

Los modelos se cargan dinámicamente desde `GET /api/v1/providers/:name/models`. Para Ollama se hace un pull de la API local en `http://localhost:11434/api/tags`. Para el resto, lista hardcodeada actualizable.

**CLI:**
```bash
$ review-ctl provider list
┌─────────────┬──────────────────────┬─────────────┐
│ Provider    │ Status               │ Models      │
├─────────────┼──────────────────────┼─────────────┤
│ claude      │ ✓ API key configured │ 3 available │
│ openai      │ ✓ API key configured │ 8 available │
│ gemini      │ ✗ No API key        │ —           │
│ groq        │ ✓ API key configured │ 5 available │
│ ollama      │ ✓ Running locally    │ 2 loaded    │
└─────────────┴──────────────────────┴─────────────┘

$ review-ctl provider models claude
┌────────────────────────────────┬──────────────┬─────────────┐
│ Model                          │ Context      │ Speed       │
├────────────────────────────────┼──────────────┼─────────────┤
│ claude-opus-4-6                │ 200k tokens  │ slow        │
│ claude-sonnet-4-6  [default]   │ 200k tokens  │ medium      │
│ claude-haiku-4-5-20251001      │ 200k tokens  │ fast        │
└────────────────────────────────┴──────────────┴─────────────┘
```

### Inferencia de datos del repo desde URL

`POST /api/v1/repos/infer` recibe la URL y:
1. Detecta la plataforma (GitLab/GitHub) por el dominio
2. Extrae el namespace y el nombre del repo del path
3. Llama a la API de la plataforma con el token configurado para obtener: rama default, descripción, visibilidad
4. Retorna todo para que el usuario confirme antes de guardar

```json
// Request
{ "url": "https://gitlab.com/company/backend-api", "token": "glpat-..." }

// Response
{
  "name": "backend-api",
  "platform": "gitlab",
  "namespace": "company",
  "default_branch": "main",
  "description": "Core backend service",
  "private": true,
  "inferred_from": "api"  // api | url_parse
}
```

### Discover de chats de Telegram

`POST /api/v1/channels/telegram/discover` recibe el token del bot y llama a `getUpdates` de la Bot API para obtener los chats donde el bot tiene presencia. También llama a `getChat` para cada ID conocido.

```json
// Response
{
  "chats": [
    { "id": -1001234567, "type": "supergroup", "title": "Dev team", "topics_enabled": true },
    { "id": -1009876543, "type": "channel", "title": "Alertas producción" },
    { "id": 123456789,   "type": "private", "first_name": "Juan" }
  ]
}
```

**Nota:** Para que el bot aparezca en grupos, el usuario debe haberlo agregado al grupo antes. La UI muestra un aviso si la lista está vacía.

---

## 17. Estructura del proyecto

```
co-review/
├── cmd/
│   ├── server/         # main.go del server
│   │   └── main.go
│   └── cli/            # main.go del CLI
│       └── main.go
│
├── internal/
│   ├── api/            # HTTP handlers (Chi)
│   │   ├── repos.go
│   │   ├── channels.go
│   │   ├── reviews.go
│   │   ├── providers.go
│   │   ├── memory.go
│   │   └── sse.go      # SSE handler
│   │
│   ├── provider/       # Model Provider pattern
│   │   ├── provider.go # interface
│   │   ├── claude.go
│   │   ├── openai_compat.go
│   │   ├── gemini.go   # via openai-compat endpoint
│   │   ├── registry.go
│   │   └── models.go   # catálogo de modelos por provider
│   │
│   ├── harness/        # Harness system
│   │   ├── harness.go
│   │   ├── validator.go
│   │   └── schemas/    # JSON schemas de validación
│   │       ├── risk.json
│   │       ├── readability.json
│   │       ├── reliability.json
│   │       └── resilience.json
│   │
│   ├── orchestrator/   # Coordinación de subagentes
│   │   ├── orchestrator.go
│   │   ├── agent.go
│   │   └── report.go
│   │
│   ├── platform/       # Integración con GitLab/GitHub
│   │   ├── platform.go # interface
│   │   ├── gitlab.go
│   │   └── github.go
│   │
│   ├── memory/         # Sistema de memoria por repo
│   │   ├── memory.go
│   │   └── injector.go
│   │
│   ├── notifier/       # Canales de notificación
│   │   ├── notifier.go
│   │   ├── telegram.go
│   │   ├── slack.go
│   │   └── webhook.go
│   │
│   ├── bot/            # Telegram bot
│   │   ├── bot.go
│   │   └── handlers.go
│   │
│   ├── db/             # Acceso a datos
│   │   ├── db.go
│   │   ├── repos.go
│   │   ├── reviews.go
│   │   └── memory.go
│   │
│   └── config/         # Config del server
│       └── config.go
│
├── migrations/         # SQL migrations
│   ├── 001_init.sql
│   └── 002_memory.sql
│
├── skills/             # Skill files (4R)
│   ├── r-risk.md
│   ├── r-readability.md
│   ├── r-reliability.md
│   └── r-resilience.md
│
├── web/                # Vue 3 frontend
│   ├── src/
│   │   ├── views/
│   │   ├── components/
│   │   ├── composables/
│   │   │   └── useReviewStream.ts  # SSE
│   │   └── stores/     # Pinia
│   └── vite.config.ts
│
├── go.mod
├── go.sum
├── .env.example
└── Makefile
```

---

## 18. Roadmap de implementación

### Fase 1 — Server core (semana 1–2)
- [ ] Scaffold Go + Chi + SQLite + migraciones
- [ ] Model Provider: Claude + OpenAI-compat (Ollama local para dev sin costo)
- [ ] Cargar skills desde filesystem (los 4 .md)
- [ ] Harness básico (timeout + validación de schema)
- [ ] Orchestrator: 4 agentes en paralelo con goroutines
- [ ] `POST /api/v1/reviews` funcional, resultado en DB
- [ ] SSE endpoint funcionando

### Fase 2 — CRUD + UI base (semana 3)
- [ ] REST API completa (repos, channels, skills, providers)
- [ ] `POST /api/v1/repos/infer` (inferencia desde URL)
- [ ] `POST /api/v1/channels/telegram/discover`
- [ ] Vue 3 scaffolding + Pinia + Tailwind
- [ ] Dashboard + listado de repos + wizard de creación
- [ ] Vista de review con progreso SSE en tiempo real

### Fase 3 — Publicación y comentarios inline (semana 4)
- [ ] `review_comments` en DB + generación desde findings
- [ ] Adaptadores GitLab y GitHub para comentarios inline
- [ ] Flujo de aprobación secuencial en UI
- [ ] `POST /api/v1/reviews/:id/publish` con modos all/sequential
- [ ] Webhook GitLab/GitHub recibidos por el server

### Fase 4 — CLI + Telegram bot (semana 5)
- [ ] CLI con Cobra: review run/watch/publish, repo CRUD, provider list
- [ ] Bubbletea para wizards interactivos en terminal
- [ ] Modos local/remote en `~/.config/review-ctl/config.toml`
- [ ] Telegram bot: comandos básicos + botones inline
- [ ] Notificaciones outgoing al finalizar review

### Fase 5 — Memoria + métricas (semana 6)
- [ ] Sistema de memoria SQLite por repo
- [ ] Inyección de contexto en prompts
- [ ] Flujo "marcar como decisión aceptada" (UI + CLI + bot)
- [ ] Dashboard de deuda técnica acumulada (gráficos por dimensión)
- [ ] Integración opcional Engram MCP (flag activable por repo)

---

## Variables de entorno

```bash
# .env.example

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_SECRET=change-me-in-production

# Base de datos
DATABASE_URL=./co_review.db
# DATABASE_URL=postgres://user:pass@localhost:5432/co_review

# API keys (referenciar por nombre en model_configs)
ANTHROPIC_API_KEY=sk-ant-...
OPENAI_API_KEY=sk-...
GOOGLE_API_KEY=...
GROQ_API_KEY=gsk_...
# Ollama no necesita key — usa base_url local

# Plataformas de código
GITLAB_TOKEN=glpat-...
GITHUB_TOKEN=ghp_...

# Telegram
TELEGRAM_BOT_TOKEN=...
TELEGRAM_ALLOWED_USERS=123456789,987654321

# Memoria (opcional)
ENGRAM_MCP_URL=      # si está vacío, usa SQLite local
```

---

*Documento de arquitectura v2. Actualizar en cada decisión de diseño relevante.*
