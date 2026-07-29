// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt_test

import (
	"path/filepath"
	"testing"

	"github.com/altessa-s/go-atlas/core/types/ptr"

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
			FetchMethod: ptr.Wrap("batch"), DefaultPageSize: ptr.Wrap(int32(50)),
			DefaultDirection: ptr.Wrap("forward"), MaxPayloadBytesInList: ptr.Wrap(int32(4096)),
			DefaultExportFormat: ptr.Wrap("ndjson"), ExportRangeLimit: ptr.Wrap(int32(1000)),
		}
		u.Live = &entities.LiveSettings{SubscriptionMode: ptr.Wrap("ordered"), MaxDisplayRate: ptr.Wrap(int32(30))}
		u.Display = &entities.DisplaySettings{
			Density: ptr.Wrap("compact"), DefaultViewMode: ptr.Wrap("json"), PayloadPreviewLen: ptr.Wrap(int32(256)),
			TimestampFormat: ptr.Wrap("iso"), JsonIndentSize: ptr.Wrap(int32(2)), AutoScrollLive: ptr.Wrap(true),
		}
		u.Publish = &entities.PublishSettings{PublishTimeoutSec: ptr.Wrap(int32(15))}
		u.Behavior = &entities.BehaviorSettings{
			ConfirmDeleteConsumer: ptr.Wrap(true), ConfirmDeleteMessage: ptr.Wrap(false),
			ConfirmDeleteKvKey: ptr.Wrap(true), ConfirmDeleteObject: ptr.Wrap(false),
			ConfirmPurgeKvHistory: ptr.Wrap(true), SecureDeleteDefault: ptr.Wrap(false),
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
		*m.MaxPayloadBytesInList != 4096 || *m.DefaultExportFormat != "ndjson" || *m.ExportRangeLimit != 1000 {
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
		u.Display = &entities.DisplaySettings{Density: ptr.Wrap("comfortable")}
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
