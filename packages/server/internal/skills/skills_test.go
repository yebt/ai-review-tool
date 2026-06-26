package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadFileValidFrontmatter(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "review-risk.md")
	if err := os.WriteFile(path, []byte(validSkillMarkdown("review-risk", "risk")), 0o600); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	skill, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}
	if skill.Name != "review-risk" || skill.Dimension != "risk" {
		t.Fatalf("unexpected skill metadata: %+v", skill)
	}
	if skill.Harness.TimeoutSeconds != 45 || skill.Harness.MaxRetries != 2 || !skill.Memory.InjectContext {
		t.Fatalf("unexpected nested metadata: %+v", skill.Harness)
	}
}

func TestLoadFileMissingFile(t *testing.T) {
	t.Parallel()
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.md"))
	if err == nil {
		t.Fatal("LoadFile() error = nil, want missing file error")
	}
}

func TestLoadFileInvalidMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	tests := []struct {
		name string
		body string
	}{
		{name: "missing frontmatter", body: "no frontmatter"},
		{name: "bad timeout", body: validSkillMarkdown("review-risk", "risk") + "\n"},
		{name: "wrong schema", body: validSkillMarkdown("review-risk", "readability")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := tt.body
			if tt.name == "bad timeout" {
				body = replace(body, "timeout_seconds: 45", "timeout_seconds: nope")
			}
			path := filepath.Join(dir, tt.name+".md")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("write skill: %v", err)
			}
			_, err := LoadFile(path)
			if err == nil {
				t.Fatal("LoadFile() error = nil, want error")
			}
			if tt.name != "missing frontmatter" && !errors.Is(err, ErrInvalidMetadata) {
				t.Fatalf("LoadFile() error = %v, want ErrInvalidMetadata", err)
			}
		})
	}
}

func validSkillMarkdown(name string, schema string) string {
	return `---
name: ` + name + `
description: Test skill.
model: inherit
readonly: true
background: false
harness:
  timeout_seconds: 45
  max_retries: 2
  output_schema: ` + schema + `
  require_evidence: true
  min_findings_quality: strict
memory:
  inject_context: true
  save_findings: true
---

Body.
`
}

func replace(value string, old string, next string) string {
	return strings.ReplaceAll(value, old, next)
}
