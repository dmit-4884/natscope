// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package proto

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bsr"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"
)

const (
	bsrCommit1 = "11111111111111111111111111111111"
	bsrCommit2 = "22222222222222222222222222222222"
	paymentV1  = `syntax = "proto3";
package acme;
import "google/protobuf/timestamp.proto";
// A payment.
message Payment { google.protobuf.Timestamp at = 1; }
`
	paymentV2 = `syntax = "proto3";
package acme;
message Payment { string id = 1; }
message Refund { string payment_id = 1; }
`
)

type fakeBSR struct {
	mu      sync.Mutex
	labels  map[string]string
	order   []string
	schemas map[string][]byte
	tokens  []string
	fail    error
}

func newFakeBSR() *fakeBSR {
	return &fakeBSR{labels: map[string]string{}, schemas: map[string][]byte{}}
}

func (f *fakeBSR) label(name, commit string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.labels[name]; !ok {
		f.order = append(f.order, name)
	}
	f.labels[name] = commit
}

func (f *fakeBSR) commit(tb testing.TB, id, content string) {
	set := prototest.DescriptorSet(tb, map[string]string{"acme/payment.proto": content})
	f.mu.Lock()
	defer f.mu.Unlock()
	f.schemas[id] = set
}

func (f *fakeBSR) DefaultLabel(_ context.Context, m bsr.Module, token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens = append(f.tokens, token)
	if m.Name != "payments" {
		return "", &errs.RegistryError{Code: "not_found", Message: "module not found"}
	}
	return "main", nil
}

func (f *fakeBSR) ListLabels(context.Context, bsr.Module, string) ([]bsr.Label, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]bsr.Label, 0, len(f.order))
	for _, name := range f.order {
		out = append(out, bsr.Label{Name: name, CommitID: f.labels[name]})
	}
	return out, nil
}

func (f *fakeBSR) ResolveRef(_ context.Context, _ bsr.Module, _ string, ref string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if commit, ok := f.labels[ref]; ok {
		return commit, nil
	}
	if _, ok := f.schemas[ref]; ok {
		return ref, nil
	}
	return "", fmt.Errorf("%w: %q", errs.ErrProtoRefNotFound, ref)
}

func (f *fakeBSR) Schema(_ context.Context, _ bsr.Module, _ string, commit string) (*bsr.Schema, error) {
	f.mu.Lock()
	set, fail := f.schemas[commit], f.fail
	f.mu.Unlock()
	if fail != nil {
		return nil, fail
	}
	return &bsr.Schema{CommitID: commit, DescriptorSet: set, OwnFiles: []string{"acme/payment.proto"}}, nil
}

func (e *testEnv) createBSR(t *testing.T, module string) *entities.ProtoSource {
	t.Helper()
	src, err := e.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
		Name: "bsr-" + t.Name(), SourceType: entities.SourceTypeBSR, Repository: module, Token: new("bsr-token"),
	})
	require.NoError(t, err)
	return src
}

func TestBSR_SelectLabelAndFollowIt(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.bsr.commit(t, bsrCommit1, paymentV1)
	env.bsr.commit(t, bsrCommit2, paymentV2)
	env.bsr.label("v1", bsrCommit1)
	env.bsr.label("main", bsrCommit1)
	src := env.createBSR(t, "buf.build/acme/payments")

	refs, err := env.svc.ListRefs(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Equal(t, []entities.ProtoRef{
		{Name: "main", Kind: entities.RefKindLabel, Revision: bsrCommit1},
		{Name: "v1", Kind: entities.RefKindLabel, Revision: bsrCommit1},
	}, refs, "the default label comes first")

	got, outcome, err := env.svc.SelectRef(t.Context(), src.Id, "main")
	require.NoError(t, err)
	require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
	assert.Equal(t, 1, outcome.MessageTypes)
	assert.Equal(t, &entities.ProtoRef{Name: "main", Kind: entities.RefKindLabel, Revision: bsrCommit1}, got.SelectedRef)
	assert.Equal(t, bsrCommit1, got.ActiveSchema.Revision)

	types, err := env.svc.ListTypes(t.Context(), src.Id)
	require.NoError(t, err)
	byName := map[string]entities.SchemaType{}
	for _, ty := range types {
		byName[ty.FullName] = ty
	}
	assert.False(t, byName["acme.Payment"].Dependency)
	assert.Equal(t, "A payment.", byName["acme.Payment"].Comment)
	assert.True(t, byName["google.protobuf.Timestamp"].Dependency)

	env.bsr.label("main", bsrCommit2)
	refreshed, _, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Equal(t, bsrCommit2, refreshed.ActiveSchema.Revision)
	assert.ElementsMatch(t, []string{"acme.Payment", "acme.Refund"}, messageNames(env.svc, t, src.Id))

	revisions, err := env.svc.ListRevisions(t.Context(), src.Id)
	require.NoError(t, err)
	assert.Len(t, revisions, 2)
	assert.Contains(t, env.bsr.tokens, "bsr-token")
}

func TestBSR_SelectCommit(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.bsr.commit(t, bsrCommit1, paymentV1)
	src := env.createBSR(t, "buf.build/acme/payments")

	got, outcome, err := env.svc.SelectRef(t.Context(), src.Id, bsrCommit1)
	require.NoError(t, err)
	require.True(t, outcome.Valid)
	assert.Equal(t, entities.RefKindCommit, got.SelectedRef.Kind)
}

func TestBSR_Errors(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.bsr.commit(t, bsrCommit1, paymentV1)
	env.bsr.label("main", bsrCommit1)

	_, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
		Name: "bad module", SourceType: entities.SourceTypeBSR, Repository: "buf.build/acme",
	})
	require.ErrorIs(t, err, errs.ErrInvalidRequest)

	src := env.createBSR(t, "buf.build/acme/payments")
	_, _, err = env.svc.SelectRef(t.Context(), src.Id, "nope")
	require.ErrorIs(t, err, errs.ErrProtoRefNotFound)

	env.bsr.mu.Lock()
	env.bsr.fail = &errs.RegistryError{Code: "unavailable", Message: "registry is down"}
	env.bsr.mu.Unlock()
	_, _, err = env.svc.SelectRef(t.Context(), src.Id, "main")
	require.Error(t, err)
	current, getErr := env.svc.GetSource(t.Context(), src.Id)
	require.NoError(t, getErr)
	assert.False(t, current.LastCompile.Ok)
	assert.Nil(t, current.ActiveSchema)
}

func TestValidateRepository_BSR(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)

	ok, err := env.svc.ValidateRepository(t.Context(), entities.SourceTypeBSR, "buf.build/acme/payments", nil)
	require.NoError(t, err)
	assert.True(t, ok.Valid)

	missing, err := env.svc.ValidateRepository(t.Context(), entities.SourceTypeBSR, "buf.build/acme/ledger", nil)
	require.NoError(t, err)
	assert.False(t, missing.Valid)
	assert.True(t, strings.HasPrefix(*missing.Error, "Module is not accessible: "))

	malformed, err := env.svc.ValidateRepository(t.Context(), entities.SourceTypeBSR, "acme/payments", nil)
	require.NoError(t, err)
	assert.False(t, malformed.Valid)
}

func TestStart_RefreshesMovedBSRLabels(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.bsr.commit(t, bsrCommit1, paymentV1)
	env.bsr.commit(t, bsrCommit2, paymentV2)
	env.bsr.label("main", bsrCommit1)
	src := env.createBSR(t, "buf.build/acme/payments")
	_, _, err := env.svc.SelectRef(t.Context(), src.Id, "main")
	require.NoError(t, err)
	env.bsr.label("main", bsrCommit2)

	env.svc.Start(t.Context())

	require.Eventually(t, func() bool {
		got, err := env.sources.Get(t.Context(), src.Id)
		return err == nil && got.SelectedRef.Revision == bsrCommit2
	}, 10*time.Second, 20*time.Millisecond)
}
