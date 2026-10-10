// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package proto

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"
)

const itemProto = `syntax = "proto3";
package shop;
// One line of an order.
message Item { string sku = 1; }
`

const orderWithItems = `syntax = "proto3";
package shop;
import "shop/item.proto";
message Order { repeated Item items = 1; }
`

func (e *testEnv) createUpload(t *testing.T) *entities.ProtoSource {
	t.Helper()
	src, err := e.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{Name: "upload-" + t.Name(), SourceType: entities.SourceTypeUpload})
	require.NoError(t, err)
	return src
}

func uploadFiles(files map[string]string) entities.SchemaUpload {
	var upload entities.SchemaUpload
	for p, content := range files {
		upload.Files = append(upload.Files, entities.ProtoFileEntry{Path: p, Content: content})
	}
	return upload
}

func TestUploadSchema_Files(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src := env.createUpload(t)

	got, outcome, err := env.svc.UploadSchema(t.Context(), src.Id, uploadFiles(map[string]string{
		"protos/shop/order.proto": orderWithItems,
		"protos/shop/item.proto":  itemProto,
		"protos/README.md":        "# docs",
	}))
	require.NoError(t, err)
	require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
	assert.Equal(t, 2, outcome.MessageTypes)
	assert.Contains(t, outcome.Diagnostics, entities.CompileDiagnostic{
		Severity: entities.DiagnosticInfo, File: "protos/README.md", Message: "skipped: not a .proto or buf config file",
	})
	require.NotNil(t, got.ActiveSchema)
	first := got.ActiveSchema.Revision
	assert.Len(t, first, uploadRevisionChars)
	assert.ElementsMatch(t, []string{"shop.Item", "shop.Order"}, messageNames(env.svc, t, src.Id))

	uploaded, err := env.svc.UploadedSchema(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Equal(t, []string{"protos/shop/item.proto", "protos/shop/order.proto"}, []string{uploaded.Files[0].Path, uploaded.Files[1].Path})

	again, _, err := env.svc.UploadSchema(t.Context(), src.Id, uploadFiles(map[string]string{
		"protos/shop/item.proto":  itemProto,
		"protos/shop/order.proto": orderWithItems,
	}))
	require.NoError(t, err)
	assert.Equal(t, first, again.ActiveSchema.Revision, "the same content keeps its revision")

	changed, _, err := env.svc.UploadSchema(t.Context(), src.Id, uploadFiles(map[string]string{"shop.proto": orderV2}))
	require.NoError(t, err)
	assert.NotEqual(t, first, changed.ActiveSchema.Revision)
	revisions, err := env.svc.ListRevisions(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Len(t, revisions, 2)
	assert.ElementsMatch(t, []string{"shop.Order", "shop.Refund"}, messageNames(env.svc, t, src.Id))
}

func TestUploadSchema_DescriptorSet(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src := env.createUpload(t)
	set := prototest.DescriptorSet(t, map[string]string{"shop/item.proto": itemProto, "shop/order.proto": `syntax = "proto3";
package shop;
import "google/protobuf/timestamp.proto";
message Order { google.protobuf.Timestamp at = 1; }
`})

	got, outcome, err := env.svc.UploadSchema(t.Context(), src.Id, entities.SchemaUpload{DescriptorSet: set})
	require.NoError(t, err)
	require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
	assert.Equal(t, 2, outcome.MessageTypes)
	assert.Equal(t, got.ActiveSchema.Fingerprint[:uploadRevisionChars], got.ActiveSchema.Revision)

	types, err := env.svc.ListTypes(t.Context(), src.Id)
	require.NoError(t, err)
	byName := map[string]entities.SchemaType{}
	for _, ty := range types {
		byName[ty.FullName] = ty
	}
	assert.False(t, byName["shop.Order"].Dependency)
	assert.Equal(t, "One line of an order.", byName["shop.Item"].Comment)
	assert.True(t, byName["google.protobuf.Timestamp"].Dependency)

	uploaded, err := env.svc.UploadedSchema(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Empty(t, uploaded.Files)
	assert.NotEmpty(t, uploaded.DescriptorSet)

	_, _, err = env.svc.RefreshSource(t.Context(), src.Id)
	require.ErrorIs(t, err, errs.ErrInvalidRequest, "a descriptor set has nothing to recompile")
}

func TestUploadSchema_Rejects(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src := env.createUpload(t)
	_, outcome, err := env.svc.UploadSchema(t.Context(), src.Id, uploadFiles(map[string]string{"shop.proto": orderV1}))
	require.NoError(t, err)
	require.True(t, outcome.Valid)
	accepted := outcome

	cases := []struct {
		name   string
		upload entities.SchemaUpload
		want   string
	}{
		{name: "nothing", upload: entities.SchemaUpload{}, want: "no files uploaded"},
		{name: "escaping path", upload: uploadFiles(map[string]string{"../shop.proto": orderV1}), want: "path must be relative and stay inside the upload"},
		{name: "absolute path", upload: uploadFiles(map[string]string{"/etc/shop.proto": orderV1}), want: "path must be relative and stay inside the upload"},
		{name: "no proto files", upload: uploadFiles(map[string]string{"notes.txt": "x"}), want: "no .proto files uploaded"},
		{name: "compile error", upload: uploadFiles(map[string]string{"shop.proto": brokenProto}), want: ""},
		{name: "garbage descriptor set", upload: entities.SchemaUpload{DescriptorSet: []byte("not protobuf")}, want: ""},
		{name: "import from the server disk", upload: uploadFiles(map[string]string{
			"shop.proto": "syntax = \"proto3\";\nimport \"/dev/zero\";\nimport \"../../../../etc/hosts\";\n",
		}), want: ""},
	}
	for _, tc := range cases {
		_, outcome, err := env.svc.UploadSchema(t.Context(), src.Id, tc.upload)
		require.NoError(t, err, tc.name)
		assert.False(t, outcome.Valid, tc.name)
		require.NotEmpty(t, outcome.Diagnostics, tc.name)
		if tc.want != "" {
			first := outcome.Diagnostics[slices.IndexFunc(outcome.Diagnostics, func(d entities.CompileDiagnostic) bool {
				return d.Severity == entities.DiagnosticError
			})]
			assert.Equal(t, tc.want, first.Message, tc.name)
		}
	}

	current, err := env.svc.GetSource(t.Context(), src.Id)
	require.NoError(t, err)
	assert.False(t, current.LastCompile.Ok)
	assert.Equal(t, []string{"shop.Order"}, messageNames(env.svc, t, src.Id), "failed uploads keep the previous schema")
	assert.True(t, accepted.Valid)

	git := env.createGit(t)
	_, _, err = env.svc.UploadSchema(t.Context(), git.Id, uploadFiles(map[string]string{"shop.proto": orderV1}))
	require.ErrorIs(t, err, errs.ErrInvalidRequest)
}

func TestRefreshSource_UploadRecompiles(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src := env.createUpload(t)

	_, _, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.ErrorIs(t, err, errs.ErrInvalidRequest, "nothing uploaded yet")

	uploaded, _, err := env.svc.UploadSchema(t.Context(), src.Id, uploadFiles(map[string]string{"shop.proto": orderV1}))
	require.NoError(t, err)
	_, err = env.descriptors.DeleteBySource(t.Context(), src.Id)
	require.NoError(t, err)

	refreshed, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	require.True(t, outcome.Valid)
	assert.Equal(t, uploaded.ActiveSchema.Revision, refreshed.ActiveSchema.Revision)
	assert.Equal(t, []string{"shop.Order"}, messageNames(env.svc, t, src.Id))
}

func TestEncode_Framing(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	src := env.createUpload(t)
	_, outcome, err := env.svc.UploadSchema(t.Context(), src.Id, uploadFiles(map[string]string{"shop.proto": orderV2}))
	require.NoError(t, err)
	require.True(t, outcome.Valid)

	req := entities.CodecRequest{
		SourceID: src.Id, MessageType: "shop.Refund", JSON: []byte(`{"order_id":"o1"}`),
		Framing: entities.Framing{Kind: entities.FramingConfluent, SchemaID: 7},
	}
	raw, err := env.svc.EncodeRaw(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, []byte{0, 0, 0, 0, 7, 2, 2, 0x0a, 0x02, 'o', '1'}, raw)

	result, violations, err := env.svc.EncodeWithValidation(t.Context(), req)
	require.NoError(t, err)
	require.True(t, result.Success)
	assert.Empty(t, violations)
	assert.Equal(t, len(raw), result.DataSize)

	decoded, err := env.svc.Decode(t.Context(), entities.CodecRequest{
		SourceID: src.Id, MessageType: "shop.Refund", Data: raw, Framing: req.Framing,
	})
	require.NoError(t, err)
	require.True(t, decoded.Success, decoded.Error)
	assert.JSONEq(t, `{"order_id":"o1"}`, string(decoded.Decoded))
}

func TestUploadSchema_PrunesOldRevisions(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	mappings := &fakeMappings{}
	env.svc.mappingsService = mappings
	src := env.createUpload(t)

	var fingerprints, revisions []string
	for i := range maxStoredRevisions + 2 {
		got, outcome, err := env.svc.UploadSchema(t.Context(), src.Id, uploadFiles(map[string]string{
			"shop.proto": fmt.Sprintf("syntax = \"proto3\";\npackage shop;\nmessage V%d {}\n", i),
		}))
		require.NoError(t, err)
		require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
		fingerprints = append(fingerprints, got.ActiveSchema.Fingerprint)
		revisions = append(revisions, got.ActiveSchema.Revision)
		if i == 0 {
			mappings.all = entities.SubjectMappings{{SourceID: src.Id, PinnedFingerprint: &fingerprints[0]}}
		}
	}

	stored, err := env.svc.ListRevisions(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Len(t, stored, maxStoredRevisions+1)
	_, err = env.descriptors.GetByFingerprint(t.Context(), src.Id, fingerprints[0])
	require.NoError(t, err, "a pinned revision stays")
	_, err = env.descriptors.GetBySourceRevision(t.Context(), src.Id, revisions[1])
	require.ErrorIs(t, err, errs.ErrProtoDescriptorNotFound)
	assert.True(t, stored[0].Active)
}
