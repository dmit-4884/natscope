// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "github.com/altessa-s/go-atlas/domain/converter"

// MessageSettings holds user preferences for message fetching. Concurrency
// now derives from runtime.NumCPU(), not a stored field.
type MessageSettings struct {
	FetchMethod      *string
	DefaultPageSize  *int32
	DefaultDirection *string

	// MaxPayloadBytesInList caps payload in list/live previews; nil = server
	// default (4 KB), 0 = unlimited; full payload via Get(sequence).
	MaxPayloadBytesInList *int32

	// DefaultExportFormat is "json"|"ndjson"|"csv" pre-selected in export dialog;
	// nil = "json".
	DefaultExportFormat *string

	// ExportRangeLimit caps full-range export size; client owns the ceiling.
	// Stored 0 normalizes to nil on Update since a range export needs a bound.
	ExportRangeLimit *int32
}

// DefaultMaxPayloadBytesInList server-side fallback, tuned small (4 KB) to
// keep list view snappy on multi-MB-message streams.
const DefaultMaxPayloadBytesInList int32 = 4 * 1024

// DefaultFetchMethod is the server-side fallback when neither the request nor
// the saved user settings pin one. "consumer" (ordered consumer with
// server-side filtering); the UI default mirrors this.
const DefaultFetchMethod = "consumer"

// LiveSettings holds user preferences for live subscriptions. SubscriptionMode
// picks Core NATS vs JetStream Ordered, for users lacking create_consumer.
type LiveSettings struct {
	SubscriptionMode *string
	MaxDisplayRate   *int32
}

// DisplaySettings holds user preferences for UI display.
type DisplaySettings struct {
	Density           *string
	DefaultViewMode   *string
	PayloadPreviewLen *int32
	TimestampFormat   *string
	JsonIndentSize    *int32
	AutoScrollLive    *bool
}

// PublishSettings holds user preferences for message publishing.
type PublishSettings struct {
	PublishTimeoutSec *int32
}

// BehaviorSettings holds confirm-toggles for destructive ops; Confirm* nil
// means ask (safe default). Irreversible-scale ops use type-to-confirm instead.
type BehaviorSettings struct {
	ConfirmDeleteConsumer *bool
	ConfirmDeleteMessage  *bool
	ConfirmDeleteKvKey    *bool
	ConfirmDeleteObject   *bool
	ConfirmPurgeKvHistory *bool
	SecureDeleteDefault   *bool
}

// UserSettings is the persisted per-user preference bag.
type UserSettings struct {
	BaseEntity
	Messages *MessageSettings
	Live     *LiveSettings
	Display  *DisplaySettings
	Publish  *PublishSettings
	Behavior *BehaviorSettings
}

// UserSettingsNew creates a new UserSettings with generated Id and timestamps.
func UserSettingsNew(init ...func(*UserSettings)) *UserSettings {
	s := &UserSettings{
		BaseEntity: *New(),
	}
	if len(init) > 0 && init[0] != nil {
		init[0](s)
	}
	return s
}

// ApplyUpdate applies non-nil fields from req.
func (s *UserSettings) ApplyUpdate(req *UserSettingsUpdate) {
	if s == nil || req == nil {
		return
	}
	// Nested structs: merge whole sub-struct if provided.
	if req.Messages != nil {
		if s.Messages == nil {
			s.Messages = &MessageSettings{}
		}
		converter.Convert(req.Messages, s.Messages, converter.WithIgnoreNilValues())
	}
	if req.Live != nil {
		if s.Live == nil {
			s.Live = &LiveSettings{}
		}
		converter.Convert(req.Live, s.Live, converter.WithIgnoreNilValues())
	}
	if req.Display != nil {
		if s.Display == nil {
			s.Display = &DisplaySettings{}
		}
		converter.Convert(req.Display, s.Display, converter.WithIgnoreNilValues())
	}
	if req.Publish != nil {
		if s.Publish == nil {
			s.Publish = &PublishSettings{}
		}
		converter.Convert(req.Publish, s.Publish, converter.WithIgnoreNilValues())
	}
	if req.Behavior != nil {
		if s.Behavior == nil {
			s.Behavior = &BehaviorSettings{}
		}
		converter.Convert(req.Behavior, s.Behavior, converter.WithIgnoreNilValues())
	}
	s.BeforeUpdate()
}

// UserSettingsUpdate is the DTO for partial updates of user settings.
type UserSettingsUpdate struct {
	Messages *MessageSettings
	Live     *LiveSettings
	Display  *DisplaySettings
	Publish  *PublishSettings
	Behavior *BehaviorSettings
}
