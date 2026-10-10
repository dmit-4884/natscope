// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package micro

import (
	"context"
	"strings"
	"unicode"

	"github.com/dmit-4884/natscope/internal/entities"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

type methodIndex map[string][]entities.ProtoMethodMatch

func (s *Service) loadMethodIndex(ctx context.Context) methodIndex {
	types, err := s.registry.ListTypes(ctx, "")
	if err != nil {
		s.logger.DebugContext(ctx, "skip proto method matching", slogx.Error(err))
		return nil
	}
	index := make(methodIndex)
	for _, t := range types {
		if t.Kind != entities.SchemaTypeService || t.Dependency {
			continue
		}
		desc, err := s.registry.DescribeType(ctx, t.SourceID, "", t.FullName, false)
		if err != nil || desc == nil || len(desc.Services) == 0 {
			continue
		}
		for _, m := range desc.Services[0].Methods {
			if m.ClientStreaming || m.ServerStreaming {
				continue
			}
			key := strings.ToLower(m.Name)
			index[key] = append(index[key], entities.ProtoMethodMatch{
				SourceID:   t.SourceID,
				Service:    t.FullName,
				Method:     m.Name,
				InputType:  m.InputType,
				OutputType: m.OutputType,
			})
		}
	}
	return index
}

func (idx methodIndex) annotate(services []entities.MicroService) {
	if len(idx) == 0 {
		return
	}
	for i := range services {
		svc := &services[i]
		for j := range svc.Endpoints {
			svc.Endpoints[j].ProtoMethod = idx.match(svc.Name, svc.Endpoints[j].Name)
		}
		for j := range svc.Instances {
			endpoints := svc.Instances[j].Endpoints
			for k := range endpoints {
				endpoints[k].ProtoMethod = idx.match(svc.Name, endpoints[k].Name)
			}
		}
	}
}

func (idx methodIndex) match(serviceName, endpointName string) *entities.ProtoMethodMatch {
	candidates := idx[strings.ToLower(endpointName)]
	if len(candidates) == 1 {
		found := candidates[0]
		return &found
	}

	want := normalizeServiceName(serviceName)
	var found *entities.ProtoMethodMatch
	for _, c := range candidates {
		if normalizeServiceName(simpleName(c.Service)) != want {
			continue
		}
		if found != nil {
			return nil
		}
		match := c
		found = &match
	}
	return found
}

func simpleName(fullName string) string {
	return fullName[strings.LastIndex(fullName, ".")+1:]
}

func normalizeServiceName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	out := b.String()
	for _, suffix := range []string{"service", "svc"} {
		if trimmed, ok := strings.CutSuffix(out, suffix); ok && trimmed != "" {
			out = trimmed
			break
		}
	}
	if trimmed, ok := strings.CutSuffix(out, "s"); ok && trimmed != "" {
		out = trimmed
	}
	return out
}
