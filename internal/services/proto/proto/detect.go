// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"cmp"
	"context"
	"regexp"
	"slices"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/services/proto/registry"
)

const (
	autoDetectMinScore = 85
	maxLearnedSubjects = 1000
	detectCandidates   = 2
	maxDetectMisses    = 3
	detectScansPerSec  = 50
	minVariableToken   = 8
)

var (
	uuidToken    = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	numericToken = regexp.MustCompile(`^[0-9]+$`)
)

// DetectTypes ranks the message types of one source, or of every enabled source, by how well data decodes as each.
func (s *Service) DetectTypes(ctx context.Context, data []byte, sourceID string, limit int) ([]entities.TypeCandidate, error) {
	out, err := s.rankTypes(ctx, data, sourceID, limit)
	if err != nil {
		return nil, err
	}
	return out[:min(len(out), limit)], nil
}

func (s *Service) rankTypes(ctx context.Context, data []byte, sourceID string, perSource int) ([]entities.TypeCandidate, error) {
	snaps, err := s.snapshotsFor(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	var out []entities.TypeCandidate
	for _, snap := range snaps {
		for _, c := range snap.Schema.DetectTypes(data, snap.Descriptor.MessageTypes, perSource) {
			decoded, renderErr := snap.Schema.RenderJSON(c.Message)
			if renderErr != nil {
				continue
			}
			out = append(out, entities.TypeCandidate{
				SourceID:       snap.SourceID,
				SourceRevision: snap.Revision,
				MessageType:    c.MessageType,
				Score:          c.Score,
				UnknownBytes:   c.UnknownBytes,
				Decoded:        decoded,
			})
		}
	}
	slices.SortStableFunc(out, func(a, b entities.TypeCandidate) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.MessageType, b.MessageType), cmp.Compare(a.SourceID, b.SourceID))
	})
	return out, nil
}

func (s *Service) snapshotsFor(ctx context.Context, sourceID string) ([]*registry.Snapshot, error) {
	if sourceID == "" {
		return s.activeSnapshots(ctx), nil
	}
	snap, err := s.snapshotForSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	return []*registry.Snapshot{snap}, nil
}

func (s *Service) autoDecode(ctx context.Context, subject string, data []byte) *entities.DecodeResult {
	key := generalizeSubject(subject)
	e, _ := s.learned.Get(key)
	if e.messageType != "" {
		if r := s.decodeLearned(ctx, e, data); r != nil {
			return r
		}
		e = learnedType{}
	}
	if e.misses >= maxDetectMisses || !s.detectBudget.Allow() {
		return nil
	}
	candidates, err := s.rankTypes(ctx, data, "", detectCandidates)
	if err != nil {
		return nil
	}
	best, ok := confident(candidates)
	if !ok {
		s.learned.Put(key, learnedType{misses: e.misses + 1})
		return nil
	}
	s.learned.Put(key, learnedType{sourceID: best.SourceID, messageType: best.MessageType})
	return &entities.DecodeResult{Success: true, Decoded: best.Decoded, MessageType: best.MessageType, SourceID: best.SourceID, Auto: true}
}

func confident(candidates []entities.TypeCandidate) (entities.TypeCandidate, bool) {
	if len(candidates) == 0 || candidates[0].UnknownBytes > 0 || candidates[0].Score < autoDetectMinScore {
		return entities.TypeCandidate{}, false
	}
	top := candidates[0]
	for _, c := range candidates[1:] {
		if c.MessageType != top.MessageType {
			return top, c.Score < top.Score
		}
	}
	return top, true
}

func (s *Service) decodeLearned(ctx context.Context, e learnedType, data []byte) *entities.DecodeResult {
	snap, err := s.snapshotForSource(ctx, e.sourceID)
	if err != nil {
		return nil
	}
	r := decodeWithSnapshot(snap, data, e.messageType, entities.Framing{})
	if !r.Success || len(r.UnknownFields) > 0 {
		return nil
	}
	r.MessageType, r.SourceID, r.Auto = e.messageType, e.sourceID, true
	return r
}

func generalizeSubject(subject string) string {
	tokens := strings.Split(subject, ".")
	for i, t := range tokens {
		if numericToken.MatchString(t) || uuidToken.MatchString(t) ||
			(len(t) >= minVariableToken && strings.ContainsAny(t, "0123456789")) {
			tokens[i] = "*"
		}
	}
	return strings.Join(tokens, ".")
}

type learnedType struct {
	sourceID    string
	messageType string
	misses      int
}
