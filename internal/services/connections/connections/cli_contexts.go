// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"errors"
	"log/slog"

	"github.com/altessa-s/go-atlas/core/collections/maps"
	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natscontext"
)

const allConnectionsLimit = 10000

// ListCliContexts reads the nats CLI contexts on this host, or the uploaded files when any are given.
func (s *Service) ListCliContexts(ctx context.Context, files []entities.CliContextFile) (*entities.CliContexts, error) {
	found, err := s.cliContexts(files)
	if err != nil {
		return nil, err
	}
	names, err := s.savedNames(ctx)
	if err != nil {
		return nil, err
	}
	for i := range found.Contexts {
		_, found.Contexts[i].Exists = names[found.Contexts[i].Name]
	}
	return found, nil
}

// ImportCliContexts creates a connection per named context; a missing, unusable or already saved one is skipped.
func (s *Service) ImportCliContexts(
	ctx context.Context,
	names []string,
	files []entities.CliContextFile,
) (*entities.CliContextImport, error) {
	found, err := s.cliContexts(files)
	if err != nil {
		return nil, err
	}
	byName := maps.FromSliceWith(found.Contexts, func(c entities.CliContext) (string, entities.CliContext) {
		return c.Name, c
	})

	res := &entities.CliContextImport{}
	skip := func(name, reason string) {
		res.Skipped = append(res.Skipped, entities.CliContextSkip{Name: name, Reason: reason})
	}
	for _, name := range names {
		c, ok := byName[name]
		if !ok {
			skip(name, "no such context")
			continue
		}
		if c.Connection == nil {
			skip(name, "not a nats CLI context")
			continue
		}
		conn, createErr := s.Create(ctx, c.Connection)
		switch {
		case createErr == nil:
			res.Created = append(res.Created, conn)
		case errors.Is(createErr, errs.ErrConnectionNameAlreadyInUse):
			skip(name, "a connection with this name already exists")
		case isInvalidConnection(createErr):
			skip(name, createErr.Error())
		default:
			return nil, createErr
		}
	}

	s.logger.InfoContext(ctx, "nats CLI contexts imported",
		slog.Int("created", len(res.Created)),
		slog.Int("skipped", len(res.Skipped)))
	return res, nil
}

func (s *Service) cliContexts(files []entities.CliContextFile) (*entities.CliContexts, error) {
	if len(files) > 0 {
		return &entities.CliContexts{Contexts: natscontext.Parse(files)}, nil
	}
	dir, err := natscontext.Dir()
	if err != nil {
		return nil, err
	}
	contexts, err := natscontext.Read(dir)
	if err != nil {
		return nil, err
	}
	return &entities.CliContexts{Dir: dir, Contexts: contexts}, nil
}

func (s *Service) savedNames(ctx context.Context) (map[string]struct{}, error) {
	list, err := s.storage.List(ctx, &entities.SavedConnectionsList{ListBase: entities.ListBase{Limit: ptr.Wrap(int64(allConnectionsLimit))}})
	if err != nil {
		return nil, err
	}
	return maps.FromSliceWith(list.Items, func(c *entities.SavedConnection) (string, struct{}) {
		return c.Name, struct{}{}
	}), nil
}

func isInvalidConnection(err error) bool {
	return errors.Is(err, errs.ErrConnectionNameRequired) ||
		errors.Is(err, errs.ErrConnectionURLCredentialsMixed) ||
		errors.Is(err, errs.ErrConnectionURLCredentialsConflict) ||
		errors.Is(err, errs.ErrConnectionURLInvalid)
}
