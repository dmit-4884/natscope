// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbolt

import "github.com/dmit-4884/natscope/internal/pkg/bbstore"

// settingsDoc is the persistence model for the single user-settings document.
// Each optional group is a nested object, present only when set.
type settingsDoc struct {
	bbstore.Base

	Messages *messagesDoc `json:"messages,omitempty"`
	Live     *liveDoc     `json:"live,omitempty"`
	Display  *displayDoc  `json:"display,omitempty"`
	Publish  *publishDoc  `json:"publish,omitempty"`
	Behavior *behaviorDoc `json:"behavior,omitempty"`
}

type messagesDoc struct {
	FetchMethod           *string `json:"fetchMethod,omitempty"`
	DefaultPageSize       *int32  `json:"defaultPageSize,omitempty"`
	DefaultDirection      *string `json:"defaultDirection,omitempty"`
	MaxPayloadBytesInList *int32  `json:"maxPayloadBytesInList,omitempty"`
	DefaultExportFormat   *string `json:"defaultExportFormat,omitempty"`
	ExportRangeLimit      *int32  `json:"exportRangeLimit,omitempty"`
}

type liveDoc struct {
	SubscriptionMode *string `json:"subscriptionMode,omitempty"`
	MaxDisplayRate   *int32  `json:"maxDisplayRate,omitempty"`
}

type displayDoc struct {
	Density           *string `json:"density,omitempty"`
	DefaultViewMode   *string `json:"defaultViewMode,omitempty"`
	PayloadPreviewLen *int32  `json:"payloadPreviewLen,omitempty"`
	TimestampFormat   *string `json:"timestampFormat,omitempty"`
	JsonIndentSize    *int32  `json:"jsonIndentSize,omitempty"`
	AutoScrollLive    *bool   `json:"autoScrollLive,omitempty"`
}

type publishDoc struct {
	PublishTimeoutSec *int32 `json:"publishTimeoutSec,omitempty"`
}

type behaviorDoc struct {
	ConfirmDeleteConsumer *bool `json:"confirmDeleteConsumer,omitempty"`
	ConfirmDeleteMessage  *bool `json:"confirmDeleteMessage,omitempty"`
	ConfirmDeleteKvKey    *bool `json:"confirmDeleteKvKey,omitempty"`
	ConfirmDeleteObject   *bool `json:"confirmDeleteObject,omitempty"`
	ConfirmPurgeKvHistory *bool `json:"confirmPurgeKvHistory,omitempty"`
	SecureDeleteDefault   *bool `json:"secureDeleteDefault,omitempty"`
}
