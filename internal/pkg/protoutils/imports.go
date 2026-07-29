// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"regexp"
	"strings"

	"github.com/altessa-s/go-atlas/core/collections/slices"
)

// importRe matches a proto import statement on comment-stripped source.
var importRe = regexp.MustCompile(`\bimport\s*(?:public\s+|weak\s+)?"([^"]+)"`)

// ExtractImports returns import paths in declaration order; comments stripped
// first so commented-out imports never match.
func ExtractImports(content string) []string {
	clean := stripComments(content)
	matches := importRe.FindAllStringSubmatch(clean, -1)
	if len(matches) == 0 {
		return nil
	}
	return slices.To(matches, func(m []string) string { return m[1] })
}

// stripComments removes // and /* */ comments, preserving string literals.
// Unterminated constructs consume to EOF.
func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	const (
		modeCode = iota
		modeString
		modeLine
		modeBlock
	)
	mode := modeCode
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch mode {
		case modeCode:
			switch {
			case c == '"':
				mode = modeString
				b.WriteByte(c)
			case c == '/' && i+1 < len(src) && src[i+1] == '/':
				mode = modeLine
				i++
			case c == '/' && i+1 < len(src) && src[i+1] == '*':
				mode = modeBlock
				i++
			default:
				b.WriteByte(c)
			}
		case modeString:
			b.WriteByte(c)
			switch {
			case c == '\\' && i+1 < len(src):
				i++
				b.WriteByte(src[i])
			case c == '"':
				mode = modeCode
			}
		case modeLine:
			if c == '\n' {
				mode = modeCode
				b.WriteByte(c)
			}
		case modeBlock:
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				mode = modeCode
				i++
			}
		}
	}
	return b.String()
}
