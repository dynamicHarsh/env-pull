package parser_test

import (
	"testing"

	"github.com/harsh-sonkar/env-pull/internal/parser"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "simple key-value pairs",
			input: []byte("FOO=bar\nBAZ=qux\n"),
			want:  map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:  "comment lines are ignored",
			input: []byte("# full-line comment\nFOO=bar\n# another comment\n"),
			want:  map[string]string{"FOO": "bar"},
		},
		{
			name:  "inline comment is part of value (not stripped)",
			input: []byte("FOO=bar # not a comment\n"),
			want:  map[string]string{"FOO": "bar # not a comment"},
		},
		{
			name:  "empty lines are ignored",
			input: []byte("\nFOO=bar\n\n\nBAZ=qux\n"),
			want:  map[string]string{"FOO": "bar", "BAZ": "qux"},
		},
		{
			name:  "double-quoted value strips quotes",
			input: []byte(`SECRET="my value with spaces"`),
			want:  map[string]string{"SECRET": "my value with spaces"},
		},
		{
			name:  "single-quoted value strips quotes",
			input: []byte(`TOKEN='abc123'`),
			want:  map[string]string{"TOKEN": "abc123"},
		},
		{
			name:    "mismatched quotes fail closed",
			input:   []byte(`KEY="value'`),
			wantErr: true,
		},
		{
			name:    "export syntax fails closed",
			input:   []byte("export MY_VAR=hello\n"),
			wantErr: true,
		},
		{
			name:  "empty value is preserved",
			input: []byte("EMPTY=\n"),
			want:  map[string]string{"EMPTY": ""},
		},
		{
			name:  "value containing equals sign uses first equals as delimiter",
			input: []byte("URL=https://example.com?a=b&c=d\n"),
			want:  map[string]string{"URL": "https://example.com?a=b&c=d"},
		},
		{
			name:    "line without equals sign fails closed",
			input:   []byte("INVALID_LINE\nGOOD=value\n"),
			wantErr: true,
		},
		{
			name:  "whitespace around key and value is trimmed",
			input: []byte("  KEY  =  value  \n"),
			want:  map[string]string{"KEY": "value"},
		},
		{
			name:  "empty input produces empty map",
			input: []byte{},
			want:  map[string]string{},
		},
		{
			name:  "only comments and blank lines produces empty map",
			input: []byte("# comment\n\n# another\n"),
			want:  map[string]string{},
		},
		{
			name:    "invalid environment key fails closed",
			input:   []byte("NOT-VALID=value\n"),
			wantErr: true,
		},
		{
			name:    "duplicate environment key fails closed",
			input:   []byte("TOKEN=first\nTOKEN=second\n"),
			wantErr: true,
		},
		{
			name:    "shell expansion fails closed",
			input:   []byte("TOKEN=${OTHER}\n"),
			wantErr: true,
		},
		{
			name:    "command substitution fails closed",
			input:   []byte("TOKEN=$(whoami)\n"),
			wantErr: true,
		},
		{
			name:    "malformed line fails closed",
			input:   []byte("TOKEN=value\nthis is malformed\n"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("Parse() returned %d entries, want %d; got = %v", len(got), len(tt.want), got)
			}
			for k, wantV := range tt.want {
				gotV, ok := got[k]
				if !ok {
					t.Errorf("Parse() missing key %q", k)
					continue
				}
				if gotV != wantV {
					t.Errorf("Parse()[%q] = %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}
