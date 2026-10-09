// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	livesvc "github.com/dmit-4884/natscope/internal/services/live"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
)

// Hardcoded session-tuning knobs; demoted from user preferences since
// frame cadence, buffer size, and poll interval are internal tuning only.
const (
	batchInterval         = 100 * time.Millisecond
	maxBatchSize          = 100
	messageBufferSize     = 100
	statsInterval         = 5 * time.Second
	maxSubjectCardinality = 1000
	maxBufferedBytes      = 16 << 20

	maxSubscriptionTargets = 100

	// defaultDeliverPolicy avoids replaying history implicitly; "see backlog"
	// is an explicit action elsewhere.
	defaultDeliverPolicy = "new"

	// User-facing setting values for SubscriptionMode.
	subscriptionModeCoreNATS         = "core_nats"
	subscriptionModeJetStreamOrdered = "jetstream_ordered"
)

// Compile-time check that Service satisfies livesvc.Service.
var _ livesvc.Service = (*Service)(nil)

// natsDeps bundles the narrow NATS roles the live service needs: stream metadata
// reads plus live subscriptions. The roles are injected separately so fx can
// resolve each, then embedded here for internal use.
type natsDeps struct {
	natssvc.StreamReader
	natssvc.Subscriber
	natssvc.KVStore
}

// Service implements livesvc.Service.
type Service struct {
	natsService     natsDeps
	protoService    protosvc.Codec
	settingsService settingssvc.Service
	logger          *slog.Logger

	sessionsMu sync.RWMutex
	sessions   map[*sessionState]struct{}
}

// sessionState is the per-Subscribe shared state used by BroadcastProtoReload.
// One session ↔ one in-flight Subscribe call.
type sessionState struct {
	decoderDirty atomic.Int32

	// totalMessages counts every message on arrival, including ones dropped later.
	totalMessages atomic.Int64

	// messagesDropped is incremented by the subscription producer (buffer full)
	// and the runLoop rate limiter, and read by the runLoop stats emitter.
	messagesDropped atomic.Int64

	// bufferedBytes is the payload volume in the message buffer, capped at maxBufferedBytes.
	bufferedBytes atomic.Int64

	connectionID string
	// exclude are the muted subject patterns, dropped as they arrive.
	exclude  []string
	lost     chan struct{}
	lostOnce sync.Once

	denials        chan *entities.LiveError
	silentMu       sync.RWMutex
	silentSubjects map[string]struct{}
	// liveSubjects are the subjects whose subscription has delivered, so the server did not refuse it.
	liveSubjects map[string]struct{}
	// held are copies waiting to learn whether an earlier subject delivers them.
	held      []*heldMessage
	heldBytes int
	ended     bool
}

func newSessionState(connectionID string) *sessionState {
	return &sessionState{
		connectionID:   connectionID,
		lost:           make(chan struct{}),
		denials:        make(chan *entities.LiveError, maxSubscriptionTargets),
		silentSubjects: make(map[string]struct{}),
		liveSubjects:   make(map[string]struct{}),
	}
}

// markLive records that subject's subscription delivers.
func (sess *sessionState) markLive(subject string) {
	sess.silentMu.RLock()
	_, known := sess.liveSubjects[subject]
	sess.silentMu.RUnlock()
	if known {
		return
	}
	sess.silentMu.Lock()
	sess.liveSubjects[subject] = struct{}{}
	shown := sess.settleLocked()
	sess.silentMu.Unlock()
	deliverHeld(shown)
}

func (sess *sessionState) reportDenied(subject string, err error) {
	if !sess.markSilent(subject) {
		return
	}
	select {
	case sess.denials <- liveErrorFor(subject, err):
	default:
	}
}

// markSilent records that subject delivers nothing; false when it was recorded before.
func (sess *sessionState) markSilent(subject string) bool {
	sess.silentMu.Lock()
	if _, seen := sess.silentSubjects[subject]; seen {
		sess.silentMu.Unlock()
		return false
	}
	sess.silentSubjects[subject] = struct{}{}
	shown := sess.settleLocked()
	sess.silentMu.Unlock()
	deliverHeld(shown)
	return true
}

func (sess *sessionState) excludes(subject string) bool {
	return slices.ContainsFunc(sess.exclude, func(pattern string) bool { return natsutil.MatchSubject(pattern, subject) })
}

func (sess *sessionState) markLost() {
	sess.lostOnce.Do(func() { close(sess.lost) })
}

// New creates a live Service.
func New(
	streamReader natssvc.StreamReader,
	subscriber natssvc.Subscriber,
	kvStore natssvc.KVStore,
	protoService protosvc.Codec,
	settingsService settingssvc.Service,
) *Service {
	s := &Service{
		natsService:     natsDeps{streamReader, subscriber, kvStore},
		protoService:    protoService,
		settingsService: settingsService,
		logger:          slog.Default().With(slogx.Module("service:live")),
		sessions:        make(map[*sessionState]struct{}),
	}
	subscriber.OnDisconnect(s.endSessionsFor)
	return s
}

// endSessionsFor ends every session subscribed through connectionID.
func (s *Service) endSessionsFor(connectionID string) {
	s.sessionsMu.RLock()
	defer s.sessionsMu.RUnlock()

	for sess := range s.sessions {
		if sess.connectionID == connectionID {
			sess.markLost()
		}
	}
}

// BroadcastProtoReload flips the dirty bit on every active session so each
// one re-initializes its decoder on the next message processed.
func (s *Service) BroadcastProtoReload() {
	s.sessionsMu.RLock()
	sessions := make([]*sessionState, 0, len(s.sessions))
	for sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.sessionsMu.RUnlock()

	for _, sess := range sessions {
		sess.decoderDirty.Store(1)
	}
}

func (s *Service) registerSession(sess *sessionState) {
	s.sessionsMu.Lock()
	s.sessions[sess] = struct{}{}
	s.sessionsMu.Unlock()
}

func (s *Service) unregisterSession(sess *sessionState) {
	s.sessionsMu.Lock()
	delete(s.sessions, sess)
	s.sessionsMu.Unlock()
}
