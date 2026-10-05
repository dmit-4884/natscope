// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"slices"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
)

// A copy of a message that an earlier subject may also deliver waits up to holdTimeout to learn whether that subject
// delivers it; at most maxHeldMessages copies wait at once, the rest are shown.
const (
	holdTimeout     = time.Second
	maxHeldMessages = 1000
)

// coreTarget is a core subscription listed before others in a session; it takes the messages it delivers.
type coreTarget struct {
	subject string
	sub     entities.Subscription
}

// coverage says whether an earlier subject takes a message.
type coverage int

const (
	// coverageNone means no earlier subject delivers the message.
	coverageNone coverage = iota
	// coverageTaken means an earlier subject that has proved live delivers the message.
	coverageTaken
	// coverageUnknown means an earlier subject delivers the message but may still be refused.
	coverageUnknown
)

// heldMessage is a copy waiting for the earlier subjects that may deliver it.
type heldMessage struct {
	msg     *entities.NatsMessage
	earlier []coreTarget
	deliver entities.MessageHandler
	timer   *time.Timer
}

// coverageLocked reports how the earlier subjects take subject; the caller holds silentMu.
func (sess *sessionState) coverageLocked(earlier []coreTarget, subject string) coverage {
	result := coverageNone
	for _, e := range earlier {
		if !e.sub.Delivers(subject) || !takesInternal(e.subject, subject) {
			continue
		}
		if _, silent := sess.silentSubjects[e.subject]; silent {
			continue
		}
		if _, live := sess.liveSubjects[e.subject]; live {
			return coverageTaken
		}
		result = coverageUnknown
	}
	return result
}

// route delivers a copy no earlier subject takes, drops one an earlier live subject takes, and holds one whose
// earlier subjects have not proved live or been refused yet.
func (sess *sessionState) route(earlier []coreTarget, msg *entities.NatsMessage, deliver entities.MessageHandler) {
	sess.silentMu.Lock()
	switch sess.coverageLocked(earlier, msg.Subject) {
	case coverageTaken:
		sess.silentMu.Unlock()
		return
	case coverageUnknown:
		if !sess.ended && len(sess.held) < maxHeldMessages {
			held := &heldMessage{msg: msg, earlier: earlier, deliver: deliver}
			held.timer = time.AfterFunc(holdTimeout, func() { sess.release(held) })
			sess.held = append(sess.held, held)
			sess.silentMu.Unlock()
			return
		}
	case coverageNone:
	}
	sess.silentMu.Unlock()
	deliver(msg)
}

// settleLocked drops the held copies an earlier subject now takes and returns the ones none does; the caller
// holds silentMu and delivers what it returns after unlocking.
func (sess *sessionState) settleLocked() []*heldMessage {
	var shown []*heldMessage
	sess.held = slices.DeleteFunc(sess.held, func(h *heldMessage) bool {
		switch sess.coverageLocked(h.earlier, h.msg.Subject) {
		case coverageTaken:
			h.timer.Stop()
			return true
		case coverageNone:
			h.timer.Stop()
			shown = append(shown, h)
			return true
		case coverageUnknown:
		}
		return false
	})
	return shown
}

// release shows a held copy whose earlier subjects stayed silent past holdTimeout.
func (sess *sessionState) release(held *heldMessage) {
	sess.silentMu.Lock()
	at := slices.Index(sess.held, held)
	if at < 0 || sess.ended {
		sess.silentMu.Unlock()
		return
	}
	sess.held = slices.Delete(sess.held, at, at+1)
	sess.silentMu.Unlock()
	held.deliver(held.msg)
}

// endHolding drops every held copy when the session ends.
func (sess *sessionState) endHolding() {
	sess.silentMu.Lock()
	defer sess.silentMu.Unlock()
	sess.ended = true
	for _, h := range sess.held {
		h.timer.Stop()
	}
	sess.held = nil
}

func deliverHeld(shown []*heldMessage) {
	for _, h := range shown {
		h.deliver(h.msg)
	}
}
