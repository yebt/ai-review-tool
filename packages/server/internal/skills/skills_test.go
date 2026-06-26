package skills

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	skillassets "co-review/server/skills"
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
	tests := []struct {
		name string
		body string
	}{
		{name: "missing frontmatter", body: "no frontmatter"},
		{name: "invalid frontmatter boundary", body: replace(validSkillMarkdown("review-risk", "risk"), "---\n\nBody.", "---not-a-boundary\n\nBody.")},
		{name: "bad timeout", body: replace(validSkillMarkdown("review-risk", "risk"), "timeout_seconds: 45", "timeout_seconds: nope")},
		{name: "negative retries", body: replace(validSkillMarkdown("review-risk", "risk"), "max_retries: 2", "max_retries: -1")},
		{name: "invalid require evidence", body: replace(validSkillMarkdown("review-risk", "risk"), "require_evidence: true", "require_evidence: sometimes")},
		{name: "invalid min quality", body: replace(validSkillMarkdown("review-risk", "risk"), "min_findings_quality: strict", "min_findings_quality: medium")},
		{name: "wrong schema", body: validSkillMarkdown("review-risk", "readability")},
		{name: "missing body", body: strings.ReplaceAll(validSkillMarkdown("review-risk", "risk"), "Body.\n", "")},
		{name: "missing required field", body: strings.ReplaceAll(validSkillMarkdown("review-risk", "risk"), "description: Test skill.\n", "")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tt.name+".md")
			if err := os.WriteFile(path, []byte(tt.body), 0o600); err != nil {
				t.Fatalf("write skill: %v", err)
			}
			_, err := LoadFile(path)
			if err == nil {
				t.Fatal("LoadFile() error = nil, want error")
			}
			if strings.Contains(tt.name, "frontmatter") {
				if !errors.Is(err, ErrMissingFrontmatter) {
					t.Fatalf("LoadFile() error = %v, want ErrMissingFrontmatter", err)
				}
				return
			}
			if !errors.Is(err, ErrInvalidMetadata) {
				t.Fatalf("LoadFile() error = %v, want ErrInvalidMetadata", err)
			}
		})
	}
}

func TestLoadFSEmbeddedSkills(t *testing.T) {
	t.Parallel()
	loaded, err := LoadFS(skillassets.FS, ".")
	if err != nil {
		t.Fatalf("LoadFS() error = %v", err)
	}
	if len(loaded) != 4 {
		t.Fatalf("skills count = %d, want 4", len(loaded))
	}

	want := map[string]string{
		"review-readability": "readability",
		"review-reliability": "reliability",
		"review-resilience":  "resilience",
		"review-risk":        "risk",
	}
	for _, skill := range loaded {
		if want[skill.Name] != skill.Dimension {
			t.Fatalf("skill %q dimension = %q, want %q", skill.Name, skill.Dimension, want[skill.Name])
		}
		if skill.Body == "" {
			t.Fatalf("skill %q body is empty", skill.Name)
		}
		delete(want, skill.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing skills: %v", want)
	}
}

func TestLoadFSReturnsFirstInvalidSkillError(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"review-risk.md":        {Data: []byte(validSkillMarkdown("review-risk", "risk"))},
		"review-readability.md": {Data: []byte(replace(validSkillMarkdown("review-readability", "readability"), "output_schema: readability", "output_schema: risk"))},
	}

	_, err := LoadFS(fsys, ".")
	if !errors.Is(err, ErrInvalidMetadata) {
		t.Fatalf("LoadFS() error = %v, want ErrInvalidMetadata", err)
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
