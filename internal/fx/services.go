// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package fx

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/pkg/bbstore"
	"github.com/dmit-4884/natscope/internal/pkg/secrets"

	connectionssvc "github.com/dmit-4884/natscope/internal/services/connections"
	connectionsService "github.com/dmit-4884/natscope/internal/services/connections/connections"
	fwsvc "github.com/dmit-4884/natscope/internal/services/filewatcher"
	gitfetchersvc "github.com/dmit-4884/natscope/internal/services/gitfetcher"
	gitfetcherService "github.com/dmit-4884/natscope/internal/services/gitfetcher/gitfetcher"
	historysvc "github.com/dmit-4884/natscope/internal/services/history"
	historyService "github.com/dmit-4884/natscope/internal/services/history/history"
	livesvc "github.com/dmit-4884/natscope/internal/services/live"
	liveService "github.com/dmit-4884/natscope/internal/services/live/live"
	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	mappingsService "github.com/dmit-4884/natscope/internal/services/mappings/mappings"
	messagessvc "github.com/dmit-4884/natscope/internal/services/messages"
	messagesService "github.com/dmit-4884/natscope/internal/services/messages/messages"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	protoService "github.com/dmit-4884/natscope/internal/services/proto/proto"
	publishsvc "github.com/dmit-4884/natscope/internal/services/publish"
	publishService "github.com/dmit-4884/natscope/internal/services/publish/publish"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
	settingsService "github.com/dmit-4884/natscope/internal/services/settings/settings"
	templatessvc "github.com/dmit-4884/natscope/internal/services/templates"
	templatesService "github.com/dmit-4884/natscope/internal/services/templates/templates"
	workspacesvc "github.com/dmit-4884/natscope/internal/services/workspace"
	workspaceSections "github.com/dmit-4884/natscope/internal/services/workspace/sections"
	workspaceService "github.com/dmit-4884/natscope/internal/services/workspace/workspace"
	connectionsStorageIface "github.com/dmit-4884/natscope/internal/storages/connections"
	connectionsBbolt "github.com/dmit-4884/natscope/internal/storages/connections/bbolt"
	historyBbolt "github.com/dmit-4884/natscope/internal/storages/history/bbolt"
	layoutsStorageIface "github.com/dmit-4884/natscope/internal/storages/layouts"
	layoutsBbolt "github.com/dmit-4884/natscope/internal/storages/layouts/bbolt"
	mappingsBbolt "github.com/dmit-4884/natscope/internal/storages/mappings/bbolt"
	conflictsBbolt "github.com/dmit-4884/natscope/internal/storages/proto/conflicts/bbolt"
	descriptorsBbolt "github.com/dmit-4884/natscope/internal/storages/proto/descriptors/bbolt"
	filesetsBbolt "github.com/dmit-4884/natscope/internal/storages/proto/filesets/bbolt"
	sourcesBbolt "github.com/dmit-4884/natscope/internal/storages/proto/sources/bbolt"
	settingsBbolt "github.com/dmit-4884/natscope/internal/storages/settings/bbolt"
	templatesBbolt "github.com/dmit-4884/natscope/internal/storages/templates/bbolt"
)

// ServicesModule provides all business logic services.
func ServicesModule() fx.Option {
	return fx.Module("services",
		fx.Provide(fx.Annotate(
			newProtoService,
			fx.As(new(protosvc.Registry)),
			fx.As(new(protosvc.Codec)),
			fx.As(new(protosvc.SourceManager)),
		)),
		fx.Provide(fx.Annotate(newConnectionsStorage, fx.As(new(connectionsStorageIface.Storage)))),
		fx.Provide(newLayoutsStorage),
		fx.Provide(fx.Annotate(connectionsService.New, fx.As(new(connectionssvc.Service)))),
		fx.Provide(fx.Annotate(newMappingsService, fx.As(new(mappingssvc.Service)))),
		fx.Provide(fx.Annotate(newHistoryService, fx.As(new(historysvc.Service)))),
		fx.Provide(fx.Annotate(newSettingsService, fx.As(new(settingssvc.Service)))),
		fx.Provide(fx.Annotate(newTemplatesService, fx.As(new(templatessvc.Service)))),
		fx.Provide(fx.Annotate(gitfetcherService.New, fx.As(new(gitfetchersvc.Service)))),
		fx.Provide(fx.Annotate(liveService.New, fx.As(new(livesvc.Service)))),
		fx.Provide(fx.Annotate(publishService.New, fx.As(new(publishsvc.Service)))),
		fx.Provide(fx.Annotate(messagesService.New, fx.As(new(messagessvc.Service)))),

		// Mapping changes re-initialize live decoders like a proto reload does.
		fx.Invoke(func(mappingsSvc mappingssvc.Service, liveSvc livesvc.Service) error {
			ms, ok := mappingsSvc.(*mappingsService.Service)
			if !ok {
				return fmt.Errorf("mappings change wiring: Service is %T, want *mappings.Service", mappingsSvc)
			}
			ms.SetOnChangeCallback(liveSvc.BroadcastProtoReload)
			return nil
		}),

		// Workspace export/import: each domain registers a Section into the
		// "workspace-sections" group; new domain = implement Section + one line here.
		fx.Provide(AsWorkspaceSection(workspaceSections.NewConnectionsSection)),
		fx.Provide(AsWorkspaceSection(workspaceSections.NewProtoSourcesSection)),
		fx.Provide(AsWorkspaceSection(workspaceSections.NewMappingsSection)),
		fx.Provide(AsWorkspaceSection(workspaceSections.NewTemplatesSection)),
		fx.Provide(AsWorkspaceSection(workspaceSections.NewSettingsSection)),
		fx.Provide(fx.Annotate(
			workspaceService.New,
			fx.ParamTags(`group:"workspace-sections"`),
			fx.As(new(workspacesvc.Service)),
		)),
	)
}

// AsWorkspaceSection annotates a Section constructor so its result lands in the
// workspace-sections group consumed by the workspace coordinator service.
func AsWorkspaceSection(f any) any {
	return fx.Annotate(f, fx.As(new(workspacesvc.Section)), fx.ResultTags(`group:"workspace-sections"`))
}

var obsoleteProtoBuckets = []string{"proto_descriptors", "proto_versions", "proto_selections"}

// newProtoService creates a new unified Proto service backed by bbolt stores.
func newProtoService(
	db *bbstore.DB,
	vault secrets.Vault,
	gitFetcher gitfetchersvc.Service,
	mappingsSvc mappingssvc.Service,
	fileWatcher fwsvc.Service,
) (*protoService.Service, error) {
	if err := db.DropBuckets(context.Background(), obsoleteProtoBuckets...); err != nil {
		return nil, errors.WrapOperation(err, "drop obsolete proto buckets")
	}
	sources, err := sourcesBbolt.New(context.Background(), db, vault)
	if err != nil {
		return nil, errors.WrapOperation(err, "create proto sources storage")
	}
	fileSets, err := filesetsBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create proto file sets storage")
	}
	descriptors, err := descriptorsBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create proto descriptors storage")
	}
	conflicts, err := conflictsBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create proto conflicts storage")
	}

	return protoService.New(
		sources,
		fileSets,
		descriptors,
		conflicts,
		gitFetcher,
		mappingsSvc,
		fileWatcher,
	), nil
}

// newConnectionsStorage creates the bbolt connections storage. Secret auth/TLS
// fields are kept in the keychain vault, never in the database.
func newConnectionsStorage(db *bbstore.DB, vault secrets.Vault) (connectionsStorageIface.Storage, error) {
	storage, err := connectionsBbolt.New(context.Background(), db, vault)
	if err != nil {
		return nil, errors.WrapOperation(err, "create connections storage")
	}
	return storage, nil
}

// newLayoutsStorage creates the bbolt sidebar layouts storage.
func newLayoutsStorage(db *bbstore.DB) (layoutsStorageIface.Storage, error) {
	storage, err := layoutsBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create sidebar layouts storage")
	}
	return storage, nil
}

// newMappingsService creates a new mappings service using bbolt storage.
func newMappingsService(db *bbstore.DB) (mappingssvc.Service, error) {
	storage, err := mappingsBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create mappings storage")
	}
	return mappingsService.New(storage), nil
}

// newHistoryService creates a new history service backed by bbolt.
func newHistoryService(db *bbstore.DB) (historysvc.Service, error) {
	storage, err := historyBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create history storage")
	}
	return historyService.New(storage), nil
}

// newSettingsService creates a new settings service using bbolt storage.
func newSettingsService(db *bbstore.DB) (settingssvc.Service, error) {
	storage, err := settingsBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create settings storage")
	}
	return settingsService.New(storage), nil
}

// newTemplatesService creates a new templates service using bbolt storage.
func newTemplatesService(db *bbstore.DB) (templatessvc.Service, error) {
	storage, err := templatesBbolt.New(context.Background(), db)
	if err != nil {
		return nil, errors.WrapOperation(err, "create templates storage")
	}
	return templatesService.New(storage), nil
}
