package util_test

import (
	"testing"

	"github.com/reeinharrrd/maestro/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestStripJSONC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain json",
			input: `{"a":1}`,
			want:  `{"a":1}`,
		},
		{
			name:  "strip line comment",
			input: `{"a":1} // comment`,
			want:  `{"a":1} `,
		},
		{
			name:  "strip block comment",
			input: `{"a":1} /* comment */`,
			want:  `{"a":1} `,
		},
		{
			name:  "trailing comma in object",
			input: `{"a":1,}`,
			want:  `{"a":1}`,
		},
		{
			name:  "trailing comma in array",
			input: `[1,2,]`,
			want:  `[1,2]`,
		},
		{
			name:  "trailing comma nested",
			input: `{"a":[1,2,],"b":{"c":3,}}`,
			want:  `{"a":[1,2],"b":{"c":3}}`,
		},
		{
			name:  "comma inside string preserved",
			input: `{"a":","}`,
			want:  `{"a":","}`,
		},
		{
			name:  "string with escaped quote",
			input: `{"a":"\"{\""}`,
			want:  `{"a":"\"{\""}`,
		},
		{
			name:  "block comment inside string preserved",
			input: `{"a":"/* not a comment */"}`,
			want:  `{"a":"/* not a comment */"}`,
		},
		{
			name:  "empty input",
			input: ``,
			want:  ``,
		},
		{
			name:  "line comments in array",
			input: "[\n1,\n// line\n2\n]",
			// Line comment is replaced with blank line (newline preserved)
			want: "[\n1,\n\n2\n]",
		},
		{
			name:  "block comment in array",
			input: "[1,/* comment */2]",
			want:  "[1,2]",
		},
		{
			name:  "multi-line block comment",
			input: "/*\n * multi\n * line\n */\n{\"a\":1}",
			want:  "\n{\"a\":1}",
		},
		{
			name:  "multiple trailing commas",
			input: `{"a":1,"b":[1,2,],}`,
			want:  `{"a":1,"b":[1,2]}`,
		},
		{
			name:  "escaped backslash before quote",
			input: `{"a":"\\"}`,
			want:  `{"a":"\\"}`,
		},
		{
			name:  "single line comment at end with trailing comma",
			input: `{"a":1,}//comment`,
			want:  `{"a":1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.StripJSONC([]byte(tt.input))
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestTrailingCommaRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trailing comma before ]",
			input: `[1,2,]`,
			want:  `[1,2]`,
		},
		{
			name:  "trailing comma before }",
			input: `{"a":1,}`,
			want:  `{"a":1}`,
		},
		{
			name:  "no trailing comma",
			input: `[1,2]`,
			want:  `[1,2]`,
		},
		{
			name:  "whitespace before closing bracket",
			input: `[1,  ]`,
			want:  `[1  ]`,
		},
		{
			name:  "comma inside string (regex is not JSON-aware)",
			input: `{"a": "[1, 2,]"}`,
			want:  `{"a": "[1, 2]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := util.TrailingComma.ReplaceAllString(tt.input, "$1")
			assert.Equal(t, tt.want, got)
		})
	}
}
