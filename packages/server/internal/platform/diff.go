package platform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderPattern = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// MapDiffPositions maps added and changed lines from a unified diff to
// provider-neutral inline comment positions.
func MapDiffPositions(diff string, file ChangedFile, baseSHA string, startSHA string, headSHA string) ([]DiffPosition, error) {
	var positions []DiffPosition
	oldLine := 0
	newLine := 0
	pendingDeletes := 0
	inHunk := false

	for _, line := range strings.Split(diff, "\n") {
		if line == "" && !inHunk {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			oldStart, newStart, err := parseHunkHeader(line)
			if err != nil {
				return nil, err
			}
			oldLine = oldStart
			newLine = newStart
			pendingDeletes = 0
			inHunk = true
			continue
		}
		if !inHunk || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "-"):
			oldLine++
			pendingDeletes++
		case strings.HasPrefix(line, "+"):
			kind := "added"
			if pendingDeletes > 0 {
				kind = "changed"
				pendingDeletes--
			}
			positions = append(positions, DiffPosition{
				BaseSHA:      baseSHA,
				StartSHA:     startSHA,
				HeadSHA:      headSHA,
				OldPath:      file.OldPath,
				NewPath:      file.NewPath,
				NewLine:      newLine,
				PositionType: "text",
				Kind:         kind,
			})
			newLine++
		case strings.HasPrefix(line, " "):
			oldLine++
			newLine++
			pendingDeletes = 0
		case strings.HasPrefix(line, `\ No newline at end of file`):
			continue
		default:
			pendingDeletes = 0
		}
	}

	return positions, nil
}

func parseHunkHeader(line string) (int, int, error) {
	matches := hunkHeaderPattern.FindStringSubmatch(line)
	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("malformed diff hunk header %q", line)
	}
	oldStart, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}
	newStart, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}
	return oldStart, newStart, nil
}
