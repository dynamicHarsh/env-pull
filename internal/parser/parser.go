// Package parser parses .env-format byte slices into key-value maps.
// It is kept separate from crypto so it can be tested without any key material.
package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

// Parse converts a .env-format byte slice into a map[string]string.
// Rules:
//   - Empty lines and lines whose first non-space character is '#' are ignored.
//   - An optional "export " prefix is stripped before splitting (bash compatibility).
//   - Each line is split on the first '=' only; values may contain '='.
//   - Surrounding whitespace is trimmed from keys and values.
//   - Values surrounded by matching single or double quotes have the quotes stripped.
//   - Lines with no '=' or an empty key are silently skipped.
//
// The caller is responsible for zeroing the input slice after Parse returns
// to limit the lifetime of plaintext in memory.
func Parse(data []byte) (map[string]string, error) {
	out := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip bash "export " prefix so exported env files work unmodified.
		line = strings.TrimPrefix(line, "export ")

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue // no '=': not a valid key-value line
		}

		key := strings.TrimSpace(line[:idx])
		if key == "" {
			continue
		}

		val := strings.TrimSpace(line[idx+1:])
		val = stripSurroundingQuotes(val)

		out[key] = val
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parser: scan error: %w", err)
	}
	return out, nil
}

// stripSurroundingQuotes removes a single pair of matching quotes (" or ') from
// the start and end of s. It does not recurse or unescape inner characters.
func stripSurroundingQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
