// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package micro

import (
	"context"
	"errors"
	"log/slog"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
	microsvc "github.com/dmit-4884/natscope/internal/services/micro"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

const (
	infoSubject  = "$SRV.INFO"
	statsSubject = "$SRV.STATS"
)

var _ microsvc.Service = (*Service)(nil)

// Service implements microsvc.Service.
type Service struct {
	nats     natssvc.ServiceDiscoverer
	registry protosvc.Registry
	logger   *slog.Logger
}

// New creates a discovery service.
func New(nats natssvc.ServiceDiscoverer, registry protosvc.Registry) *Service {
	return &Service{
		nats:     nats,
		registry: registry,
		logger:   slog.Default().With(slogx.Module("service:micro")),
	}
}

// ListServices lists the running services with the user's access to their info and stats.
func (s *Service) ListServices(ctx context.Context, connectionID string, skipStats bool) (*entities.MicroDiscovery, error) {
	infos, err := s.nats.MicroInfo(ctx, connectionID)
	infoAccess, err := accessOf(infoSubject, err)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "discover services")
	}
	result := &entities.MicroDiscovery{InfoAccess: infoAccess}
	if infoAccess.Status == entities.AccessDenied {
		return result, nil
	}

	var stats []entities.MicroReport
	if !skipStats {
		stats, err = s.nats.MicroStats(ctx, connectionID)
		statsAccess, accessErr := accessOf(statsSubject, err)
		if accessErr != nil {
			return nil, coreerrs.WrapOperation(accessErr, "read service statistics")
		}
		result.StatsAccess = &statsAccess
	}
	result.Services = groupServices(infos, stats)
	if len(result.Services) > 0 {
		s.loadMethodIndex(ctx).annotate(result.Services)
	}
	return result, nil
}

func accessOf(subject string, err error) (entities.AccessCheck, error) {
	if permErr, ok := errors.AsType[*errs.NATSPermissionError](err); ok {
		return entities.AccessCheck{Status: entities.AccessDenied, Operation: permErr.Operation, Subject: permErr.Subject}, nil
	}
	if err != nil {
		return entities.AccessCheck{}, err
	}
	return entities.AccessCheck{Status: entities.AccessAllowed, Operation: errs.PermissionOperationPublish, Subject: subject}, nil
}
