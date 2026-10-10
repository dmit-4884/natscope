// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/bbstore"

	settingsbbolt "github.com/dmit-4884/natscope/internal/storages/settings/bbolt"
)

func newDB(t *testing.T) *bbstore.DB {
	t.Helper()
	db, err := bbstore.NewDB(filepath.Join(t.TempDir(), "test.bolt"))
	if err != nil {
		t.Fatalf("new db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestSettings_AllGroupsRoundTrip exercises every field so a dropped value
// surfaces here.
func TestSettings_AllGroupsRoundTrip(t *testing.T) {
	s, err := settingsbbolt.New(t.Context(), newDB(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := t.Context()

	in := entities.UserSettingsNew(func(u *entities.UserSettings) {
		u.Messages = &entities.MessageSettings{
			FetchMethod: new("batch"), DefaultPageSize: new(int32(50)),
			DefaultDirection: new("forward"), MaxPayloadBytesInList: new(int32(4096)),
			DefaultExportFormat: new("ndjson"), ExportRangeLimit: new(int32(1000)), DetectTypes: new(false),
		}
		u.Live = &entities.LiveSettings{SubscriptionMode: new("ordered"), MaxDisplayRate: new(int32(30))}
		u.Display = &entities.DisplaySettings{
			Density: new("compact"), DefaultViewMode: new("json"), PayloadPreviewLen: new(int32(256)),
			TimestampFormat: new("iso"), JsonIndentSize: new(int32(2)), AutoScrollLive: new(true),
		}
		u.Publish = &entities.PublishSettings{PublishTimeoutSec: new(int32(15))}
		u.Behavior = &entities.BehaviorSettings{
			ConfirmDeleteConsumer: new(true), ConfirmDeleteMessage: new(false),
			ConfirmDeleteKvKey: new(true), ConfirmDeleteObject: new(false),
			ConfirmPurgeKvHistory: new(true), SecureDeleteDefault: new(false),
		}
	})
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	m := got.Messages
	if m == nil || *m.FetchMethod != "batch" || *m.DefaultPageSize != 50 || *m.DefaultDirection != "forward" ||
		*m.MaxPayloadBytesInList != 4096 || *m.DefaultExportFormat != "ndjson" || *m.ExportRangeLimit != 1000 || m.DetectsTypes() {
		t.Fatalf("messages mismatch: %+v", m)
	}
	if got.Live == nil || *got.Live.SubscriptionMode != "ordered" || *got.Live.MaxDisplayRate != 30 {
		t.Fatalf("live mismatch: %+v", got.Live)
	}
	d := got.Display
	if d == nil || *d.Density != "compact" || *d.DefaultViewMode != "json" || *d.PayloadPreviewLen != 256 ||
		*d.TimestampFormat != "iso" || *d.JsonIndentSize != 2 || !*d.AutoScrollLive {
		t.Fatalf("display mismatch: %+v", d)
	}
	if got.Publish == nil || *got.Publish.PublishTimeoutSec != 15 {
		t.Fatalf("publish mismatch: %+v", got.Publish)
	}
	b := got.Behavior
	if b == nil || !*b.ConfirmDeleteConsumer || *b.ConfirmDeleteMessage || !*b.ConfirmDeleteKvKey ||
		*b.ConfirmDeleteObject || !*b.ConfirmPurgeKvHistory || *b.SecureDeleteDefault {
		t.Fatalf("behavior mismatch: %+v", b)
	}
}

// TestSettings_EmptyGroupsStayNil verifies an unset group reconstructs as nil.
func TestSettings_EmptyGroupsStayNil(t *testing.T) {
	s, _ := settingsbbolt.New(t.Context(), newDB(t))
	ctx := t.Context()

	in := entities.UserSettingsNew(func(u *entities.UserSettings) {
		u.Display = &entities.DisplaySettings{Density: new("comfortable")}
	})
	if err := s.Save(ctx, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := s.Get(ctx)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Messages != nil || got.Live != nil || got.Publish != nil || got.Behavior != nil {
		t.Fatalf("expected nil groups, got messages=%v live=%v publish=%v behavior=%v",
			got.Messages, got.Live, got.Publish, got.Behavior)
	}
	if got.Display == nil || *got.Display.Density != "comfortable" {
		t.Fatalf("display mismatch: %+v", got.Display)
	}
}

// TestSettings_UpdateConcurrentPartialUpdatesDoNotLoseWrites checks that concurrent group updates all persist.
func TestSettings_UpdateConcurrentPartialUpdatesDoNotLoseWrites(t *testing.T) {
	s, err := settingsbbolt.New(t.Context(), newDB(t))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	ctx := t.Context()

	const rounds = 20
	for r := 1; r <= rounds; r++ {
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			_, _ = s.Update(ctx, func(existing *entities.UserSettings) {
				if existing.Messages == nil {
					existing.Messages = &entities.MessageSettings{}
				}
				existing.Messages.DefaultPageSize = new(int32(r))
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = s.Update(ctx, func(existing *entities.UserSettings) {
				if existing.Publish == nil {
					existing.Publish = &entities.PublishSettings{}
				}
				existing.Publish.PublishTimeoutSec = new(int32(r))
			})
		}()
		go func() {
			defer wg.Done()
			_, _ = s.Update(ctx, func(existing *entities.UserSettings) {
				if existing.Live == nil {
					existing.Live = &entities.LiveSettings{}
				}
				existing.Live.MaxDisplayRate = new(int32(r))
			})
		}()
		wg.Wait()

		got, err := s.Get(ctx)
		if err != nil {
			t.Fatalf("round %d get: %v", r, err)
		}
		if got.Messages == nil || got.Messages.DefaultPageSize == nil || *got.Messages.DefaultPageSize != int32(r) {
			t.Fatalf("round %d: messages.defaultPageSize lost, got %+v", r, got.Messages)
		}
		if got.Publish == nil || got.Publish.PublishTimeoutSec == nil || *got.Publish.PublishTimeoutSec != int32(r) {
			t.Fatalf("round %d: publish.publishTimeoutSec lost, got %+v", r, got.Publish)
		}
		if got.Live == nil || got.Live.MaxDisplayRate == nil || *got.Live.MaxDisplayRate != int32(r) {
			t.Fatalf("round %d: live.maxDisplayRate lost, got %+v", r, got.Live)
		}
	}
}
