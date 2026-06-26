package harness

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"co-review/server/internal/provider"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed schemas/*.json
var schemaFiles embed.FS

const (
	ErrorTimeout       = "TIMEOUT"
	ErrorProvider      = "PROVIDER_ERROR"
	ErrorInvalidJSON   = "INVALID_JSON"
	ErrorInvalidSchema = "INVALID_SCHEMA"
)

type Config struct {
	Dimension    string
	Timeout      time.Duration
	MaxRetries   int
	OutputSchema string
	MaxTokens    int
	RetryBackoff time.Duration
}

type AgentPrompt struct {
	System string
	User   string
}

type TokenMetadata struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

type Result struct {
	Dimension string          `json:"dimension"`
	Attempts  int             `json:"attempts"`
	Duration  time.Duration   `json:"duration"`
	Tokens    TokenMetadata   `json:"tokens"`
	Output    json.RawMessage `json:"output,omitempty"`
	Error     *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Raw     string `json:"raw,omitempty"`
}

// Run executes a prompt through a provider. MaxRetries means retries after the
// initial attempt, so total attempts are 1 + MaxRetries.
func Run(ctx context.Context, cfg Config, p provider.ModelProvider, prompt AgentPrompt) Result {
	start := time.Now()
	result := Result{Dimension: cfg.Dimension}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.OutputSchema == "" {
		cfg.OutputSchema = cfg.Dimension
	}
	totalAttempts := 1 + cfg.MaxRetries
	if totalAttempts < 1 {
		totalAttempts = 1
	}

	var lastErr *Error
	for attempt := 1; attempt <= totalAttempts; attempt++ {
		result.Attempts = attempt
		attemptCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
		resp, err := p.Complete(attemptCtx, provider.CompletionRequest{
			System:    prompt.System,
			User:      prompt.User,
			MaxTokens: cfg.MaxTokens,
		})
		cancel()

		if err != nil {
			lastErr = classifyProviderError(err)
			if !sleepBeforeRetry(ctx, cfg.RetryBackoff, attempt, totalAttempts) {
				break
			}
			continue
		}

		result.Tokens = TokenMetadata{Input: resp.InputTokens, Output: resp.OutputTokens, Total: resp.TokenTotal()}
		validated, validationErr := ValidateOutput([]byte(resp.Content), cfg.OutputSchema)
		if validationErr != nil {
			lastErr = validationErr
			if lastErr.Raw == "" {
				lastErr.Raw = resp.Content
			}
			if !sleepBeforeRetry(ctx, cfg.RetryBackoff, attempt, totalAttempts) {
				break
			}
			continue
		}

		result.Output = validated
		result.Duration = time.Since(start)
		return result
	}

	result.Duration = time.Since(start)
	if lastErr == nil {
		lastErr = &Error{Code: ErrorProvider, Message: "provider returned no result"}
	}
	result.Error = lastErr
	return result
}

func sleepBeforeRetry(ctx context.Context, base time.Duration, attempt int, totalAttempts int) bool {
	if attempt >= totalAttempts {
		return false
	}
	if base <= 0 {
		return true
	}
	wait := base * time.Duration(attempt)
	select {
	case <-time.After(wait):
		return true
	case <-ctx.Done():
		return false
	}
}

func classifyProviderError(err error) *Error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &Error{Code: ErrorTimeout, Message: err.Error()}
	}
	return &Error{Code: ErrorProvider, Message: err.Error()}
}

// ValidateOutput validates a model response against the Phase 2 4R output contract.
func ValidateOutput(raw []byte, schemaName string) (json.RawMessage, *Error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &Error{Code: ErrorInvalidJSON, Message: err.Error(), Raw: string(raw)}
	}
	if err := validateSchema(raw, schemaName); err != nil {
		return nil, &Error{Code: ErrorInvalidSchema, Message: err.Error(), Raw: string(raw)}
	}
	compact, err := json.Marshal(payload)
	if err != nil {
		return nil, &Error{Code: ErrorInvalidJSON, Message: err.Error(), Raw: string(raw)}
	}
	return compact, nil
}

func validateSchema(raw []byte, schemaName string) error {
	schemaPath := fmt.Sprintf("schemas/%s.json", schemaName)
	schemaData, err := schemaFiles.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("unknown output schema %q", schemaName)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaPath, strings.NewReader(string(schemaData))); err != nil {
		return err
	}
	compiled, err := compiler.Compile(schemaPath)
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	return compiled.Validate(value)
}
