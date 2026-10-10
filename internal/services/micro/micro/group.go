// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package micro

import (
	"cmp"
	"maps"
	"slices"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"

	"golang.org/x/mod/semver"
)

type instanceKey struct{ name, id string }

type endpointKey struct{ name, subject string }

func groupServices(infos, stats []entities.MicroReport) []entities.MicroService {
	statsByInstance := make(map[instanceKey]entities.MicroReport, len(stats))
	for _, r := range stats {
		statsByInstance[instanceKey{r.Name, r.ID}] = r
	}
	infoByInstance := make(map[instanceKey]entities.MicroReport, len(infos))
	for _, r := range infos {
		infoByInstance[instanceKey{r.Name, r.ID}] = r
	}

	seen := make(map[instanceKey]struct{}, len(infos))
	byName := make(map[string][]entities.MicroReport)
	for _, r := range slices.Concat(infos, stats) {
		k := instanceKey{r.Name, r.ID}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		byName[r.Name] = append(byName[r.Name], r)
	}

	services := make([]entities.MicroService, 0, len(byName))
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		reports := byName[name]
		slices.SortFunc(reports, func(a, b entities.MicroReport) int { return cmp.Compare(a.ID, b.ID) })

		svc := entities.MicroService{Name: name}
		versions := make(map[string]struct{})
		for _, r := range reports {
			k := instanceKey{r.Name, r.ID}
			st, hasStats := statsByInstance[k]
			instance := buildInstance(r, st, hasStats)
			instance.RTT = cmp.Or(infoByInstance[k].RTT, st.RTT)
			instance.InfoJSON = infoByInstance[k].Raw
			svc.Instances = append(svc.Instances, instance)
			if svc.Description == "" {
				svc.Description = r.Description
			}
			if instance.Version != "" {
				versions[instance.Version] = struct{}{}
			}
		}
		svc.Versions = slices.SortedFunc(maps.Keys(versions), compareVersions)
		svc.Endpoints = mergeEndpoints(svc.Instances)
		services = append(services, svc)
	}
	return services
}

func buildInstance(info, stats entities.MicroReport, hasStats bool) entities.MicroInstance {
	instance := entities.MicroInstance{
		ID:        info.ID,
		Version:   cmp.Or(info.Version, stats.Version),
		Metadata:  info.Metadata,
		Endpoints: slices.Clone(info.Endpoints),
	}
	if !hasStats {
		for i := range instance.Endpoints {
			instance.Endpoints[i].Stats = nil
		}
		return instance
	}

	started := stats.Started
	instance.Started = &started
	instance.StatsJSON = stats.Raw
	statsByEndpoint := make(map[endpointKey]*entities.MicroEndpointStats, len(stats.Endpoints))
	for _, e := range stats.Endpoints {
		statsByEndpoint[endpointKey{e.Name, e.Subject}] = e.Stats
	}
	for i, e := range instance.Endpoints {
		instance.Endpoints[i].Stats = statsByEndpoint[endpointKey{e.Name, e.Subject}]
	}
	return instance
}

func mergeEndpoints(instances []entities.MicroInstance) []entities.MicroEndpoint {
	var order []endpointKey
	merged := make(map[endpointKey]*entities.MicroEndpoint)
	for _, instance := range instances {
		for _, e := range instance.Endpoints {
			k := endpointKey{e.Name, e.Subject}
			m, ok := merged[k]
			if !ok {
				first := e
				first.Stats = nil
				m = &first
				merged[k] = m
				order = append(order, k)
			}
			if e.Stats != nil {
				m.Stats = addStats(m.Stats, e.Stats)
			}
		}
	}

	out := make([]entities.MicroEndpoint, 0, len(order))
	for _, k := range order {
		m := merged[k]
		if m.Stats != nil && m.Stats.NumRequests > 0 {
			m.Stats.AverageProcessingTime = m.Stats.ProcessingTime / time.Duration(m.Stats.NumRequests)
		}
		out = append(out, *m)
	}
	return out
}

func addStats(acc, s *entities.MicroEndpointStats) *entities.MicroEndpointStats {
	if acc == nil {
		acc = &entities.MicroEndpointStats{}
	}
	acc.NumRequests += s.NumRequests
	acc.NumErrors += s.NumErrors
	acc.ProcessingTime += s.ProcessingTime
	acc.LastError = cmp.Or(acc.LastError, s.LastError)
	return acc
}

func compareVersions(a, b string) int {
	va, vb := "v"+a, "v"+b
	switch validA, validB := semver.IsValid(va), semver.IsValid(vb); {
	case validA && validB:
		return cmp.Or(semver.Compare(va, vb), cmp.Compare(a, b))
	case validA != validB:
		if validA {
			return -1
		}
		return 1
	default:
		return cmp.Compare(a, b)
	}
}
