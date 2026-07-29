// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractImports(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "plain import",
			content: `syntax = "proto3";` + "\n" + `import "common/types.proto";`,
			want:    []string{"common/types.proto"},
		},
		{
			name:    "import public",
			content: `import public "a/b.proto";`,
			want:    []string{"a/b.proto"},
		},
		{
			name:    "import weak",
			content: `import weak "a/b.proto";`,
			want:    []string{"a/b.proto"},
		},
		{
			name:    "no space before quote",
			content: `import"a.proto";`,
			want:    []string{"a.proto"},
		},
		{
			name:    "two statements on one line",
			content: `import "a.proto"; import "b.proto";`,
			want:    []string{"a.proto", "b.proto"},
		},
		{
			name:    "trailing line comment with quotes does not pollute",
			content: `import "a.proto"; // see "common.proto" for details`,
			want:    []string{"a.proto"},
		},
		{
			name:    "commented-out line import ignored",
			content: "// import \"dead.proto\";\nimport \"live.proto\";",
			want:    []string{"live.proto"},
		},
		{
			name:    "block comment ignored",
			content: "/*\nimport \"dead.proto\";\n*/\nimport \"live.proto\";",
			want:    []string{"live.proto"},
		},
		{
			name:    "inline block comment between keyword and path",
			content: `import /* hmm */ "a.proto";`,
			want:    []string{"a.proto"},
		},
		{
			name:    "crlf endings",
			content: "import \"a.proto\";\r\nimport \"b.proto\";\r\n",
			want:    []string{"a.proto", "b.proto"},
		},
		{
			name:    "import keyword inside option string is not an import",
			content: `option (foo) = "import \"x.proto\"";`,
			want:    nil,
		},
		{
			name:    "empty file",
			content: "",
			want:    nil,
		},
		{
			name:    "wkt import",
			content: `import "google/protobuf/timestamp.proto";`,
			want:    []string{"google/protobuf/timestamp.proto"},
		},
		{
			name:    "declaration order preserved",
			content: "import \"z.proto\";\nimport \"a.proto\";",
			want:    []string{"z.proto", "a.proto"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractImports(tt.content))
		})
	}
}

func TestStripComments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"line comment removed", "code // comment\nnext", "code \nnext"},
		{"block comment removed", "a /* b */ c", "a  c"},
		{"string with slashes preserved", `s = "http://x";`, `s = "http://x";`},
		{"escaped quote in string", `s = "a\"b"; // tail`, `s = "a\"b"; `},
		{"unterminated block comment", "a /* b", "a "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripComments(tt.in))
		})
	}
}
