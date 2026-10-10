// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"errors"
	"log/slog"

	"github.com/altessa-s/go-atlas/core/collections/maps"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natscontext"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
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

// ImportCliContexts creates a connection per named context; a missing, unusable or already saved one is skipped, as is
// one that fails to save, so the contexts already created are reported.
func (s *Service) ImportCliContexts(
	ctx context.Context,
	names []string,
	files []entities.CliContextFile,
) (*entities.CliContextImport, error) {
	found, err := s.cliContexts(files)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]entities.CliContext, len(found.Contexts))
	for _, c := range found.Contexts {
		if _, taken := byName[c.Name]; !taken {
			byName[c.Name] = c
		}
	}

	res := &entities.CliContextImport{}
	skip := func(name, reason string) {
		res.Skipped = append(res.Skipped, entities.CliContextSkip{Name: name, Reason: reason})
	}
	done := make(map[string]bool, len(names))
	for _, name := range names {
		if done[name] {
			continue
		}
		done[name] = true
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
			s.logger.WarnContext(ctx, "nats CLI context not imported", slog.String("name", name), slogx.Error(createErr))
			skip(name, "the connection could not be saved")
		}
	}

	s.logger.InfoContext(ctx, "nats CLI contexts imported",
		slog.Int("created", len(res.Created)),
		slog.Int("skipped", len(res.Skipped)))
	return res, nil
}

func (s *Service) cliContexts(files []entities.CliContextFile) (*entities.CliContexts, error) {
	found := &entities.CliContexts{}
	switch {
	case len(files) > 0:
		found.Contexts = natscontext.Parse(files)
	case !s.hostCliContexts:
		return nil, errs.ErrCliContextsHostDisabled
	default:
		dir, err := natscontext.Dir()
		if err != nil {
			return nil, err
		}
		if found.Contexts, err = natscontext.Read(dir); err != nil {
			return nil, err
		}
		found.Dir = dir
	}
	for i := range found.Contexts {
		liftContextCredentials(&found.Contexts[i])
	}
	return found, nil
}

// liftContextCredentials moves credentials a context embeds in its URLs into its auth, as Create would.
func liftContextCredentials(c *entities.CliContext) {
	if c.Connection == nil {
		return
	}
	urls, auth, err := liftURLCredentials(c.Connection.URLs, c.Connection.Auth)
	if err != nil {
		c.Connection.URLs = natsutil.StripCredentials(c.Connection.URLs)
		c.Warnings = append(c.Warnings, err.Error())
		return
	}
	c.Connection.URLs, c.Connection.Auth = urls, auth
}

func (s *Service) savedNames(ctx context.Context) (map[string]struct{}, error) {
	list, err := s.storage.List(ctx, &entities.SavedConnectionsList{ListBase: entities.ListBase{Limit: new(int64(allConnectionsLimit))}})
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
