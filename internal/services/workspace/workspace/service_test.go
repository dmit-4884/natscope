// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package workspace

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	workspacesvc "github.com/dmit-4884/natscope/internal/services/workspace"
)

// fakeSection is a configurable Section stub.
type fakeSection struct {
	key      string
	title    string
	count    int32
	exported json.RawMessage
	report   entities.WorkspaceSectionReport
	result   entities.WorkspaceSectionResult

	lastStrategy entities.WorkspaceStrategy
	importCalls  int
}

func (f *fakeSection) Key() string { return f.key }
func (f *fakeSection) Describe(context.Context) (entities.WorkspaceSectionInfo, error) {
	return entities.WorkspaceSectionInfo{Key: f.key, Title: f.title, Count: f.count}, nil
}
func (f *fakeSection) Export(context.Context) (json.RawMessage, error) { return f.exported, nil }
func (f *fakeSection) Validate(_ context.Context, _ json.RawMessage, s entities.WorkspaceStrategy) (entities.WorkspaceSectionReport, error) {
	f.lastStrategy = s
	return f.report, nil
}
func (f *fakeSection) Import(_ context.Context, _ json.RawMessage, s entities.WorkspaceStrategy) (entities.WorkspaceSectionResult, error) {
	f.lastStrategy = s
	f.importCalls++
	return f.result, nil
}

func newSvc(secs ...workspacesvc.Section) *Service { return New(secs) }

func TestListSections_SortedAndComplete(t *testing.T) {
	t.Parallel()
	svc := newSvc(
		&fakeSection{key: "templates", title: "Templates", count: 2},
		&fakeSection{key: "mappings", title: "Mappings", count: 5},
	)
	infos, err := svc.ListSections(t.Context())
	require.NoError(t, err)
	require.Len(t, infos, 2)
	// Sorted by key for stable output.
	assert.Equal(t, "mappings", infos[0].Key)
	assert.Equal(t, "templates", infos[1].Key)
	assert.Equal(t, int32(5), infos[0].Count)
}

func TestExport_OnlySelectedKeys(t *testing.T) {
	t.Parallel()
	svc := newSvc(
		&fakeSection{key: "mappings", exported: json.RawMessage(`{"v":1}`)},
		&fakeSection{key: "templates", exported: json.RawMessage(`{"v":2}`)},
	)
	payload, err := svc.Export(t.Context(), []string{"mappings"})
	require.NoError(t, err)

	var file entities.WorkspaceFile
	require.NoError(t, json.Unmarshal(payload, &file))
	assert.Equal(t, entities.WorkspaceFileVersion, file.Version)
	assert.Contains(t, file.Sections, "mappings")
	assert.NotContains(t, file.Sections, "templates")
}

func TestExport_EmptyKeysMeansAll(t *testing.T) {
	t.Parallel()
	svc := newSvc(
		&fakeSection{key: "mappings", exported: json.RawMessage(`{}`)},
		&fakeSection{key: "templates", exported: json.RawMessage(`{}`)},
	)
	payload, err := svc.Export(t.Context(), nil)
	require.NoError(t, err)
	var file entities.WorkspaceFile
	require.NoError(t, json.Unmarshal(payload, &file))
	assert.Len(t, file.Sections, 2)
}

func buildFile(t *testing.T, sections map[string]any) []byte {
	t.Helper()
	raw := map[string]json.RawMessage{}
	for k, v := range sections {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		raw[k] = b
	}
	payload, err := json.Marshal(entities.WorkspaceFile{Version: 1, Sections: raw})
	require.NoError(t, err)
	return payload
}

func TestImport_PassesStrategyAndSkipsUnknown(t *testing.T) {
	t.Parallel()
	mappings := &fakeSection{key: "mappings", result: entities.WorkspaceSectionResult{Created: 3}}
	svc := newSvc(mappings)

	payload := buildFile(t, map[string]any{
		"mappings": map[string]any{"version": 1, "items": []any{}},
		"unknownX": map[string]any{"version": 9},
	})

	results, err := svc.Import(t.Context(), payload, nil, entities.WorkspaceStrategyReplace)
	require.NoError(t, err)
	require.Len(t, results, 2)

	byKey := map[string]entities.WorkspaceSectionResult{}
	for _, r := range results {
		byKey[r.Key] = r
	}
	assert.Equal(t, int32(3), byKey["mappings"].Created)
	assert.Equal(t, entities.WorkspaceStrategyReplace, mappings.lastStrategy)
	assert.Equal(t, 1, mappings.importCalls)

	// Unknown section is reported with a warning, not applied.
	require.Contains(t, byKey, "unknownX")
	assert.NotEmpty(t, byKey["unknownX"].Warnings)
}

func TestValidate_UnknownSectionFlagged(t *testing.T) {
	t.Parallel()
	svc := newSvc(&fakeSection{key: "mappings"})
	payload := buildFile(t, map[string]any{"futureThing": map[string]any{"version": 2}})

	reports, err := svc.Validate(t.Context(), payload, nil, entities.WorkspaceStrategyMerge)
	require.NoError(t, err)
	require.Len(t, reports, 1)
	assert.True(t, reports[0].Unknown)
	assert.Equal(t, "futureThing", reports[0].Key)
}

func TestImport_InvalidFileRejected(t *testing.T) {
	t.Parallel()
	svc := newSvc(&fakeSection{key: "mappings"})
	_, err := svc.Import(t.Context(), []byte("not json"), nil, entities.WorkspaceStrategyMerge)
	require.Error(t, err)

	// A JSON object without a sections envelope is also rejected.
	_, err = svc.Validate(t.Context(), []byte(`{"foo":1}`), nil, entities.WorkspaceStrategyMerge)
	require.Error(t, err)
}

func TestNew_DeduplicatesKeysAndKeepsOrderStable(t *testing.T) {
	t.Parallel()
	// Two sections with the same key: last wins, ListSections stays unique.
	svc := newSvc(
		&fakeSection{key: "dup", title: "first"},
		&fakeSection{key: "dup", title: "second"},
		&fakeSection{key: "other"},
	)
	infos, err := svc.ListSections(t.Context())
	require.NoError(t, err)

	keys := map[string]int{}
	for _, i := range infos {
		keys[i.Key]++
	}
	for k, n := range keys {
		assert.Equal(t, 1, n, "key %q must be unique", k)
	}
	assert.Len(t, infos, 2)
}
