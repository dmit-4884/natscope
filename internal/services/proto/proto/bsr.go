// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bsr"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

func (s *Service) bsrRefs(ctx context.Context, src *entities.ProtoSource) ([]entities.ProtoRef, error) {
	m, err := bsr.ParseModule(src.Repository)
	if err != nil {
		return nil, err
	}
	token := ptr.Unwrap(src.Token)
	defaultLabel, err := s.bsrRegistry.DefaultLabel(ctx, m, token)
	if err != nil {
		return nil, err
	}
	labels, err := s.bsrRegistry.ListLabels(ctx, m, token)
	if err != nil {
		return nil, err
	}
	refs := make([]entities.ProtoRef, 0, len(labels))
	for _, l := range labels {
		ref := entities.ProtoRef{Name: l.Name, Kind: entities.RefKindLabel, Revision: l.CommitID}
		if l.Name == defaultLabel {
			refs = append([]entities.ProtoRef{ref}, refs...)
			continue
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (s *Service) resolveBSRRef(ctx context.Context, src *entities.ProtoSource, ref string) (entities.ProtoRef, error) {
	m, err := bsr.ParseModule(src.Repository)
	if err != nil {
		return entities.ProtoRef{}, err
	}
	commit, err := s.bsrRegistry.ResolveRef(ctx, m, ptr.Unwrap(src.Token), ref)
	if err != nil {
		return entities.ProtoRef{}, err
	}
	kind := entities.RefKindLabel
	if bsr.IsCommitID(ref) {
		kind = entities.RefKindCommit
	}
	return entities.ProtoRef{Name: ref, Kind: kind, Revision: commit}, nil
}

func (s *Service) storeBSRRevision(
	ctx context.Context,
	src *entities.ProtoSource,
	ref entities.ProtoRef,
) (*entities.ProtoDescriptor, *entities.CompileOutcome, error) {
	m, err := bsr.ParseModule(src.Repository)
	if err != nil {
		return nil, nil, err
	}
	schema, err := s.bsrRegistry.Schema(ctx, m, ptr.Unwrap(src.Token), ref.Revision)
	if err != nil {
		s.recordCompile(ctx, src.Id, false, err.Error(), 0, 0, nil, nil, "", nil)
		return nil, nil, err
	}
	set, parsed, err := protoutils.NormalizeDescriptorSet(schema.DescriptorSet)
	if err != nil {
		diags := []entities.CompileDiagnostic{{Severity: entities.DiagnosticError, Message: "registry returned a broken descriptor set: " + err.Error()}}
		s.recordCompile(ctx, src.Id, false, diags[0].Message, 0, 0, diags, nil, "", nil)
		return nil, &entities.CompileOutcome{Diagnostics: diags}, nil
	}
	messageTypes := parsed.MessagesIn(schema.OwnFiles)
	d, err := s.saveSchema(ctx, src.Id, ref.Revision, set, messageTypes, schema.OwnFiles, "")
	if err != nil {
		return nil, nil, err
	}
	files := parsed.Files.NumFiles()
	s.recordCompile(ctx, src.Id, true, "", len(messageTypes), files, nil, nil, "", nil)
	return d, &entities.CompileOutcome{Valid: true, MessageTypes: len(messageTypes), FileDescriptors: files}, nil
}

func (s *Service) validateModule(ctx context.Context, module string, token *string) error {
	m, err := bsr.ParseModule(module)
	if err != nil {
		return err
	}
	_, err = s.bsrRegistry.DefaultLabel(ctx, m, ptr.Unwrap(token))
	return err
}
