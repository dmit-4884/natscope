// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	connectionsvc "github.com/dmit-4884/natscope/internal/services/connections"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	storage "github.com/dmit-4884/natscope/internal/storages/connections"
	layoutsstorage "github.com/dmit-4884/natscope/internal/storages/layouts"
)

// Service implements connections.Service.
type Service struct {
	storage         storage.Storage
	layouts         layoutsstorage.Storage
	natService      natssvc.ConnectionManager
	hostCliContexts bool
	logger          *slog.Logger
}

// New creates a new connections service; hostCliContexts lets it read the nats CLI contexts of the host it runs on.
func New(storage storage.Storage, layouts layoutsstorage.Storage, natService natssvc.ConnectionManager, hostCliContexts bool) *Service {
	return &Service{
		storage:         storage,
		layouts:         layouts,
		natService:      natService,
		hostCliContexts: hostCliContexts,
		logger:          slog.Default().With(slogx.Module("service:connections")),
	}
}

// Create creates a new saved connection for the user.
func (s *Service) Create(
	ctx context.Context,
	in *entities.SavedConnectionCreate,
) (*entities.SavedConnection, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	// buf.validate min_len=1 runs on the raw request; re-check after the trim
	// normalizer so a whitespace-only name can't persist as an empty name.
	if strings.TrimSpace(in.Name) == "" {
		return nil, errs.ErrConnectionNameRequired
	}
	if err := validateLabel(in.Label); err != nil {
		return nil, err
	}

	urls, auth, err := liftURLCredentials(in.URLs, in.Auth)
	if err != nil {
		return nil, err
	}
	in.URLs, in.Auth = urls, auth

	conn := converter.Convert(in, entities.SavedConnectionNew())
	if conn.Label.IsEmpty() {
		conn.Label = nil
	}

	if err := s.storage.Save(ctx, conn); err != nil {
		s.logger.ErrorContext(ctx, "failed to save connection",
			slog.String("name", in.Name),
			slogx.Error(err))
		return nil, err
	}

	s.logger.InfoContext(ctx, "connection created",
		slog.String("id", conn.Id),
		slog.String("name", in.Name))

	return conn, nil
}

// Get returns errs.ErrSavedConnectionNotFound if not found.
func (s *Service) Get(ctx context.Context, id string) (*entities.SavedConnection, error) {
	return s.storage.Get(ctx, id)
}

func (s *Service) List(
	ctx context.Context,
	in *entities.SavedConnectionsList,
) (*entities.List[entities.SavedConnections], error) {
	return s.storage.List(ctx, in)
}

// Update merges via SavedConnection.ApplyUpdate (merge +
// collapse-empty-subtrees).
func (s *Service) Update(
	ctx context.Context,
	in *entities.SavedConnectionUpdate,
) (*entities.SavedConnection, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	// A provided name that trims to empty must be rejected (UpdateConnection has
	// no buf.validate min_len on name, so this is the only guard).
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return nil, errs.ErrConnectionNameRequired
	}
	if err := validateLabel(in.Label); err != nil {
		return nil, err
	}

	urls, auth, err := liftURLCredentials(in.URLs, in.Auth)
	if err != nil {
		return nil, err
	}
	in.URLs, in.Auth = urls, auth

	// A lifted URL credential counts as an Auth replace too.
	authReplaced := in.Auth != nil
	tlsReplaced := in.TLS != nil

	updated, err := s.storage.Update(ctx, in.Id, func(existing *entities.SavedConnection) (bool, bool) {
		existing.ApplyUpdate(in)
		return authReplaced, tlsReplaced
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update connection",
			slog.String("id", in.Id),
			slogx.Error(err))
		return nil, err
	}

	s.natService.DisconnectFromPool(in.Id)

	s.logger.InfoContext(ctx, "connection updated",
		slog.String("id", in.Id))

	return updated, nil
}

// Delete permanently removes a connection.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.storage.Delete(ctx, id); err != nil {
		// A missing connection is a client error; don't log it at ERROR.
		if errors.Is(err, errs.ErrSavedConnectionNotFound) {
			s.logger.DebugContext(ctx, "delete connection: not found",
				slog.String("id", id))
			return err
		}
		s.logger.ErrorContext(ctx, "failed to delete connection",
			slog.String("id", id),
			slogx.Error(err))
		return err
	}

	s.natService.DisconnectFromPool(id)

	if err := s.layouts.Delete(ctx, id); err != nil && !errors.Is(err, errs.ErrSidebarLayoutNotFound) {
		s.logger.WarnContext(ctx, "failed to delete sidebar layout",
			slog.String("id", id),
			slogx.Error(err))
	}

	s.logger.InfoContext(ctx, "connection deleted",
		slog.String("id", id))

	return nil
}

// Duplicate copies a connection under a new name; BaseEntity is saved/restored
// around the copy because WithIgnoreFields can't block embedded-struct paths.
func (s *Service) Duplicate(
	ctx context.Context,
	id string,
	newName string,
) (*entities.SavedConnection, error) {
	if strings.TrimSpace(newName) == "" {
		return nil, errs.ErrConnectionNameRequired
	}

	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	conn := entities.SavedConnectionNew()
	base := conn.BaseEntity
	converter.Convert(existing, conn, converter.WithIgnoreFields("Name", "Meta"))
	conn.BaseEntity = base
	conn.Name = strings.TrimSpace(newName)

	if err := s.storage.Save(ctx, conn); err != nil {
		s.logger.ErrorContext(ctx, "failed to duplicate connection",
			slog.String("source_id", id),
			slogx.Error(err))
		return nil, err
	}

	s.logger.InfoContext(ctx, "connection duplicated",
		slog.String("source_id", id),
		slog.String("new_id", conn.Id),
		slog.String("new_name", newName))

	return conn, nil
}

// TestConnection probes NATS. With ConnectionID set, the saved config
// overlays the request (blocks credential smuggling); the result is saved to Meta when the request tests the saved
// connection settings too.
func (s *Service) TestConnection(
	ctx context.Context,
	in *entities.TestConnectionRequest,
) (*entities.TestConnectionResult, error) {
	// Lift before the saved-config overlay so an ad-hoc probe carries no
	// credentials into the URL handed to nats.Connect.
	urls, auth, err := liftURLCredentials(in.URLs, in.Auth)
	if err != nil {
		return nil, err
	}
	in.URLs, in.Auth = urls, auth

	recordable := false
	if in.ConnectionID != "" {
		saved, getErr := s.storage.Get(ctx, in.ConnectionID)
		if getErr != nil {
			return nil, getErr
		}
		in.URLs, in.Auth, in.TLS = saved.URLs, saved.Auth, saved.TLS
		if in.Connection == nil {
			in.Connection = saved.Connection
		}
		recordable = settingsOf(in.Connection) == settingsOf(saved.Connection)
	}
	if in.ConnectTimeout == nil && in.Connection != nil {
		in.ConnectTimeout = in.Connection.ConnectTimeout
	}

	result, err := s.natService.TestConnection(ctx, in)
	if err != nil {
		return nil, err
	}

	if recordable {
		s.recordTestResult(ctx, in.ConnectionID, result)
	}

	return result, nil
}

// connectionSettings is what a ConnectionConfig changes about a connection, with unset and empty alike.
type connectionSettings struct {
	connectTimeout                             time.Duration
	name, inboxPrefix, domain, apiPrefix       string
	noEcho, noRandomize, ignoreDiscoveredHosts bool
}

func settingsOf(c *entities.ConnectionConfig) connectionSettings {
	if c == nil {
		return connectionSettings{}
	}
	return connectionSettings{
		connectTimeout:        ptr.Unwrap(c.ConnectTimeout),
		name:                  ptr.Unwrap(c.ConnectionName),
		inboxPrefix:           ptr.Unwrap(c.InboxPrefix),
		domain:                ptr.Unwrap(c.JetstreamDomain),
		apiPrefix:             ptr.Unwrap(c.JetstreamAPIPrefix),
		noEcho:                c.NoEcho,
		noRandomize:           c.NoRandomize,
		ignoreDiscoveredHosts: c.IgnoreDiscoveredServers,
	}
}

// recordTestResult writes a probe result to Meta in one storage transaction without bumping
// timestamps/ETag; errors are logged.
func (s *Service) recordTestResult(ctx context.Context, id string, result *entities.TestConnectionResult) {
	_, err := s.storage.Update(ctx, id, func(existing *entities.SavedConnection) (bool, bool) {
		existing.Meta = entities.NewConnectionMetaFromTestResult(result)
		// Meta-only write: keep existing vault secrets.
		return false, false
	})
	if err != nil {
		s.logger.WarnContext(ctx, "failed to persist test meta",
			slog.String("id", id),
			slogx.Error(err))
	}
}

// maxLabelText is the longest label text, in characters, a connection takes.
const maxLabelText = 16

// validateLabel applies the API's label rules to every path, workspace import included.
func validateLabel(label *entities.ConnectionLabel) error {
	if label == nil {
		return nil
	}
	formatting := func(r rune) bool { return unicode.In(r, unicode.Cc, unicode.Cf) }
	switch {
	case utf8.RuneCountInString(label.Text) > maxLabelText, strings.ContainsFunc(label.Text, formatting):
		return errs.ErrConnectionLabelInvalid
	case label.Color < entities.LabelColorUnspecified || label.Color > entities.LabelColorRed:
		return errs.ErrConnectionLabelInvalid
	}
	return nil
}

// liftURLCredentials moves credentials embedded in server URLs into the auth
// config, which is vault-backed; leaving them in the URL would persist them as
// plaintext. An explicit auth config in the same request is a conflict, not a
// tie to resolve.
func liftURLCredentials(urls []string, auth *entities.AuthConfig) ([]string, *entities.AuthConfig, error) {
	cleaned, creds, err := natsutil.SplitCredentials(urls)
	if err != nil {
		return nil, nil, err
	}
	if creds == nil {
		return cleaned, auth, nil
	}
	if !auth.IsEmpty() {
		return nil, nil, errs.ErrConnectionURLCredentialsConflict
	}
	return cleaned, authFromURLCredentials(creds), nil
}

// authFromURLCredentials maps lifted URL credentials onto the auth method.
func authFromURLCredentials(creds *natsutil.URLCredentials) *entities.AuthConfig {
	if creds.Token != "" {
		return &entities.AuthConfig{
			Method: entities.AuthMethodToken,
			Token:  ptr.Wrap(creds.Token),
		}
	}
	return &entities.AuthConfig{
		Method:   entities.AuthMethodUserPass,
		Username: ptr.Wrap(creds.Username),
		Password: ptr.Wrap(creds.Password),
	}
}

// Compile-time interface check.
var _ connectionsvc.Service = (*Service)(nil)
