// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"

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
	lost         chan struct{}
	lostOnce     sync.Once

	denials        chan *entities.LiveError
	deniedMu       sync.Mutex
	deniedSubjects map[string]struct{}
}

func newSessionState(connectionID string) *sessionState {
	return &sessionState{
		connectionID:   connectionID,
		lost:           make(chan struct{}),
		denials:        make(chan *entities.LiveError, maxSubscriptionTargets),
		deniedSubjects: make(map[string]struct{}),
	}
}

func (sess *sessionState) reportDenied(subject string, err error) {
	sess.deniedMu.Lock()
	if _, seen := sess.deniedSubjects[subject]; seen {
		sess.deniedMu.Unlock()
		return
	}
	sess.deniedSubjects[subject] = struct{}{}
	sess.deniedMu.Unlock()

	select {
	case sess.denials <- liveErrorFor(subject, err):
	default:
	}
}

func (sess *sessionState) markLost() {
	sess.lostOnce.Do(func() { close(sess.lost) })
}

// New creates a live Service.
func New(
	streamReader natssvc.StreamReader,
	subscriber natssvc.Subscriber,
	protoService protosvc.Codec,
	settingsService settingssvc.Service,
) *Service {
	s := &Service{
		natsService:     natsDeps{streamReader, subscriber},
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
