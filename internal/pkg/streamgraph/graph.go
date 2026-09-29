// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package streamgraph

import (
	"cmp"
	"slices"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
)

const (
	kvStreamPrefix     = "KV_"
	objectStreamPrefix = "OBJ_"
)

// Build returns the relations graph of the given streams. Streams that take part in no relation are
// left out. External and subject placeholder IDs contain a space, which stream names cannot, so they
// never clash with a stream; a missing upstream keeps its name, which no listed stream has.
func Build(streams []entities.StreamInfo) *entities.StreamRelations {
	b := &builder{
		byName: make(map[string]*entities.StreamInfo, len(streams)),
		nodes:  map[string]entities.StreamRelationNode{},
	}
	for i := range streams {
		s := &streams[i]
		b.byName[s.Config.Name] = s
		b.sorted = append(b.sorted, s)
	}
	slices.SortFunc(b.sorted, func(x, y *entities.StreamInfo) int { return cmp.Compare(x.Config.Name, y.Config.Name) })

	for _, s := range b.sorted {
		b.addLinks(s)
	}
	for _, s := range b.sorted {
		b.addRepublish(s)
	}

	nodes := make([]entities.StreamRelationNode, 0, len(b.nodes))
	for _, node := range b.nodes {
		nodes = append(nodes, node)
	}
	slices.SortFunc(nodes, func(x, y entities.StreamRelationNode) int { return cmp.Compare(x.ID, y.ID) })

	return &entities.StreamRelations{Nodes: nodes, Edges: b.edges}
}

type builder struct {
	byName map[string]*entities.StreamInfo
	sorted []*entities.StreamInfo
	nodes  map[string]entities.StreamRelationNode
	edges  []entities.StreamRelationEdge
}

func (b *builder) addLinks(s *entities.StreamInfo) {
	if s.Config.Mirror != nil {
		b.link(entities.StreamRelationMirror, s, s.Config.Mirror, s.Mirror)
	}
	states := matchStates(s.Config.Sources, s.Sources)
	for i, ref := range s.Config.Sources {
		if ref == nil {
			continue
		}
		b.link(entities.StreamRelationSource, s, ref, states[i])
	}
}

func (b *builder) link(kind entities.StreamRelationKind, s *entities.StreamInfo, ref *entities.StreamSourceRef, state *entities.StreamSourceInfo) {
	b.edges = append(b.edges, entities.StreamRelationEdge{
		Kind:   kind,
		From:   b.upstream(ref),
		To:     b.stream(s),
		Source: ref,
		State:  state,
	})
}

func (b *builder) addRepublish(s *entities.StreamInfo) {
	republish := s.Config.Republish
	if republish == nil || republish.Dest == "" {
		return
	}
	pattern := natsutil.TransformDestinationPattern(republish.Dest)
	captured := false
	for _, target := range b.sorted {
		if target == s || !capturesAny(target.Config.Subjects, pattern) {
			continue
		}
		captured = true
		b.edges = append(b.edges, entities.StreamRelationEdge{
			Kind:      entities.StreamRelationRepublish,
			From:      b.stream(s),
			To:        b.stream(target),
			Republish: republish,
		})
	}
	if captured {
		return
	}
	id := "subject " + republish.Dest
	b.nodes[id] = entities.StreamRelationNode{ID: id, Name: republish.Dest, Kind: entities.StreamNodeSubject}
	b.edges = append(b.edges, entities.StreamRelationEdge{
		Kind:      entities.StreamRelationRepublish,
		From:      b.stream(s),
		To:        id,
		Republish: republish,
	})
}

func (b *builder) upstream(ref *entities.StreamSourceRef) string {
	if api := externalAPI(ref.External); api != "" {
		id := "external " + api + " " + ref.Name
		b.nodes[id] = entities.StreamRelationNode{ID: id, Name: ref.Name, Kind: entities.StreamNodeExternal, External: ref.External}
		return id
	}
	if s, ok := b.byName[ref.Name]; ok {
		return b.stream(s)
	}
	b.nodes[ref.Name] = entities.StreamRelationNode{ID: ref.Name, Name: ref.Name, Kind: entities.StreamNodeMissing}
	return ref.Name
}

func (b *builder) stream(s *entities.StreamInfo) string {
	name := s.Config.Name
	if _, ok := b.nodes[name]; !ok {
		info := *s
		info.Raw = ""
		b.nodes[name] = entities.StreamRelationNode{ID: name, Name: name, Kind: kindOf(name), Info: &info}
	}
	return name
}

func kindOf(name string) entities.StreamNodeKind {
	switch {
	case strings.HasPrefix(name, kvStreamPrefix):
		return entities.StreamNodeKV
	case strings.HasPrefix(name, objectStreamPrefix):
		return entities.StreamNodeObjectStore
	default:
		return entities.StreamNodeStream
	}
}

func capturesAny(subjects []string, pattern string) bool {
	return slices.ContainsFunc(subjects, func(subject string) bool { return natsutil.SubjectsCollide(subject, pattern) })
}

// matchStates pairs each source with the live state the server reports for it. States come in no
// particular order: a state is matched by upstream and filters first, and only states left over are
// then matched by upstream alone, so one source's fallback never takes another source's exact state.
func matchStates(refs []*entities.StreamSourceRef, states []*entities.StreamSourceInfo) []*entities.StreamSourceInfo {
	matched := make([]*entities.StreamSourceInfo, len(refs))
	free := slices.Clone(states)
	claim := func(i int, exact bool) {
		ref := refs[i]
		for j, state := range free {
			if state == nil || state.Name != ref.Name || externalAPI(state.External) != externalAPI(ref.External) {
				continue
			}
			if exact && !slices.Equal(filters(state.FilterSubject, state.SubjectTransforms), filters(ref.FilterSubject, ref.SubjectTransforms)) {
				continue
			}
			matched[i] = state
			free[j] = nil
			return
		}
	}
	for _, exact := range []bool{true, false} {
		for i, ref := range refs {
			if ref != nil && matched[i] == nil {
				claim(i, exact)
			}
		}
	}
	return matched
}

func filters(filterSubject string, transforms []entities.SubjectTransformConfig) []string {
	out := []string{}
	if filterSubject != "" {
		out = append(out, filterSubject)
	}
	for _, t := range transforms {
		if t.Source != "" {
			out = append(out, t.Source)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func externalAPI(external *entities.ExternalStreamRef) string {
	if external == nil {
		return ""
	}
	return external.ApiPrefix
}
