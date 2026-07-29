// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func pf(p, content string) entities.ProtoFileEntry {
	return entities.ProtoFileEntry{Path: p, Content: content, Size: int64(len(content))}
}

// protoFile builds a minimal proto file content with the given imports.
// Named protoFile, not proto, to avoid a name clash with the proto package.
func protoFile(imports ...string) string {
	s := "syntax = \"proto3\";\n"
	for _, i := range imports {
		s += "import \"" + i + "\";\n"
	}
	return s
}

func errorDiags(diags []entities.CompileDiagnostic) []entities.CompileDiagnostic {
	var out []entities.CompileDiagnostic
	for _, d := range diags {
		if d.Severity == entities.DiagnosticError {
			out = append(out, d)
		}
	}
	return out
}

func TestInferRoots(t *testing.T) {
	t.Run("flat tree imports relative to source root -> no extra roots", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("common/types.proto", protoFile()),
			pf("api/service.proto", protoFile("common/types.proto")),
		}
		roots, diags := inferRoots(files)
		assert.Empty(t, roots)
		assert.Empty(t, errorDiags(diags))
	})

	t.Run("single nested root discovered", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("proto/common/types.proto", protoFile()),
			pf("proto/api/service.proto", protoFile("common/types.proto")),
		}
		roots, _ := inferRoots(files)
		assert.Equal(t, []string{"proto"}, roots)
	})

	t.Run("third_party root discovered without hardcode", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("third_party/buf/validate/validate.proto", protoFile()),
			pf("api/service.proto", protoFile("buf/validate/validate.proto")),
		}
		roots, _ := inferRoots(files)
		assert.Equal(t, []string{"third_party"}, roots)
	})

	t.Run("multiple roots discovered", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("apis/order/v1/order.proto", protoFile("acme/money.proto")),
			pf("vendor/acme/money.proto", protoFile()),
			pf("apis/user/v1/user.proto", protoFile("order/v1/order.proto")),
		}
		roots, _ := inferRoots(files)
		assert.ElementsMatch(t, []string{"vendor", "apis"}, roots)
	})

	t.Run("vendored basename does not poison resolution (audit 1.1)", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("myapi/common.proto", protoFile()),
			pf("third_party/x/common.proto", protoFile()),
			pf("myapi/svc.proto", protoFile("myapi/common.proto")),
		}
		roots, diags := inferRoots(files)
		assert.Empty(t, roots) // full-path import -> root ""
		assert.Empty(t, errorDiags(diags))
	})

	t.Run("dirs starting with pb are ordinary dirs (audit 1.2)", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("src/pbevents/events.proto", protoFile()),
			pf("src/svc.proto", protoFile("pbevents/events.proto")),
		}
		roots, _ := inferRoots(files)
		assert.Equal(t, []string{"src"}, roots)
	})

	t.Run("same basename different dirs resolved by suffix", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("a/common.proto", protoFile()),
			pf("b/common.proto", protoFile()),
			pf("svc.proto", protoFile("a/common.proto", "b/common.proto")),
		}
		roots, diags := inferRoots(files)
		assert.Empty(t, roots)
		assert.Empty(t, errorDiags(diags))
	})

	t.Run("ambiguous suffix match -> longest path wins, warning emitted", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("x/types.proto", "content A"),
			pf("deep/nested/types.proto", "content B"),
			pf("svc.proto", protoFile("types.proto")),
		}
		roots, diags := inferRoots(files)
		assert.Equal(t, []string{"deep/nested"}, roots)
		var warned bool
		for _, d := range diags {
			if d.Severity == entities.DiagnosticWarning {
				warned = true
			}
		}
		assert.True(t, warned, "ambiguity must produce a warning")
	})

	t.Run("ambiguous match with identical content -> no warning", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("x/types.proto", "same"),
			pf("y/types.proto", "same"),
			pf("svc.proto", protoFile("types.proto")),
		}
		_, diags := inferRoots(files)
		for _, d := range diags {
			assert.NotEqual(t, entities.DiagnosticWarning, d.Severity)
		}
	})

	t.Run("wkt imports do not drive inference", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("svc.proto", protoFile("google/protobuf/timestamp.proto")),
		}
		roots, diags := inferRoots(files)
		assert.Empty(t, roots)
		assert.Empty(t, diags)
	})

	t.Run("deterministic across input order", func(t *testing.T) {
		a := []entities.ProtoFileEntry{
			pf("p/x/a.proto", protoFile()),
			pf("p/y/b.proto", protoFile("x/a.proto")),
		}
		b := []entities.ProtoFileEntry{a[1], a[0]}
		r1, _ := inferRoots(a)
		r2, _ := inferRoots(b)
		assert.Equal(t, r1, r2)
	})

	t.Run("five levels deep", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("a/b/c/d/e/types.proto", protoFile()),
			pf("a/b/svc.proto", protoFile("c/d/e/types.proto")),
		}
		roots, _ := inferRoots(files)
		assert.Equal(t, []string{"a/b"}, roots)
	})
}

func TestApplyRoots(t *testing.T) {
	t.Run("strips longest matching root", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("proto/a.proto", "A"),
			pf("proto/vendor/b.proto", "B"),
		}
		srcs, targets, diags := applyRoots(files, []string{"proto", "proto/vendor"}, true)
		assert.Equal(t, "A", srcs["a.proto"])
		assert.Equal(t, "B", srcs["b.proto"])
		assert.Equal(t, []string{"a.proto", "b.proto"}, targets)
		assert.Empty(t, diags)
	})

	t.Run("file outside roots kept under full path when fallback on", func(t *testing.T) {
		files := []entities.ProtoFileEntry{pf("tools/x.proto", "X")}
		srcs, targets, _ := applyRoots(files, []string{"proto"}, true)
		assert.Equal(t, "X", srcs["tools/x.proto"])
		assert.Equal(t, []string{"tools/x.proto"}, targets)
	})

	t.Run("file outside roots skipped with info when fallback off (buf mode)", func(t *testing.T) {
		files := []entities.ProtoFileEntry{pf("tools/x.proto", "X"), pf("proto/a.proto", "A")}
		srcs, targets, diags := applyRoots(files, []string{"proto"}, false)
		assert.NotContains(t, srcs, "tools/x.proto")
		assert.Equal(t, []string{"a.proto"}, targets)
		require.Len(t, diags, 1)
		assert.Equal(t, entities.DiagnosticInfo, diags[0].Severity)
	})

	t.Run("key collision identical content deduplicated with info", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("a/x.proto", "same"),
			pf("b/x.proto", "same"),
		}
		srcs, targets, diags := applyRoots(files, []string{"a", "b"}, true)
		assert.Len(t, srcs, 1)
		assert.Equal(t, []string{"x.proto"}, targets)
		require.Len(t, diags, 1)
		assert.Equal(t, entities.DiagnosticInfo, diags[0].Severity)
	})

	t.Run("key collision different content is an error", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("a/x.proto", "AAA"),
			pf("b/x.proto", "BBB"),
		}
		_, _, diags := applyRoots(files, []string{"a", "b"}, true)
		errs := errorDiags(diags)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Message, "collision")
		assert.NotEmpty(t, errs[0].Hint)
	})

	t.Run("vendored wkt keyed canonically", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("third_party/google/protobuf/timestamp.proto", "vendored"),
		}
		srcs, _, _ := applyRoots(files, []string{"third_party"}, true)
		assert.Equal(t, "vendored", srcs["google/protobuf/timestamp.proto"])
	})
}

func TestResolveLayout(t *testing.T) {
	t.Run("manual roots win over everything", func(t *testing.T) {
		files := []entities.ProtoFileEntry{pf("custom/a.proto", protoFile())}
		configs := []entities.ProtoFileEntry{cfg("buf.yaml", "version: v2\nmodules:\n  - path: other\n")}
		l := ResolveLayout(files, []string{"custom/"}, configs)
		assert.Equal(t, RootsOriginManual, l.Origin)
		assert.Equal(t, []string{"custom"}, l.Roots)
		assert.Contains(t, l.Srcs, "a.proto")
	})

	t.Run("buf config wins over inference", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("proto/a.proto", protoFile()),
			pf("stray.proto", protoFile()),
		}
		configs := []entities.ProtoFileEntry{cfg("buf.yaml", "version: v2\nmodules:\n  - path: proto\n")}
		l := ResolveLayout(files, nil, configs)
		assert.Equal(t, RootsOriginBuf, l.Origin)
		assert.Equal(t, []string{"proto"}, l.Roots)
		assert.NotContains(t, l.Srcs, "stray.proto") // outside modules -> skipped
	})

	t.Run("no configs -> inference", func(t *testing.T) {
		files := []entities.ProtoFileEntry{
			pf("proto/common/t.proto", protoFile()),
			pf("proto/api/s.proto", protoFile("common/t.proto")),
		}
		l := ResolveLayout(files, nil, nil)
		assert.Equal(t, RootsOriginInferred, l.Origin)
		assert.Equal(t, []string{"proto"}, l.Roots)
		assert.Contains(t, l.Srcs, "common/t.proto")
		assert.Contains(t, l.Srcs, "api/s.proto")
	})

	t.Run("buf config with empty roots means source root module", func(t *testing.T) {
		files := []entities.ProtoFileEntry{pf("a/b.proto", protoFile())}
		configs := []entities.ProtoFileEntry{cfg("buf.yaml", "version: v1\n")}
		l := ResolveLayout(files, nil, configs)
		assert.Equal(t, RootsOriginBuf, l.Origin)
		assert.Contains(t, l.Srcs, "a/b.proto") // root "" -> key = full path
	})

	t.Run("locks propagated", func(t *testing.T) {
		files := []entities.ProtoFileEntry{pf("a.proto", protoFile())}
		configs := []entities.ProtoFileEntry{cfg("buf.lock", "version: v2\n")}
		l := ResolveLayout(files, nil, configs)
		require.Len(t, l.Locks, 1)
	})

	t.Run("empty input produces error diag", func(t *testing.T) {
		l := ResolveLayout(nil, nil, nil)
		require.NotEmpty(t, errorDiags(l.Diags))
		assert.Empty(t, l.Targets)
	})
}
