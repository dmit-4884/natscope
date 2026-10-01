// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/altessa-s/go-atlas/core/encoding/hash"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

const (
	maxUploadFileBytes  = 5 << 20
	maxUploadTotalBytes = 24 << 20
	uploadRevisionChars = 12
)

var uploadConfigNames = []string{"buf.yaml", "buf.work.yaml", "buf.lock"}

// UploadSchema makes uploaded .proto files, or a compiled descriptor set, the active schema of an upload source.
func (s *Service) UploadSchema(
	ctx context.Context,
	sourceID string,
	upload entities.SchemaUpload,
) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	src, err := s.uploadSource(ctx, sourceID)
	if err != nil {
		return nil, nil, err
	}

	unlock := s.compileLocks.lock(sourceID)
	var (
		d       *entities.ProtoDescriptor
		outcome *entities.CompileOutcome
	)
	if len(upload.DescriptorSet) > 0 {
		d, outcome, err = s.storeUploadedSet(ctx, src, upload.DescriptorSet)
	} else {
		d, outcome, err = s.compileUploadedFiles(ctx, src, upload.Files)
	}
	unlock()
	if err != nil {
		return nil, outcome, err
	}
	if outcome.Valid {
		s.registryCache.InvalidateSource(sourceID)
		s.notifyReload(ctx)
		s.logger.InfoContext(ctx, "uploaded proto schema",
			slog.String("source_id", sourceID),
			slog.String("revision", d.Revision),
			slog.Int("message_types", len(d.MessageTypes)))
	}
	updated, err := s.GetSource(ctx, sourceID)
	return updated, outcome, err
}

// UploadedSchema returns what was uploaded for the active schema of an upload source.
func (s *Service) UploadedSchema(ctx context.Context, sourceID string) (*entities.SchemaUpload, error) {
	src, err := s.uploadSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if src.ActiveSchema == nil {
		return nil, errs.ErrMappingSelectionMissing
	}
	fileSet, err := s.fileSetsStorage.GetBySourceRevision(ctx, sourceID, src.ActiveSchema.Revision)
	switch {
	case err == nil:
		return &entities.SchemaUpload{Files: slices.Concat(fileSet.Configs, fileSet.Files)}, nil
	case !errors.Is(err, errs.ErrProtoFileSetNotFound):
		return nil, err
	}
	d, err := s.descriptorsStorage.GetBySourceRevision(ctx, sourceID, src.ActiveSchema.Revision)
	if err != nil {
		return nil, err
	}
	return &entities.SchemaUpload{DescriptorSet: d.DescriptorSet}, nil
}

func (s *Service) uploadSource(ctx context.Context, sourceID string) (*entities.ProtoSource, error) {
	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, err
	}
	if src.SourceType != entities.SourceTypeUpload {
		return nil, fmt.Errorf("%w: only upload sources take uploaded schemas", errs.ErrInvalidRequest)
	}
	return src, nil
}

func (s *Service) storeUploadedSet(
	ctx context.Context,
	src *entities.ProtoSource,
	data []byte,
) (*entities.ProtoDescriptor, *entities.CompileOutcome, error) {
	set, schema, err := protoutils.NormalizeDescriptorSet(data)
	if err != nil {
		diags := []entities.CompileDiagnostic{{
			Severity: entities.DiagnosticError,
			Message:  "not a valid descriptor set: " + err.Error(),
			Hint:     "upload the output of `buf build -o schema.binpb` or `protoc --include_imports --descriptor_set_out`",
		}}
		s.recordCompile(ctx, src.Id, false, diags[0].Message, 0, 0, diags, nil, "", nil)
		return nil, &entities.CompileOutcome{Diagnostics: diags}, nil
	}

	own := protoutils.OwnFiles(schema)
	messageTypes := schema.MessagesIn(own)

	fingerprint := hash.SHA256HexBytes(set)
	d, err := s.saveSchema(ctx, src.Id, fingerprint[:uploadRevisionChars], set, messageTypes, own, "")
	if err != nil {
		return nil, nil, err
	}
	files := schema.Files.NumFiles()
	s.recordCompile(ctx, src.Id, true, "", len(messageTypes), files, nil, nil, "", schemaRevision(d, true))
	return d, &entities.CompileOutcome{Valid: true, MessageTypes: len(messageTypes), FileDescriptors: files}, nil
}

func (s *Service) compileUploadedFiles(
	ctx context.Context,
	src *entities.ProtoSource,
	upload []entities.ProtoFileEntry,
) (*entities.ProtoDescriptor, *entities.CompileOutcome, error) {
	files, configs, diags := sortUpload(upload)
	if hasErrors(diags) {
		s.recordCompile(ctx, src.Id, false, summarizeDiagnostics(diags), 0, len(files), diags, nil, "", nil)
		return nil, &entities.CompileOutcome{Diagnostics: diags}, nil
	}
	return s.compileFileSet(ctx, src, entities.ProtoFileSetNew(func(fs *entities.ProtoFileSet) {
		fs.SourceID = src.Id
		fs.Revision = uploadRevision(files, configs)
		fs.Files = files
		fs.Configs = configs
		fs.FetchedAt = time.Now().UTC()
	}), diags)
}

func (s *Service) compileFileSet(
	ctx context.Context,
	src *entities.ProtoSource,
	fileSet *entities.ProtoFileSet,
	diags []entities.CompileDiagnostic,
) (*entities.ProtoDescriptor, *entities.CompileOutcome, error) {
	out, err := s.compile(ctx, src, fileSet.Files, fileSet.Configs, "")
	if err != nil {
		s.recordCompile(ctx, src.Id, false, err.Error(), 0, len(fileSet.Files), diags, nil, "", nil)
		return nil, nil, coreerrs.WrapOperation(err, "compile proto files")
	}
	diags = append(diags, out.Diags...)
	if out.HasErrors() {
		s.recordCompile(ctx, src.Id, false, summarizeDiagnostics(diags), 0, len(fileSet.Files),
			diags, out.Roots, string(out.Origin), nil)
		return nil, &entities.CompileOutcome{Diagnostics: diags}, nil
	}
	if saveErr := s.fileSetsStorage.Save(ctx, fileSet); saveErr != nil {
		return nil, nil, coreerrs.WrapOperation(saveErr, "save proto files")
	}
	d, err := s.storeSchema(ctx, src, fileSet.Revision, out.FDS)
	if err != nil {
		return nil, nil, err
	}
	s.recordCompile(ctx, src.Id, true, "", len(d.MessageTypes), len(out.FDS),
		diags, out.Roots, string(out.Origin), schemaRevision(d, true))
	return d, &entities.CompileOutcome{
		Valid:           true,
		MessageTypes:    len(d.MessageTypes),
		FileDescriptors: len(out.FDS),
		Diagnostics:     diags,
	}, nil
}

func (s *Service) compileUpload(ctx context.Context, src *entities.ProtoSource) (*entities.CompileOutcome, error) {
	if src.ActiveSchema == nil {
		return nil, fmt.Errorf("%w: upload a schema first", errs.ErrInvalidRequest)
	}
	unlock := s.compileLocks.lock(src.Id)
	defer unlock()

	fileSet, err := s.fileSetsStorage.GetBySourceRevision(ctx, src.Id, src.ActiveSchema.Revision)
	if errors.Is(err, errs.ErrProtoFileSetNotFound) {
		return nil, fmt.Errorf("%w: a descriptor set upload has nothing to recompile", errs.ErrInvalidRequest)
	}
	if err != nil {
		return nil, err
	}
	_, outcome, err := s.compileFileSet(ctx, src, fileSet, nil)
	if err == nil && outcome.Valid {
		s.notifyReload(ctx)
	}
	return outcome, err
}

func sortUpload(upload []entities.ProtoFileEntry) (files, configs []entities.ProtoFileEntry, diags []entities.CompileDiagnostic) {
	if len(upload) == 0 {
		return nil, nil, []entities.CompileDiagnostic{{Severity: entities.DiagnosticError, Message: "no files uploaded"}}
	}
	seen := make(map[string]bool, len(upload))
	total := 0
	for _, f := range upload {
		p, ok := cleanUploadPath(f.Path)
		size := len(f.Content)
		total += size
		switch {
		case !ok:
			diags = append(diags, uploadError(f.Path, "path must be relative and stay inside the upload"))
		case seen[p]:
			diags = append(diags, uploadError(p, "uploaded twice"))
		case size > maxUploadFileBytes:
			diags = append(diags, uploadError(p, fmt.Sprintf("file too large: %d bytes (max %d)", size, maxUploadFileBytes)))
		case slices.Contains(uploadConfigNames, path.Base(p)):
			configs = append(configs, entities.ProtoFileEntry{Path: p, Content: f.Content, Size: int64(size)})
		case strings.HasSuffix(p, ".proto"):
			files = append(files, entities.ProtoFileEntry{Path: p, Content: f.Content, Size: int64(size)})
		default:
			diags = append(diags, entities.CompileDiagnostic{Severity: entities.DiagnosticInfo, File: p, Message: "skipped: not a .proto or buf config file"})
		}
		seen[p] = true
	}
	if total > maxUploadTotalBytes {
		diags = append(diags, uploadError("", fmt.Sprintf("upload too large: %d bytes (max %d)", total, maxUploadTotalBytes)))
	}
	if len(files) == 0 && !hasErrors(diags) {
		diags = append(diags, uploadError("", "no .proto files uploaded"))
	}
	byPath := func(a, b entities.ProtoFileEntry) int { return cmp.Compare(a.Path, b.Path) }
	slices.SortFunc(files, byPath)
	slices.SortFunc(configs, byPath)
	return files, configs, diags
}

func cleanUploadPath(p string) (string, bool) {
	p = path.Clean(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
	if p == "." || p == ".." || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "../") {
		return "", false
	}
	return p, true
}

func uploadError(file, msg string) entities.CompileDiagnostic {
	return entities.CompileDiagnostic{Severity: entities.DiagnosticError, File: file, Message: msg}
}

func hasErrors(diags []entities.CompileDiagnostic) bool {
	return slices.ContainsFunc(diags, func(d entities.CompileDiagnostic) bool { return d.Severity == entities.DiagnosticError })
}

func uploadRevision(files, configs []entities.ProtoFileEntry) string {
	var b strings.Builder
	for _, f := range slices.Concat(configs, files) {
		b.WriteString(f.Path)
		b.WriteByte(0)
		b.WriteString(f.Content)
		b.WriteByte(0)
	}
	return hash.SHA256HexBytes([]byte(b.String()))[:uploadRevisionChars]
}
