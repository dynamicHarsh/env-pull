// Package parser parses .env-format byte slices into key-value maps.
// It is kept separate from crypto so it can be tested without any key material.
package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

var environmentKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Parse converts a .env-format byte slice into a map[string]string.
// Empty lines and full-line comments are ignored. Every other line must be a
// unique, valid environment-variable assignment with a literal value.
//
// The caller is responsible for zeroing the input slice after Parse returns
// to limit the lifetime of plaintext in memory.
func Parse(data []byte) (map[string]string, error) {
	out := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			return nil, fmt.Errorf("parser: line %d is not an assignment", lineNumber)
		}

		key := strings.TrimSpace(line[:idx])
		if !environmentKey.MatchString(key) {
			return nil, fmt.Errorf("parser: line %d has invalid environment key %q", lineNumber, key)
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("parser: line %d duplicates environment key %q", lineNumber, key)
		}

		val := strings.TrimSpace(line[idx+1:])
		if strings.Contains(val, "$") || strings.Contains(val, "`") {
			return nil, fmt.Errorf("parser: line %d contains shell evaluation syntax", lineNumber)
		}
		if hasUnmatchedQuote(val) {
			return nil, fmt.Errorf("parser: line %d has unmatched quotes", lineNumber)
		}
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

func hasUnmatchedQuote(s string) bool {
	if len(s) < 1 {
		return false
	}
	first, last := s[0], s[len(s)-1]
	if first == '"' || first == '\'' {
		return first != last
	}
	return last == '"' || last == '\''
}
