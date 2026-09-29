// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"

	mappingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/mappings/v1/mappings"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	settingspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/settings/v1/settings"
	templatespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/templates/v1/templates"
	settingstypes "github.com/dmit-4884/natscope/proto/gen/types/settings"
)

// TestPagination creates more items than one page and walks next_page_token
// to exhaustion, asserting no duplicates/gaps and total_size matches created count.
func TestPagination(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	const total = 25
	const pageSize = 10

	t.Run("mappings", func(t *testing.T) {
		for i := 0; i < total; i++ {
			_, err := env.mappings.CreateMapping(ctx, connect.NewRequest(&mappingspb.CreateMappingRequest{
				Pattern:     fmt.Sprintf("page.test.%03d", i),
				MessageType: "x.Y",
				SourceId:    "page-source",
			}))
			require.NoError(t, err)
		}

		seen := map[string]bool{}
		token := ""
		pages := 0
		var reportedTotal int64
		for {
			resp, err := env.mappings.ListMappings(ctx, connect.NewRequest(&mappingspb.ListMappingsRequest{
				PageSize: pageSize, PageToken: token, IncludeTotalCount: true,
			}))
			require.NoError(t, err)
			pages++
			require.LessOrEqual(t, pages, total, "must not loop forever")
			for _, m := range resp.Msg.GetMappings() {
				assert.False(t, seen[m.GetId()], "item %s seen twice across pages", m.GetId())
				seen[m.GetId()] = true
			}
			reportedTotal = resp.Msg.GetTotalSize()
			next := resp.Msg.GetNextPageToken()
			if next == "" {
				break
			}
			token = next
		}
		// setupE2E boots a fresh app+storage per test, so all `total` mappings
		// created here are the only ones — the count below is exact.
		assert.GreaterOrEqual(t, len(seen), total)
		assert.EqualValues(t, len(seen), reportedTotal, "total_size must match the number of distinct items paged through")
	})

	t.Run("templates", func(t *testing.T) {
		for i := 0; i < total; i++ {
			_, err := env.templates.CreateTemplate(ctx, connect.NewRequest(&templatespb.CreateTemplateRequest{
				Name: fmt.Sprintf("page-template-%03d", i), Subject: "x", Data: "{}",
			}))
			require.NoError(t, err)
		}

		seen := map[string]bool{}
		token := ""
		pages := 0
		var reportedTotal int64
		for {
			resp, err := env.templates.ListTemplates(ctx, connect.NewRequest(&templatespb.ListTemplatesRequest{
				PageSize: pageSize, PageToken: token, IncludeTotalCount: true,
			}))
			require.NoError(t, err)
			pages++
			require.LessOrEqual(t, pages, total, "must not loop forever")
			for _, tmpl := range resp.Msg.GetTemplates() {
				assert.False(t, seen[tmpl.GetId()], "item %s seen twice across pages", tmpl.GetId())
				seen[tmpl.GetId()] = true
			}
			reportedTotal = resp.Msg.GetTotalSize()
			next := resp.Msg.GetNextPageToken()
			if next == "" {
				break
			}
			token = next
		}
		assert.GreaterOrEqual(t, len(seen), total)
		assert.EqualValues(t, len(seen), reportedTotal)
	})

	t.Run("connections (no total_size field, just exhaustion)", func(t *testing.T) {
		for i := 0; i < total; i++ {
			_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
				Name: fmt.Sprintf("page-conn-%03d", i), Urls: []string{"nats://127.0.0.1:1"},
			}))
			require.NoError(t, err)
		}

		seen := map[string]bool{}
		token := ""
		pages := 0
		for {
			resp, err := env.connections.ListConnections(ctx, connect.NewRequest(&connectionspb.ListConnectionsRequest{
				PageSize: pageSize, PageToken: token,
			}))
			require.NoError(t, err)
			pages++
			require.LessOrEqual(t, pages, total, "must not loop forever")
			for _, c := range resp.Msg.GetConnections() {
				assert.False(t, seen[c.GetId()], "item %s seen twice across pages", c.GetId())
				seen[c.GetId()] = true
			}
			next := resp.Msg.GetNextPageToken()
			if next == "" {
				break
			}
			token = next
		}
		assert.GreaterOrEqual(t, len(seen), total)
	})
}

func setMessagesFetchMethod(t *testing.T, env *e2eEnv, method string) {
	t.Helper()
	_, err := env.settings.UpdateSettings(t.Context(), connect.NewRequest(&settingspb.UpdateSettingsRequest{
		Messages: &settingstypes.MessageSettings{FetchMethod: strPtr(method)},
	}))
	require.NoError(t, err)
}

// walkMessages pages ListMessages to the end and returns every sequence seen, failing on duplicates.
func walkMessages(
	t *testing.T,
	env *e2eEnv,
	connID, stream string,
	direction messagespb.Direction,
	subjectFilter string,
	limit int64,
) []uint64 {
	t.Helper()
	ctx := t.Context()

	const maxPages = 500
	var seqs []uint64
	seen := map[uint64]bool{}
	var startSeq *uint64

	for page := 0; page < maxPages; page++ {
		req := &messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream,
			Direction: direction, Limit: &limit, StartSeq: startSeq,
		}
		if subjectFilter != "" {
			req.SubjectFilter = &subjectFilter
		}
		resp, err := env.messages.ListMessages(ctx, connect.NewRequest(req))
		require.NoError(t, err)

		for _, m := range resp.Msg.GetMessages() {
			seq := m.GetSequence()
			require.False(t, seen[seq], "sequence %d returned twice while paging", seq)
			seen[seq] = true
			seqs = append(seqs, seq)
		}
		if !resp.Msg.GetHasMore() {
			return seqs
		}
		next := resp.Msg.GetNextSeq()
		startSeq = &next
	}

	t.Fatalf("pagination did not terminate within %d pages", maxPages)
	return nil
}

// TestMessagesPaginationSurvivesDeletionGap checks that a backward walk crosses a wide deletion gap.
func TestMessagesPaginationSurvivesDeletionGap(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "pagination-gap", env.natsURL, nil)

	const stream = "GAPPED"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"gapped.>"},
	}))
	require.NoError(t, err)

	const total = 150
	for i := 0; i < total; i++ {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "gapped.a", Data: "{}",
		}))
		require.NoError(t, err)
	}

	const gapStart, gapEnd = 21, 120
	for seq := gapStart; seq <= gapEnd; seq++ {
		_, err := env.management.DeleteMessage(ctx, connect.NewRequest(&managementpb.DeleteMessageRequest{
			ConnectionId: connID, StreamName: stream, Sequence: uint64(seq),
		}))
		require.NoError(t, err)
	}

	wantLive := map[uint64]bool{}
	for seq := 1; seq <= total; seq++ {
		if seq < gapStart || seq > gapEnd {
			wantLive[uint64(seq)] = true
		}
	}

	for _, method := range []string{"consumer", "direct"} {
		t.Run(method, func(t *testing.T) {
			setMessagesFetchMethod(t, env, method)

			seqs := walkMessages(t, env, connID, stream, messagespb.Direction_DIRECTION_BACKWARD, "", 5)
			got := map[uint64]bool{}
			for _, s := range seqs {
				got[s] = true
			}
			assert.Equal(t, wantLive, got, "backward pagination must reach every live message across the deletion gap")
		})
	}
}

// TestMessagesPaginationSubjectFilterFindsOldMatches checks that a filter finds matches older than the first window.
func TestMessagesPaginationSubjectFilterFindsOldMatches(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "pagination-subject-filter", env.natsURL, nil)

	const stream = "SPARSE"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"sparse.>"},
	}))
	require.NoError(t, err)

	publish := func(subject string) uint64 {
		resp, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: subject, Data: "{}",
		}))
		require.NoError(t, err)
		return resp.Msg.GetSequence()
	}

	rare1 := publish("sparse.rare")
	const filler = 120
	for i := 0; i < filler; i++ {
		publish("sparse.a")
	}
	rare2 := publish("sparse.rare")
	for i := 0; i < filler; i++ {
		publish("sparse.a")
	}
	rare3 := publish("sparse.rare")

	for _, method := range []string{"consumer", "direct"} {
		t.Run(method, func(t *testing.T) {
			setMessagesFetchMethod(t, env, method)

			seqs := walkMessages(t, env, connID, stream, messagespb.Direction_DIRECTION_BACKWARD, "sparse.rare", 5)
			assert.ElementsMatch(t, []uint64{rare1, rare2, rare3}, seqs,
				"a subject filter must find matches beyond the first browse window, not just the newest one")
		})
	}
}

// TestMessagesPaginationDirectModeBoundaries covers direct-mode page-size, post-purge and filter-direction edges.
func TestMessagesPaginationDirectModeBoundaries(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "pagination-direct-boundaries", env.natsURL, nil)
	setMessagesFetchMethod(t, env, "direct")

	t.Run("off-by-one on the last page", func(t *testing.T) {
		const stream = "OB"
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"ob.>"},
		}))
		require.NoError(t, err)
		for i := 0; i < 51; i++ {
			_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
				ConnectionId: connID, Subject: "ob.a", Data: "{}",
			}))
			require.NoError(t, err)
		}

		limit := int64(50)
		resp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream, Direction: messagespb.Direction_DIRECTION_FORWARD,
			StartSeq: uint64Ptr(1), Limit: &limit,
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.GetMessages(), 50, "a stream of exactly limit+1 messages must not drop the last one")
		assert.True(t, resp.Msg.GetHasMore())
		assert.EqualValues(t, 51, resp.Msg.GetNextSeq())

		next, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream, Direction: messagespb.Direction_DIRECTION_FORWARD,
			StartSeq: uint64Ptr(51), Limit: &limit,
		}))
		require.NoError(t, err)
		require.Len(t, next.Msg.GetMessages(), 1)
		assert.EqualValues(t, 51, next.Msg.GetMessages()[0].GetSequence())
		assert.False(t, next.Msg.GetHasMore())
	})

	t.Run("purge gap does not loop back to sequence one", func(t *testing.T) {
		const stream = "DG"
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"dg.>"},
		}))
		require.NoError(t, err)
		for i := 0; i < 200; i++ {
			_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
				ConnectionId: connID, Subject: "dg.a", Data: "{}",
			}))
			require.NoError(t, err)
		}
		_, err = env.management.PurgeStream(ctx, connect.NewRequest(&managementpb.PurgeStreamRequest{
			ConnectionId: connID, StreamName: stream, Sequence: uint64Ptr(120),
		}))
		require.NoError(t, err)

		limit := int64(50)
		resp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream, Direction: messagespb.Direction_DIRECTION_FORWARD,
			StartSeq: uint64Ptr(1), Limit: &limit,
		}))
		require.NoError(t, err)
		require.NotEmpty(t, resp.Msg.GetMessages(), "a page starting before FirstSeq must not come back empty and loop to sequence 1")
		assert.EqualValues(t, 120, resp.Msg.GetMessages()[0].GetSequence())
		assert.True(t, resp.Msg.GetHasMore())
		assert.NotEqualValues(t, 1, resp.Msg.GetNextSeq(), "must not rewind to the start of the stream")

		seqs := walkMessages(t, env, connID, stream, messagespb.Direction_DIRECTION_FORWARD, "", 50)
		assert.Len(t, seqs, 81, "walking forward from before FirstSeq must reach every live message exactly once")
	})

	t.Run("wildcard filter without a start seq begins at the stream boundary", func(t *testing.T) {
		const stream = "WF"
		_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
			ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"wf.>"},
		}))
		require.NoError(t, err)
		for i := 0; i < 20; i++ {
			_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
				ConnectionId: connID, Subject: "wf.a", Data: "{}",
			}))
			require.NoError(t, err)
		}
		_, err = env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "wf.b", Data: "{}",
		}))
		require.NoError(t, err)

		limit := int64(10)
		resp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream, Direction: messagespb.Direction_DIRECTION_FORWARD,
			SubjectFilter: strPtr("wf.*"), Limit: &limit,
		}))
		require.NoError(t, err)
		require.Len(t, resp.Msg.GetMessages(), 10, "forward+wildcard without a start seq must start at the oldest message")
		assert.EqualValues(t, 1, resp.Msg.GetMessages()[0].GetSequence())
		assert.True(t, resp.Msg.GetHasMore())

		limitExact := int64(3)
		exact, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream, Direction: messagespb.Direction_DIRECTION_FORWARD,
			SubjectFilter: strPtr("wf.a"), Limit: &limitExact,
		}))
		require.NoError(t, err)
		require.Len(t, exact.Msg.GetMessages(), 3)
		var exactSeqs []uint64
		for _, m := range exact.Msg.GetMessages() {
			exactSeqs = append(exactSeqs, m.GetSequence())
		}
		assert.Equal(t, []uint64{1, 2, 3}, exactSeqs, "an exact-subject filter must honor direction, not always return newest-first")
	})
}

// TestMessagesPaginationStartTimeFarFuture checks that a start time past the last message returns nothing.
func TestMessagesPaginationStartTimeFarFuture(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "start-time-future", env.natsURL, nil)

	const stream = "FT"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"ft.>"},
	}))
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
			ConnectionId: connID, Subject: "ft.a", Data: "{}",
		}))
		require.NoError(t, err)
	}

	farFuture := timestamppb.New(time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC))
	limit := int64(3)

	for _, method := range []string{"consumer", "direct"} {
		t.Run(method, func(t *testing.T) {
			setMessagesFetchMethod(t, env, method)

			resp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
				ConnectionId: connID, StreamName: stream, Direction: messagespb.Direction_DIRECTION_FORWARD,
				StartTime: farFuture, Limit: &limit,
			}))
			require.NoError(t, err)
			assert.Empty(t, resp.Msg.GetMessages(), "a start time after the last message must return nothing, not jump to the beginning")
		})
	}
}

// TestMessagesPaginationEmptyStream checks that an empty stream returns an empty page, not an error.
func TestMessagesPaginationEmptyStream(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "empty-stream", env.natsURL, nil)

	const stream = "EMPTY"
	_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
		ConnectionId: connID, Name: stream, Storage: 1, Subjects: []string{"empty.>"},
	}))
	require.NoError(t, err)

	t.Run("consumer mode default direction", func(t *testing.T) {
		setMessagesFetchMethod(t, env, "consumer")
		resp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream, Direction: messagespb.Direction_DIRECTION_FORWARD,
		}))
		require.NoError(t, err, "a never-written stream must return an empty page, not a delivery-policy error")
		assert.Empty(t, resp.Msg.GetMessages())
	})

	t.Run("direct mode with a wildcard filter", func(t *testing.T) {
		setMessagesFetchMethod(t, env, "direct")
		resp, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
			ConnectionId: connID, StreamName: stream, SubjectFilter: strPtr("empty.*"),
		}))
		require.NoError(t, err, "a never-written stream must return an empty page, not a NATS bad-request error")
		assert.Empty(t, resp.Msg.GetMessages())
	})
}
