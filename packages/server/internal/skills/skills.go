package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrMissingFrontmatter = errors.New("skill frontmatter missing")
	ErrInvalidMetadata    = errors.New("skill metadata invalid")
)

type HarnessConfig struct {
	TimeoutSeconds     int
	MaxRetries         int
	OutputSchema       string
	RequireEvidence    bool
	MinFindingsQuality string
}

type MemoryConfig struct {
	InjectContext bool
	SaveFindings  bool
}

type Skill struct {
	Name        string
	Description string
	Dimension   string
	Model       string
	Readonly    bool
	Background  bool
	Harness     HarnessConfig
	Memory      MemoryConfig
	Body        string
	FilePath    string
}

func LoadDir(dir string) ([]Skill, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(paths)

	skills := make([]Skill, 0, len(paths))
	for _, path := range paths {
		skill, err := LoadFile(path)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func LoadFS(fsys fs.FS, dir string) ([]Skill, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(dir, entry.Name())))
	}
	sort.Strings(paths)

	skills := make([]Skill, 0, len(paths))
	for _, path := range paths {
		skill, err := LoadFSFile(fsys, path)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func LoadFile(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Skill{}, err
	}
	return parseSkill(data, path)
}

func LoadFSFile(fsys fs.FS, path string) (Skill, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return Skill{}, err
	}
	return parseSkill(data, path)
}

func parseSkill(data []byte, path string) (Skill, error) {
	frontmatter, body, err := splitFrontmatter(string(data))
	if err != nil {
		return Skill{}, err
	}
	skill, err := parseFrontmatter(frontmatter)
	if err != nil {
		return Skill{}, err
	}
	skill.Body = strings.TrimSpace(body)
	skill.FilePath = path
	if err := validate(&skill); err != nil {
		return Skill{}, err
	}
	return skill, nil
}

func splitFrontmatter(content string) (string, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", ErrMissingFrontmatter
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", ErrMissingFrontmatter
	}
	frontmatter := rest[:idx]
	body := strings.TrimPrefix(rest[idx:], "\n---")
	body = strings.TrimPrefix(body, "\n")
	return frontmatter, body, nil
}

func parseFrontmatter(frontmatter string) (Skill, error) {
	var skill Skill
	section := ""
	for _, raw := range strings.Split(frontmatter, "\n") {
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		line := strings.TrimSpace(raw)
		if strings.HasSuffix(line, ":") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return Skill{}, fmt.Errorf("%w: malformed line %q", ErrInvalidMetadata, line)
		}
		key = strings.TrimSpace(key)
		value = cleanValue(value)
		if indent == 0 {
			section = ""
		}
		switch section {
		case "harness":
			if err := assignHarness(&skill.Harness, key, value); err != nil {
				return Skill{}, err
			}
		case "memory":
			if err := assignMemory(&skill.Memory, key, value); err != nil {
				return Skill{}, err
			}
		default:
			if err := assignTopLevel(&skill, key, value); err != nil {
				return Skill{}, err
			}
		}
	}
	return skill, nil
}

func cleanValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	if idx := strings.Index(value, " #"); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}

func assignTopLevel(skill *Skill, key string, value string) error {
	switch key {
	case "name":
		skill.Name = value
	case "description":
		skill.Description = value
	case "model":
		skill.Model = value
	case "readonly":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%w: readonly must be boolean", ErrInvalidMetadata)
		}
		skill.Readonly = parsed
	case "background":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%w: background must be boolean", ErrInvalidMetadata)
		}
		skill.Background = parsed
	}
	return nil
}

func assignHarness(cfg *HarnessConfig, key string, value string) error {
	switch key {
	case "timeout_seconds":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			return fmt.Errorf("%w: harness.timeout_seconds must be positive integer", ErrInvalidMetadata)
		}
		cfg.TimeoutSeconds = parsed
	case "max_retries":
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return fmt.Errorf("%w: harness.max_retries must be non-negative integer", ErrInvalidMetadata)
		}
		cfg.MaxRetries = parsed
	case "output_schema":
		cfg.OutputSchema = value
	case "require_evidence":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("%w: harness.require_evidence must be boolean", ErrInvalidMetadata)
		}
		cfg.RequireEvidence = parsed
	case "min_findings_quality":
		cfg.MinFindingsQuality = value
	}
	return nil
}

func assignMemory(cfg *MemoryConfig, key string, value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("%w: memory.%s must be boolean", ErrInvalidMetadata, key)
	}
	switch key {
	case "inject_context":
		cfg.InjectContext = parsed
	case "save_findings":
		cfg.SaveFindings = parsed
	}
	return nil
}

func validate(skill *Skill) error {
	if skill.Name == "" || skill.Description == "" || skill.Model == "" {
		return fmt.Errorf("%w: name, description, and model are required", ErrInvalidMetadata)
	}
	dimension, ok := dimensionByName(skill.Name)
	if !ok {
		return fmt.Errorf("%w: unsupported skill name %q", ErrInvalidMetadata, skill.Name)
	}
	if skill.Harness.TimeoutSeconds <= 0 || skill.Harness.OutputSchema == "" {
		return fmt.Errorf("%w: harness timeout and output_schema are required", ErrInvalidMetadata)
	}
	if skill.Harness.OutputSchema != dimension {
		return fmt.Errorf("%w: output_schema %q does not match dimension %q", ErrInvalidMetadata, skill.Harness.OutputSchema, dimension)
	}
	if skill.Harness.MinFindingsQuality != "" && skill.Harness.MinFindingsQuality != "strict" && skill.Harness.MinFindingsQuality != "lenient" {
		return fmt.Errorf("%w: min_findings_quality must be strict or lenient", ErrInvalidMetadata)
	}
	skill.Dimension = dimension
	return nil
}

func dimensionByName(name string) (string, bool) {
	switch name {
	case "review-risk":
		return "risk", true
	case "review-readability":
		return "readability", true
	case "review-reliability":
		return "reliability", true
	case "review-resilience":
		return "resilience", true
	default:
		return "", false
	}
}
