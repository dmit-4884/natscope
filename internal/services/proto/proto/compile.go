// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/reporter"

	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// compileOutput is everything one compile pass produced.
type compileOutput struct {
	FDS    []protoreflect.FileDescriptor
	Diags  []entities.CompileDiagnostic
	Roots  []string
	Origin protoutils.RootsOrigin
}

// HasErrors reports whether any diagnostic is error-severity.
func (o *compileOutput) HasErrors() bool {
	for _, d := range o.Diags {
		if d.Severity == entities.DiagnosticError {
			return true
		}
	}
	return false
}

// compile resolves the import layout and runs protocompile; error
// diagnostics mean no descriptors, err is reserved for non-compile failures.
func (s *Service) compile(
	ctx context.Context,
	src *entities.ProtoSource,
	files []entities.ProtoFileEntry,
	configs []entities.ProtoFileEntry,
	diskRoot string,
) (*compileOutput, error) {
	files, excludedCount := filterExcluded(files, protoutils.NormalizeRoots(src.ExcludePrefixes))

	layout := protoutils.ResolveLayout(files, src.ImportRoots, configs)
	out := &compileOutput{Diags: layout.Diags, Roots: layout.Roots, Origin: layout.Origin}
	if excludedCount > 0 {
		out.Diags = append(out.Diags, entities.CompileDiagnostic{
			Severity: entities.DiagnosticInfo,
			Message:  fmt.Sprintf("%d file(s) excluded by ExcludePrefixes %v", excludedCount, src.ExcludePrefixes),
		})
	}
	if len(layout.Targets) == 0 && len(files)+excludedCount > 0 {
		out.Diags = append(out.Diags, entities.CompileDiagnostic{
			Severity: entities.DiagnosticError,
			Message:  "no compile targets after resolution",
			Hint:     "all files excluded (ExcludePrefixes) or outside buf modules — check the source configuration",
		})
	}
	if out.HasErrors() {
		return out, nil
	}

	var bufDeps []string
	for _, lock := range layout.Locks {
		bufDeps = append(bufDeps, protoutils.ResolveBufDepsFromLock(lock)...)
	}
	if len(bufDeps) == 0 && diskRoot != "" {
		bufDeps = protoutils.ResolveBufDeps(diskRoot)
	}

	memResolver := &protocompile.SourceResolver{
		Accessor: protocompile.SourceAccessorFromMap(layout.Srcs),
	}
	diskResolver := &protocompile.SourceResolver{ImportPaths: bufDeps}
	resolver := protocompile.WithStandardImports(protocompile.CompositeResolver{
		memResolver,
		diskResolver,
	})

	rep := &collectingReporter{}
	compiler := protocompile.Compiler{
		Resolver:       resolver,
		SourceInfoMode: protocompile.SourceInfoNone,
		Reporter:       rep,
	}

	s.logger.Debug("[proto-compile] resolved layout",
		slog.String("origin", string(layout.Origin)),
		slog.Any("roots", layout.Roots),
		slog.Int("targets", len(layout.Targets)),
		slog.Int("buf_dep_paths", len(bufDeps)))

	compiled, compileErr := compiler.Compile(ctx, layout.Targets...)

	for _, e := range rep.errs {
		out.Diags = append(out.Diags, s.enrichHint(errorWithPosToDiagnostic(e, entities.DiagnosticError, nil), layout))
	}
	for _, w := range rep.warnings {
		out.Diags = append(out.Diags, errorWithPosToDiagnostic(w, entities.DiagnosticWarning, nil))
	}
	if compileErr != nil && !errors.Is(compileErr, reporter.ErrInvalidSource) {
		switch ewp, ok := errors.AsType[reporter.ErrorWithPos](compileErr); {
		case ok:
			out.Diags = append(out.Diags, s.enrichHint(errorWithPosToDiagnostic(ewp, entities.DiagnosticError, nil), layout))
		case len(out.Diags) == 0:
			return nil, compileErr
		default:
			out.Diags = append(out.Diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError, Message: compileErr.Error(),
			})
		}
	}

	if out.HasErrors() {
		return out, nil
	}
	out.FDS = slices.To(compiled, func(f linker.File) protoreflect.FileDescriptor { return f })
	return out, nil
}

// enrichHint swaps in the auto-walk-aware hint for missing-import diagnostics.
func (s *Service) enrichHint(
	d entities.CompileDiagnostic,
	layout *protoutils.ResolvedLayout,
) entities.CompileDiagnostic {
	if d.MissingImport != "" {
		d.Hint = autoWalkHint(d.MissingImport, layout)
	}
	return d
}

// filterExcluded drops files under any excluded prefix ("gen" excludes "gen"
// and "gen/...").
func filterExcluded(files []entities.ProtoFileEntry, prefixes []string) ([]entities.ProtoFileEntry, int) {
	if len(prefixes) == 0 {
		return files, 0
	}
	kept := files[:0:0]
	excluded := 0
	for _, f := range files {
		drop := slices.Any(prefixes, func(p string) bool {
			return f.Path == p || strings.HasPrefix(f.Path, p+"/")
		})
		if drop {
			excluded++
			continue
		}
		kept = append(kept, f)
	}
	return kept, excluded
}

// serialize serializes file descriptors to FileDescriptorSet bytes.
func (s *Service) serialize(fds []protoreflect.FileDescriptor) ([]byte, error) {
	descSet := &descriptorpb.FileDescriptorSet{}

	// Collect all file descriptors including dependencies.
	seen := make(map[string]bool)
	var addFile func(fd protoreflect.FileDescriptor)
	addFile = func(fd protoreflect.FileDescriptor) {
		name := fd.Path()
		if seen[name] {
			return
		}
		seen[name] = true

		// Add dependencies first.
		imports := fd.Imports()
		for i := range imports.Len() {
			addFile(imports.Get(i).FileDescriptor)
		}

		// Convert to FileDescriptorProto.
		fdp := protodesc.ToFileDescriptorProto(fd)
		fdp.SourceCodeInfo = nil // Strip to save ~90 % memory.

		descSet.File = append(descSet.File, fdp)
	}

	for _, fd := range fds {
		addFile(fd)
	}

	data, err := proto.Marshal(descSet)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "marshal descriptor set")
	}

	return data, nil
}

// extractTypes extracts all message type full names from file descriptors.
func (s *Service) extractTypes(fds []protoreflect.FileDescriptor) []string {
	var types []string
	seen := make(map[string]bool)

	var addMessage func(md protoreflect.MessageDescriptor)
	addMessage = func(md protoreflect.MessageDescriptor) {
		fullName := string(md.FullName())
		if seen[fullName] {
			return
		}
		seen[fullName] = true
		types = append(types, fullName)

		// Add nested messages.
		nested := md.Messages()
		for i := range nested.Len() {
			addMessage(nested.Get(i))
		}
	}

	for _, fd := range fds {
		msgs := fd.Messages()
		for i := range msgs.Len() {
			addMessage(msgs.Get(i))
		}
	}

	return types
}
