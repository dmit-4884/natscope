// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bsr_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/bsr"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils/prototest"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const commit = "0123456789abcdef0123456789abcdef"

type fakeRegistry struct {
	t        *testing.T
	set      *descriptorpb.FileDescriptorSet
	mu       sync.Mutex
	requests []map[string]any
	auth     []string
}

func (f *fakeRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	require.NoError(f.t, json.NewDecoder(r.Body).Decode(&body))
	f.mu.Lock()
	f.requests = append(f.requests, body)
	f.auth = append(f.auth, r.Header.Get("Authorization"))
	f.mu.Unlock()
	assert.Equal(f.t, "1", r.Header.Get("Connect-Protocol-Version"))

	write := func(v any) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(f.t, json.NewEncoder(w).Encode(v))
	}
	switch strings.TrimPrefix(r.URL.Path, "/") {
	case "buf.registry.module.v1.ModuleService/GetModules":
		write(map[string]any{"modules": []any{map[string]any{"name": "payments", "defaultLabelName": "main"}}})
	case "buf.registry.module.v1.LabelService/ListLabels":
		if body["pageToken"] == "" {
			write(map[string]any{"nextPageToken": "p2", "labels": []any{
				map[string]any{"name": "main", "commitId": commit},
				map[string]any{"name": "old", "commitId": commit, "archiveTime": "2026-01-01T00:00:00Z"},
			}})
			return
		}
		write(map[string]any{"labels": []any{map[string]any{"name": "v1.0.0", "commitId": strings.Repeat("a", 32)}}})
	case "buf.registry.module.v1.CommitService/GetCommits":
		ref := body["resourceRefs"].([]any)[0].(map[string]any)["name"].(map[string]any)["ref"]
		if ref == "missing" {
			w.WriteHeader(http.StatusNotFound)
			write(map[string]any{"code": "not_found", "message": "resource was not found"})
			return
		}
		write(map[string]any{"commits": []any{map[string]any{"id": commit}}})
	case "buf.registry.module.v1.FileDescriptorSetService/GetFileDescriptorSet":
		set := f.set
		if body["excludeImports"] == true {
			set = &descriptorpb.FileDescriptorSet{}
			for _, file := range f.set.GetFile() {
				if !strings.HasPrefix(file.GetName(), "google/") {
					set.File = append(set.File, file)
				}
			}
		}
		raw, err := protojson.Marshal(set)
		require.NoError(f.t, err)
		write(map[string]json.RawMessage{"fileDescriptorSet": raw})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newFake(t *testing.T) (*fakeRegistry, bsr.Module) {
	t.Helper()
	set := &descriptorpb.FileDescriptorSet{}
	require.NoError(t, proto.Unmarshal(prototest.DescriptorSet(t, map[string]string{"acme/payments.proto": `syntax = "proto3";
package acme;
import "google/protobuf/timestamp.proto";
// A payment.
message Payment { google.protobuf.Timestamp at = 1; }
`}), set))
	fake := &fakeRegistry{t: t, set: set}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	m, err := bsr.ParseModule(srv.URL + "/acme/payments")
	require.NoError(t, err)
	return fake, m
}

func TestParseModule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want bsr.Module
	}{
		{name: "bare", in: "buf.build/acme/payments", want: bsr.Module{BaseURL: "https://buf.build", Owner: "acme", Name: "payments"}},
		{name: "https", in: " https://acme.buf.dev/acme/payments/ ", want: bsr.Module{BaseURL: "https://acme.buf.dev", Owner: "acme", Name: "payments"}},
		{name: "http", in: "http://127.0.0.1:9/acme/payments", want: bsr.Module{BaseURL: "http://127.0.0.1:9", Owner: "acme", Name: "payments"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := bsr.ParseModule(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
	for _, bad := range []string{"", "buf.build/acme", "buf.build//payments", "buf.build/acme/payments/extra"} {
		_, err := bsr.ParseModule(bad)
		require.ErrorIs(t, err, errs.ErrInvalidRequest, bad)
	}
}

func TestClient(t *testing.T) {
	t.Parallel()
	fake, m := newFake(t)
	client := bsr.New(http.DefaultClient)

	label, err := client.DefaultLabel(t.Context(), m, "secret")
	require.NoError(t, err)
	assert.Equal(t, "main", label)

	labels, err := client.ListLabels(t.Context(), m, "")
	require.NoError(t, err)
	assert.Equal(t, []bsr.Label{{Name: "main", CommitID: commit}, {Name: "v1.0.0", CommitID: strings.Repeat("a", 32)}}, labels)

	resolved, err := client.ResolveRef(t.Context(), m, "", "main")
	require.NoError(t, err)
	assert.Equal(t, commit, resolved)
	_, err = client.ResolveRef(t.Context(), m, "", "missing")
	require.ErrorIs(t, err, errs.ErrProtoRefNotFound)

	schema, err := client.Schema(t.Context(), m, "", commit)
	require.NoError(t, err)
	assert.Equal(t, []string{"acme/payments.proto"}, schema.OwnFiles)
	got := &descriptorpb.FileDescriptorSet{}
	require.NoError(t, proto.Unmarshal(schema.DescriptorSet, got))
	assert.Len(t, got.GetFile(), 2)
	assert.Equal(t, " A payment.\n", got.GetFile()[1].GetSourceCodeInfo().GetLocation()[0].GetLeadingComments())

	assert.Equal(t, "Bearer secret", fake.auth[0])
	assert.Empty(t, fake.auth[1])
}

func TestClient_Errors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"unauthenticated","message":"token is invalid"}`))
	}))
	t.Cleanup(srv.Close)
	m, err := bsr.ParseModule(srv.URL + "/acme/private")
	require.NoError(t, err)

	_, err = bsr.New(http.DefaultClient).ListLabels(t.Context(), m, "bad")
	regErr, ok := errors.AsType[*errs.RegistryError](err)
	require.True(t, ok, "error: %v", err)
	assert.Equal(t, &errs.RegistryError{Code: "unauthenticated", Message: "token is invalid"}, regErr)

	unreachable, err := bsr.ParseModule("http://127.0.0.1:1/acme/payments")
	require.NoError(t, err)
	_, err = bsr.New(http.DefaultClient).DefaultLabel(t.Context(), unreachable, "")
	regErr, ok = errors.AsType[*errs.RegistryError](err)
	require.True(t, ok)
	assert.Equal(t, "unavailable", regErr.Code)

	assert.True(t, bsr.IsCommitID(commit))
	assert.False(t, bsr.IsCommitID("main"))
}
