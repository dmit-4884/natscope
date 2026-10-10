// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package mcptransport

import (
	"context"
	"strings"

	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"

	connectionssvc "github.com/dmit-4884/natscope/internal/services/connections"
)

// Connections resolves the `connection` argument of NATS tools to a saved connection id.
type Connections struct {
	svc connectionssvc.Service
}

// NewConnections creates the connection resolver shared by all toolsets.
func NewConnections(svc connectionssvc.Service) *Connections {
	return &Connections{svc: svc}
}

// List returns every saved connection.
func (c *Connections) List(ctx context.Context) (entities.SavedConnections, error) {
	var all entities.SavedConnections
	req := &entities.SavedConnectionsList{ListBase: entities.ListBase{Limit: new(entities.MaxListLimit)}}
	for {
		page, err := c.svc.List(ctx, req)
		if err != nil {
			return nil, err
		}
		all = append(all, page.Items...)
		if page.NextCursor == nil || *page.NextCursor == "" {
			return all, nil
		}
		req.Cursor = *page.NextCursor
	}
}

// Resolve maps a connection id or name (case-insensitive) to its id; an empty ref selects the only saved connection.
func (c *Connections) Resolve(ctx context.Context, ref string) (string, error) {
	all, err := c.List(ctx)
	if err != nil {
		return "", err
	}
	if len(all) == 0 {
		return "", Errorf("no saved connections: add one in the natscope UI first")
	}

	ref = strings.TrimSpace(ref)
	if ref == "" {
		if len(all) == 1 {
			return all[0].Id, nil
		}
		return "", Errorf("several connections are saved (%s): pass `connection`", names(all))
	}
	var folded []*entities.SavedConnection
	for _, conn := range all {
		if conn.Id == ref || conn.Name == ref {
			return conn.Id, nil
		}
		if strings.EqualFold(conn.Name, ref) {
			folded = append(folded, conn)
		}
	}
	switch len(folded) {
	case 1:
		return folded[0].Id, nil
	case 0:
		return "", Errorf("connection %q not found; saved connections: %s", ref, names(all))
	default:
		return "", Errorf("connection %q matches several names (%s): pass the exact name or id", ref, names(folded))
	}
}

func names(all entities.SavedConnections) string {
	return strings.Join(slices.To(all, func(c *entities.SavedConnection) string { return c.Name }), ", ")
}
