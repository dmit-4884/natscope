// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sections

import (
	"context"
	"encoding/json"

	"github.com/altessa-s/go-atlas/core/collections/maps"
	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	connectionssvc "github.com/dmit-4884/natscope/internal/services/connections"
)

// connectionItem is the NON-SECRET projection of a saved connection. Redaction
// is structural: the converter can only copy fields this shape declares, so a
// secret added to the entity later cannot silently start being exported.
type connectionItem struct {
	Name        string                     `json:"name"`
	Description *string                    `json:"description,omitempty"`
	URLs        []string                   `json:"urls"`
	Auth        *connAuthItem              `json:"auth,omitempty"`
	TLS         *connTLSItem               `json:"tls,omitempty"`
	Connection  *entities.ConnectionConfig `json:"connection,omitempty"`
	Reconnect   *entities.ReconnectConfig  `json:"reconnect,omitempty"`
	Ping        *entities.PingConfig       `json:"ping,omitempty"`
}

type connAuthItem struct {
	Method   entities.AuthMethod `json:"method"`
	Username *string             `json:"username,omitempty"`
}

type connTLSItem struct {
	CaCert     *string `json:"caCert,omitempty"`
	SkipVerify bool    `json:"skipVerify,omitempty"`
	TlsFirst   bool    `json:"tlsFirst,omitempty"`
}

// ConnectionsSection exports/imports saved connections WITHOUT credentials;
// merge keeps existing names (local secrets intact), creates new ones.
type ConnectionsSection struct {
	svc connectionssvc.Service
}

// NewConnectionsSection constructs the connections workspace section.
func NewConnectionsSection(svc connectionssvc.Service) *ConnectionsSection {
	return &ConnectionsSection{svc: svc}
}

func (s *ConnectionsSection) Key() string { return "connections" }

func (s *ConnectionsSection) Describe(ctx context.Context) (entities.WorkspaceSectionInfo, error) {
	all, err := s.all(ctx)
	if err != nil {
		return entities.WorkspaceSectionInfo{}, err
	}
	return entities.WorkspaceSectionInfo{Key: s.Key(), Title: "Connections (no credentials)", Count: int32(len(all))}, nil
}

func (s *ConnectionsSection) Export(ctx context.Context) (json.RawMessage, error) {
	all, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	items := slices.To(all, redactConnection)
	return json.Marshal(newItemsPayload(items))
}

func (s *ConnectionsSection) Validate(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionReport, error) {
	items, err := decodeItems[connectionItem](raw)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	existing, err := s.existingNames(ctx)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	created, deleted, conflicts := createOnlyPlan(
		existing,
		slices.To(items, func(i connectionItem) string { return i.Name }),
		strategy,
	)
	rep := entities.WorkspaceSectionReport{Created: created, Deleted: deleted, Conflicts: conflicts}
	if created > 0 {
		rep.Warnings = append(rep.Warnings, "imported connections have no credentials — set them after import")
	}
	if deleted > 0 {
		rep.Warnings = append(rep.Warnings,
			"REPLACE will permanently delete the existing saved connections AND their stored credentials")
	}
	return rep, nil
}

func (s *ConnectionsSection) Import(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionResult, error) {
	items, err := decodeItems[connectionItem](raw)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}
	all, err := s.all(ctx)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}

	var res entities.WorkspaceSectionResult
	if strategy == entities.WorkspaceStrategyReplace {
		// Delete before create: the active-name unique index rejects a create that
		// collides with a not-yet-deleted record.
		for _, c := range all {
			if dErr := s.svc.Delete(ctx, c.Id); dErr != nil {
				return res, dErr
			}
			res.Deleted++
		}
		for _, it := range items {
			if _, cErr := s.svc.Create(ctx, converter.Convert(it, &entities.SavedConnectionCreate{})); cErr != nil {
				return res, cErr
			}
			res.Created++
		}
		if res.Created > 0 {
			res.Warnings = append(res.Warnings, "imported connections have no credentials — set them before connecting")
		}
		return res, nil
	}

	// Merge: keep existing names (their local secrets stay intact), create new.
	existing := maps.FromSliceWith(all, func(c *entities.SavedConnection) (string, struct{}) {
		return c.Name, struct{}{}
	})
	for _, it := range items {
		if _, ok := existing[it.Name]; ok {
			continue // keep the local one (with its credentials)
		}
		if _, cErr := s.svc.Create(ctx, converter.Convert(it, &entities.SavedConnectionCreate{})); cErr != nil {
			return res, cErr
		}
		res.Created++
	}
	if res.Created > 0 {
		res.Warnings = append(res.Warnings, "imported connections have no credentials — set them before connecting")
	}
	return res, nil
}

// redactConnection projects only the non-secret fields of a saved connection.
func redactConnection(c *entities.SavedConnection) connectionItem {
	item := converter.Convert(c, &connectionItem{})
	item.URLs = natsutil.StripCredentials(c.URLs)
	return *item
}

func (s *ConnectionsSection) all(ctx context.Context) (entities.SavedConnections, error) {
	res, err := s.svc.List(ctx, &entities.SavedConnectionsList{ListBase: entities.ListBase{Limit: ptr.Wrap(listAllLimit)}})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func (s *ConnectionsSection) existingNames(ctx context.Context) (map[string]struct{}, error) {
	all, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	return maps.FromSliceWith(all, func(c *entities.SavedConnection) (string, struct{}) {
		return c.Name, struct{}{}
	}), nil
}
