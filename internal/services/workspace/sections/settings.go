// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sections

import (
	"context"
	"encoding/json"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
)

// settingsPayload carries the shareable settings sections (no secrets —
// UI/behavior toggles only).
type settingsPayload struct {
	Version  int                          `json:"version"`
	Settings *entities.UserSettingsUpdate `json:"settings"`
}

// SettingsSection exports/imports the singleton UserSettings.
type SettingsSection struct {
	svc settingssvc.Service
}

// NewSettingsSection constructs the settings workspace section.
func NewSettingsSection(svc settingssvc.Service) *SettingsSection {
	return &SettingsSection{svc: svc}
}

func (s *SettingsSection) Key() string { return "settings" }

func (s *SettingsSection) Describe(ctx context.Context) (entities.WorkspaceSectionInfo, error) {
	// Settings is a singleton; count is 1 when any section is configured.
	cur, err := s.svc.Get(ctx)
	if err != nil {
		return entities.WorkspaceSectionInfo{}, err
	}
	hasSettings := cur != nil && (cur.Messages != nil || cur.Live != nil ||
		cur.Display != nil || cur.Publish != nil || cur.Behavior != nil)
	count := int32(0)
	if hasSettings {
		count = 1
	}
	return entities.WorkspaceSectionInfo{Key: s.Key(), Title: "User Settings", Count: count}, nil
}

func (s *SettingsSection) Export(ctx context.Context) (json.RawMessage, error) {
	cur, err := s.svc.Get(ctx)
	if err != nil {
		return nil, err
	}
	upd := &entities.UserSettingsUpdate{}
	if cur != nil {
		upd = converter.Convert(cur, &entities.UserSettingsUpdate{})
	}
	return json.Marshal(settingsPayload{Version: sectionVersion, Settings: upd})
}

func (s *SettingsSection) Validate(
	_ context.Context,
	raw json.RawMessage,
	_ entities.WorkspaceStrategy,
) (entities.WorkspaceSectionReport, error) {
	var p settingsPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return entities.WorkspaceSectionReport{}, errs.ErrWorkspaceSectionInvalid
	}
	if p.Version > sectionVersion {
		return entities.WorkspaceSectionReport{}, errs.ErrWorkspaceSectionInvalid
	}
	// Singleton: import (re)writes the one settings object only when the file
	// carries settings; empty payload changes nothing.
	if p.Settings == nil {
		return entities.WorkspaceSectionReport{}, nil
	}
	return entities.WorkspaceSectionReport{Updated: 1}, nil
}

func (s *SettingsSection) Import(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionResult, error) {
	var p settingsPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return entities.WorkspaceSectionResult{}, errs.ErrWorkspaceSectionInvalid
	}
	if p.Version > sectionVersion {
		return entities.WorkspaceSectionResult{}, errs.ErrWorkspaceSectionInvalid
	}
	// Replace resets to defaults first so omitted sections are cleared.
	if strategy == entities.WorkspaceStrategyReplace {
		if _, err := s.svc.Reset(ctx); err != nil {
			return entities.WorkspaceSectionResult{}, err
		}
	}
	if p.Settings != nil {
		if _, err := s.svc.Update(ctx, p.Settings); err != nil {
			return entities.WorkspaceSectionResult{}, err
		}
		return entities.WorkspaceSectionResult{Updated: 1}, nil
	}
	return entities.WorkspaceSectionResult{}, nil
}
