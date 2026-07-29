// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// ParseDescriptorSet parses FileDescriptorSet bytes into a map of message
// descriptors keyed by fully-qualified name.
func ParseDescriptorSet(data []byte) (map[string]protoreflect.MessageDescriptor, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty descriptor set")
	}

	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fds); err != nil {
		return nil, errors.WrapOperation(err, "unmarshal descriptor set")
	}

	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, errors.WrapOperation(err, "create file registry")
	}

	result := make(map[string]protoreflect.MessageDescriptor)
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		msgs := fd.Messages()
		for i := range msgs.Len() {
			collectMessages(result, msgs.Get(i))
		}
		return true
	})

	return result, nil
}

// SchemaInput is one descriptor set fed to MergeWithReport, tagged with its
// source snapshot so conflict reports can name winners and losers.
type SchemaInput struct {
	SourceID string
	Tag      string
	Bytes    []byte
}

// MergeReport carries the merged result and any conflicts detected during merge.
type MergeReport struct {
	Result    []byte
	Conflicts []*entities.SchemaConflict
}

// symbolOwner records who owns an FQN symbol and its source file. Used to
// detect cross-file conflicts.
type symbolOwner struct {
	ref       entities.SchemaRef
	fileName  string
	fileBytes []byte // marshaled FileDescriptorProto containing this symbol
}

// MergeWithReport merges schema inputs into one FileDescriptorSet, recording
// each dropped file/symbol as a SchemaConflict. First-wins.
func MergeWithReport(inputs []SchemaInput) (*MergeReport, error) {
	merged := &descriptorpb.FileDescriptorSet{}
	report := &MergeReport{}

	type seenFile struct {
		ref     entities.SchemaRef
		content []byte
	}
	seen := make(map[string]seenFile)

	typeOwner := make(map[string]symbolOwner)

	for _, in := range inputs {
		if len(in.Bytes) == 0 {
			continue
		}
		fds := &descriptorpb.FileDescriptorSet{}
		if err := proto.Unmarshal(in.Bytes, fds); err != nil {
			continue
		}

		for _, file := range fds.File {
			name := file.GetName()
			fileBytes, _ := proto.Marshal(file) //nolint:errcheck // deterministic marshal; error unreachable

			// === File-level conflict ===
			if existing, ok := seen[name]; ok {
				if bytes.Equal(existing.content, fileBytes) {
					continue // identical, harmless duplicate
				}
				if strings.HasPrefix(name, "google/") {
					continue // well-known types, skip silently
				}
				report.Conflicts = append(report.Conflicts, entities.SchemaConflictNew(func(c *entities.SchemaConflict) {
					c.Kind = entities.DuplicateFileDifferentContent
					c.Severity = entities.SeverityError
					c.Symbol = name
					c.Winner = existing.ref
					c.Loser = entities.SchemaRef{SourceID: in.SourceID, Tag: in.Tag, File: name}
					c.Reason = fmt.Sprintf("file %q has different content in two snapshots", name)
					c.Policy = "first-wins; reject by strict policy"
				}))
				continue
			}

			// === Symbol-level conflict ===
			conflictedSymbols := scanFileForExistingSymbols(file, typeOwner)
			if len(conflictedSymbols) > 0 {
				first := conflictedSymbols[0]
				prev := typeOwner[first]

				kind := entities.SameSymbolDifferentShape
				severity := entities.SeverityError
				if bytes.Equal(prev.fileBytes, fileBytes) {
					kind = entities.SameSymbolSameShape
					severity = entities.SeverityInfo
				}

				report.Conflicts = append(report.Conflicts, entities.SchemaConflictNew(func(c *entities.SchemaConflict) {
					c.Kind = kind
					c.Severity = severity
					c.Symbol = first
					c.Winner = prev.ref
					c.Loser = entities.SchemaRef{SourceID: in.SourceID, Tag: in.Tag, File: name}
					if kind == entities.SameSymbolSameShape {
						c.Reason = fmt.Sprintf("symbol %q duplicated with identical shape", first)
						c.Policy = "first-wins; allowed (info only)"
					} else {
						c.Reason = fmt.Sprintf("symbol %q has different shape in another snapshot", first)
						c.Policy = "first-wins; reject by strict policy"
					}
				}))
				continue
			}

			ref := entities.SchemaRef{SourceID: in.SourceID, Tag: in.Tag, File: name}
			registerFileSymbols(file, typeOwner, ref, fileBytes)

			seen[name] = seenFile{ref: ref, content: fileBytes}
			merged.File = append(merged.File, file)
		}
	}

	out, err := proto.Marshal(merged)
	if err != nil {
		return nil, err
	}
	report.Result = out
	return report, nil
}

// scanFileForExistingSymbols returns the FQNs declared in `file` that are
// already registered in `typeOwner` from a previous file.
func scanFileForExistingSymbols(file *descriptorpb.FileDescriptorProto, typeOwner map[string]symbolOwner) []string {
	pkg := file.GetPackage()
	out := []string{}
	check := func(fqn string) {
		if owner, exists := typeOwner[fqn]; exists && owner.fileName != file.GetName() {
			out = append(out, fqn)
		}
	}
	for _, msg := range file.GetMessageType() {
		check(qualifiedName(pkg, msg.GetName()))
	}
	for _, enum := range file.GetEnumType() {
		check(qualifiedName(pkg, enum.GetName()))
	}
	for _, svc := range file.GetService() {
		check(qualifiedName(pkg, svc.GetName()))
	}
	return out
}

// registerFileSymbols records ownership of every top-level FQN in the file.
func registerFileSymbols(
	file *descriptorpb.FileDescriptorProto,
	typeOwner map[string]symbolOwner,
	ref entities.SchemaRef,
	fileBytes []byte,
) {
	pkg := file.GetPackage()
	owner := symbolOwner{ref: ref, fileName: file.GetName(), fileBytes: fileBytes}
	for _, msg := range file.GetMessageType() {
		typeOwner[qualifiedName(pkg, msg.GetName())] = owner
	}
	for _, enum := range file.GetEnumType() {
		typeOwner[qualifiedName(pkg, enum.GetName())] = owner
	}
	for _, svc := range file.GetService() {
		typeOwner[qualifiedName(pkg, svc.GetName())] = owner
	}
}

// qualifiedName builds a fully qualified proto type name.
func qualifiedName(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

// collectMessages recursively collects all message descriptors including nested.
func collectMessages(result map[string]protoreflect.MessageDescriptor, md protoreflect.MessageDescriptor) {
	fullName := string(md.FullName())
	if _, exists := result[fullName]; !exists {
		result[fullName] = md
	}
	nested := md.Messages()
	for i := range nested.Len() {
		collectMessages(result, nested.Get(i))
	}
}
