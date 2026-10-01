// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	"golang.org/x/time/rate"

	"google.golang.org/protobuf/encoding/protowire"

	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
)

const detectProto = `syntax = "proto3";
package det;
message User { string name = 1; int64 id = 2; string email = 3; }
message Note { string text = 1; }
`

type fakeMappings struct {
	mappingssvc.Service
	resolver *natsutil.MappingResolver
	all      entities.SubjectMappings
}

func (f *fakeMappings) Resolver(context.Context) *natsutil.MappingResolver { return f.resolver }

func (f *fakeMappings) GetAll(context.Context) (entities.SubjectMappings, error) { return f.all, nil }

func userWire(name string) []byte {
	var b []byte
	b = protowire.AppendTag(b, 1, protowire.BytesType)
	b = protowire.AppendString(b, name)
	b = protowire.AppendTag(b, 2, protowire.VarintType)
	b = protowire.AppendVarint(b, 7)
	b = protowire.AppendTag(b, 3, protowire.BytesType)
	return protowire.AppendString(b, "a@b")
}

func unknownWire() []byte {
	b := protowire.AppendTag(nil, 9, protowire.VarintType)
	return protowire.AppendVarint(b, 1)
}

func newDetectEnv(t *testing.T, mappings entities.SubjectMappings) (*testEnv, string) {
	t.Helper()
	env := newTestEnv(t)
	env.svc.mappingsService = &fakeMappings{resolver: natsutil.NewMappingResolver(mappings)}
	src, _ := env.createLocal(t, map[string]string{"det.proto": detectProto})
	_, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
	require.NoError(t, err)
	require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
	return env, src.Id
}

func TestDetectTypes_Service(t *testing.T) {
	t.Parallel()
	env, sourceID := newDetectEnv(t, nil)

	got, err := env.svc.DetectTypes(t.Context(), userWire("ann"), "", 5)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "det.User", got[0].MessageType)
	assert.Equal(t, sourceID, got[0].SourceID)
	assert.Equal(t, LocalRevision, got[0].SourceRevision)
	assert.JSONEq(t, `{"name":"ann","id":"7","email":"a@b"}`, string(got[0].Decoded))

	_, err = env.svc.DetectTypes(t.Context(), userWire("ann"), "missing", 5)
	require.Error(t, err)
}

func TestAutoDecode(t *testing.T) {
	t.Parallel()

	t.Run("learns the subject", func(t *testing.T) {
		t.Parallel()
		env, sourceID := newDetectEnv(t, nil)

		r := env.svc.autoDecode(t.Context(), "users.1001.created", userWire("ann"))
		require.NotNil(t, r)
		assert.True(t, r.Auto)
		assert.Equal(t, "det.User", r.MessageType)
		assert.Equal(t, sourceID, r.SourceID)

		learned, ok := env.svc.learned.Get("users.*.created")
		require.True(t, ok)
		assert.Equal(t, "det.User", learned.messageType)

		next := env.svc.autoDecode(t.Context(), "users.2002.created", userWire("bob"))
		require.NotNil(t, next)
		assert.JSONEq(t, `{"name":"bob","id":"7","email":"a@b"}`, string(next.Decoded))
	})

	t.Run("gives up after repeated misses", func(t *testing.T) {
		t.Parallel()
		env, _ := newDetectEnv(t, nil)

		for range maxDetectMisses {
			assert.Nil(t, env.svc.autoDecode(t.Context(), "noise", unknownWire()))
		}
		learned, _ := env.svc.learned.Get("noise")
		assert.Equal(t, maxDetectMisses, learned.misses)
		assert.Nil(t, env.svc.autoDecode(t.Context(), "noise", userWire("ann")))
	})

	t.Run("redetects when the learned type stops fitting", func(t *testing.T) {
		t.Parallel()
		env, _ := newDetectEnv(t, nil)
		env.svc.learned.Put("notes", learnedType{sourceID: "gone", messageType: "det.Gone"})

		r := env.svc.autoDecode(t.Context(), "notes", userWire("ann"))
		require.NotNil(t, r)
		assert.Equal(t, "det.User", r.MessageType)
	})

	t.Run("an exhausted scan budget skips detection without counting a miss", func(t *testing.T) {
		t.Parallel()
		env, _ := newDetectEnv(t, nil)
		env.svc.detectBudget = rate.NewLimiter(0, 0)

		assert.Nil(t, env.svc.autoDecode(t.Context(), "busy", userWire("ann")))
		_, seen := env.svc.learned.Get("busy")
		assert.False(t, seen)
	})

	t.Run("schema reload forgets", func(t *testing.T) {
		t.Parallel()
		env, _ := newDetectEnv(t, nil)
		require.NotNil(t, env.svc.autoDecode(t.Context(), "users", userWire("ann")))

		env.svc.notifyReload(t.Context())
		_, ok := env.svc.learned.Get("users")
		assert.False(t, ok)
	})
}

func TestAutoDecode_TieBehindDuplicateSources(t *testing.T) {
	t.Parallel()
	env := newTestEnv(t)
	env.svc.mappingsService = &fakeMappings{resolver: natsutil.NewMappingResolver(nil)}
	for i, content := range []string{
		"syntax = \"proto3\";\npackage dup;\nmessage Alpha { string name = 1; }\n",
		"syntax = \"proto3\";\npackage dup;\nmessage Alpha { string name = 1; }\n",
		"syntax = \"proto3\";\npackage dup;\nmessage Beta { string title = 1; }\n",
	} {
		dir := t.TempDir()
		writeTree(t, dir, map[string]string{"dup.proto": content})
		src, err := env.svc.CreateSource(t.Context(), &entities.ProtoSourceCreate{
			Name: fmt.Sprintf("dup-%d", i), SourceType: entities.SourceTypeLocal, LocalPath: &dir,
		})
		require.NoError(t, err)
		_, outcome, err := env.svc.RefreshSource(t.Context(), src.Id)
		require.NoError(t, err)
		require.True(t, outcome.Valid, "diagnostics: %v", outcome.Diagnostics)
	}
	payload := protowire.AppendString(protowire.AppendTag(nil, 1, protowire.BytesType), "x")

	assert.Nil(t, env.svc.autoDecode(t.Context(), "dup.one", payload))
}

func TestDecodeMessages_Detect(t *testing.T) {
	t.Parallel()
	env, sourceID := newDetectEnv(t, nil)
	newMsg := func(subject string, data []byte) *entities.Message {
		return &entities.Message{
			Subject: subject, DataBase64: base64.StdEncoding.EncodeToString(data), ContentType: entities.DetectContentType(data),
		}
	}

	auto := newMsg("users.created", userWire("ann"))
	text := newMsg("logs", []byte("plain text"))
	env.svc.DecodeMessages(t.Context(), []*entities.Message{auto, text}, true)

	assert.True(t, auto.DecodedAuto)
	assert.Equal(t, "det.User", auto.DecodedType)
	assert.Equal(t, sourceID, auto.DecodedSourceID)
	assert.JSONEq(t, `{"name":"ann","id":"7","email":"a@b"}`, string(auto.Decoded))
	assert.Nil(t, text.Decoded)

	off := newMsg("users.created", userWire("ann"))
	env.svc.DecodeMessages(t.Context(), []*entities.Message{off}, false)
	assert.Nil(t, off.Decoded)
	assert.False(t, off.DecodedAuto)
}

func TestDecodeSubject(t *testing.T) {
	t.Parallel()
	env, sourceID := newDetectEnv(t, nil)
	env.svc.mappingsService = &fakeMappings{resolver: natsutil.NewMappingResolver(entities.SubjectMappings{
		{BaseEntity: entities.BaseEntity{Id: "m1"}, Pattern: "$KV.notes.>", MessageType: "det.Note", SourceID: sourceID},
	})}
	ctx := t.Context()

	mapped := env.svc.DecodeSubject(ctx, "$KV.notes.a", userWire("ann"), false)
	require.NotNil(t, mapped)
	assert.Equal(t, "det.Note", mapped.MessageType)
	assert.Equal(t, sourceID, mapped.SourceID)
	assert.False(t, mapped.Auto)
	assert.True(t, mapped.Success)
	assert.NotEmpty(t, mapped.UnknownFields)

	auto := env.svc.DecodeSubject(ctx, "$KV.users.a", userWire("ann"), true)
	require.NotNil(t, auto)
	assert.True(t, auto.Auto)
	assert.Equal(t, "det.User", auto.MessageType)

	assert.Nil(t, env.svc.DecodeSubject(ctx, "$KV.users.b", userWire("ann"), false))
	assert.Nil(t, env.svc.DecodeSubject(ctx, "$KV.users.c", []byte("plain text"), true))

	missing := &fakeMappings{resolver: natsutil.NewMappingResolver(entities.SubjectMappings{
		{BaseEntity: entities.BaseEntity{Id: "m2"}, Pattern: "$KV.gone.>", MessageType: "det.Note", SourceID: "missing"},
	})}
	env.svc.mappingsService = missing
	broken := env.svc.DecodeSubject(ctx, "$KV.gone.a", userWire("ann"), true)
	require.NotNil(t, broken)
	assert.False(t, broken.Success)
	assert.NotEmpty(t, broken.Error)
	assert.Equal(t, "det.Note", broken.MessageType)
}

func TestLiveDecoder_Detect(t *testing.T) {
	t.Parallel()
	env, sourceID := newDetectEnv(t, entities.SubjectMappings{})

	on := env.svc.NewLiveDecoder(true)
	on.Init(t.Context())
	r := on.Decode(t.Context(), userWire("ann"), "users.created")
	require.NotNil(t, r)
	assert.True(t, r.Auto)
	assert.Equal(t, sourceID, r.SourceID)
	assert.Nil(t, on.Decode(t.Context(), []byte(`{"name":"ann"}`), "users.json"))

	off := env.svc.NewLiveDecoder(false)
	off.Init(t.Context())
	assert.Nil(t, off.Decode(t.Context(), userWire("ann"), "users.created"))
}

func TestGeneralizeSubject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		subject string
		want    string
	}{
		{name: "static", subject: "orders.created", want: "orders.created"},
		{name: "numeric id", subject: "orders.12345.created", want: "orders.*.created"},
		{name: "short numeric id", subject: "orders.7", want: "orders.*"},
		{name: "uuid", subject: "users.3f2b8c1e-9a4d-4c2b-8f1e-2d3c4b5a6f7e.events", want: "users.*.events"},
		{name: "names with digits kept", subject: "v1.eu2.ipv4.orders", want: "v1.eu2.ipv4.orders"},
		{name: "long id with digits", subject: "device.a1b2c3d4e5.state", want: "device.*.state"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, generalizeSubject(tt.subject))
		})
	}
}

func TestConfident(t *testing.T) {
	t.Parallel()
	user := entities.TypeCandidate{SourceID: "a", MessageType: "det.User", Score: 93}
	tests := []struct {
		name       string
		candidates []entities.TypeCandidate
		want       bool
	}{
		{name: "none", want: false},
		{name: "clear winner", candidates: []entities.TypeCandidate{user, {MessageType: "det.Note", Score: 44}}, want: true},
		{name: "unknown bytes", candidates: []entities.TypeCandidate{{MessageType: "det.User", Score: 93, UnknownBytes: 1}}, want: false},
		{name: "low score", candidates: []entities.TypeCandidate{{MessageType: "det.User", Score: autoDetectMinScore - 1}}, want: false},
		{name: "tie with another type", candidates: []entities.TypeCandidate{user, {MessageType: "det.Twin", Score: 93}}, want: false},
		{name: "same type in two sources", candidates: []entities.TypeCandidate{user, {SourceID: "b", MessageType: "det.User", Score: 93}}, want: true},
		{
			name: "tie behind a duplicate",
			candidates: []entities.TypeCandidate{
				user, {SourceID: "b", MessageType: "det.User", Score: 93}, {SourceID: "c", MessageType: "det.Twin", Score: 93},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, ok := confident(tt.candidates)
			assert.Equal(t, tt.want, ok)
		})
	}
}
