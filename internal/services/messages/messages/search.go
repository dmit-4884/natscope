// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"
	"encoding/base64"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// One search run stops at whichever of these budgets it reaches first; the caller continues with the next run.
const (
	searchMaxScanned   = 100_000
	searchMaxDuration  = 20 * time.Second
	searchMaxMatches   = 500
	searchFirstWindow  = 1_000
	searchMaxWindow    = 20_000
	searchProgressTick = 250 * time.Millisecond
	searchMatchBatch   = 50
	searchFastWindow   = time.Second
	searchWindowGrowth = 2
	searchSubjectCache = 10_000
)

// searchClock is the clock budgets and progress are measured with.
var searchClock = time.Now

// searchMatcher holds the conditions a message must meet.
type searchMatcher struct {
	text        string
	re          *regexp.Regexp
	headerName  string
	headerValue string
	fromTime    *time.Time
	toTime      *time.Time
}

func newSearchMatcher(in *entities.MessageSearchRequest) (*searchMatcher, error) {
	if strings.TrimSpace(in.HeaderName) == "" && in.HeaderValue != "" {
		return nil, &errs.NATSValidationError{Description: "a header value needs a header name"}
	}
	m := &searchMatcher{
		headerName:  strings.TrimSpace(in.HeaderName),
		headerValue: in.HeaderValue,
		fromTime:    in.FromTime,
		toTime:      in.ToTime,
	}
	switch {
	case in.Text == "":
	case in.Regex:
		re, err := regexp.Compile(in.Text)
		if err != nil {
			return nil, &errs.NATSValidationError{Description: fmt.Sprintf("invalid regular expression: %v", err), Cause: err}
		}
		m.re = re
	default:
		m.text = strings.ToLower(in.Text)
	}
	return m, nil
}

func (m *searchMatcher) needsPayload() bool {
	return m.text != "" || m.re != nil
}

func (m *searchMatcher) matchText(s string) bool {
	if m.re != nil {
		return m.re.MatchString(s)
	}
	return strings.Contains(strings.ToLower(s), m.text)
}

// matchHeaders checks the header and time conditions, which need no payload.
func (m *searchMatcher) matchHeaders(msg *entities.Message) bool {
	if m.fromTime != nil && msg.Timestamp.Before(*m.fromTime) {
		return false
	}
	if m.toTime != nil && msg.Timestamp.After(*m.toTime) {
		return false
	}
	if m.headerName == "" {
		return true
	}
	for name, value := range msg.Headers {
		if !strings.EqualFold(name, m.headerName) {
			continue
		}
		if m.headerValue == "" || value == m.headerValue || slices.Contains(strings.Split(value, ", "), m.headerValue) {
			return true
		}
	}
	return false
}

// matchPayload checks the text condition against the stored payload, and against the decoded one when the stored
// payload does not match.
func (m *searchMatcher) matchPayload(msg *entities.Message, decoded func() string) bool {
	if !m.needsPayload() {
		return true
	}
	if raw, err := base64.StdEncoding.DecodeString(msg.DataBase64); err == nil && m.matchText(string(raw)) {
		return true
	}
	text := decoded()
	return text != "" && m.matchText(text)
}

// searchPlan is the resolved range and settings of one run.
type searchPlan struct {
	rangeFirst, rangeLast uint64
	from, to              uint64
	backward              bool
	fetchMethod           string
	maxPayload            int32
	detect                bool
}

func (s *Service) planSearch(ctx context.Context, in *entities.MessageSearchRequest) (*searchPlan, error) {
	opts := s.buildOptions(ctx, &entities.MessageListRequest{Direction: in.Direction, MaxPayloadBytes: in.MaxPayloadBytes})
	plan := &searchPlan{
		backward:    opts.Direction != "forward",
		fetchMethod: opts.FetchMethod,
		maxPayload:  opts.MaxPayloadBytes,
		detect:      s.detectsTypes(ctx),
	}

	info, err := s.natsService.GetStreamInfo(ctx, in.ConnectionID, in.StreamName)
	if err != nil {
		return nil, err
	}
	if info.State == nil || info.State.Msgs == 0 {
		return plan, nil
	}
	if info.Config.Retention == entities.RetentionWorkQueue {
		plan.fetchMethod = "direct"
	}

	lo, hi := info.State.FirstSeq, info.State.LastSeq
	if in.FromSeq != nil {
		lo = max(lo, *in.FromSeq)
	}
	if in.ToSeq != nil {
		hi = min(hi, *in.ToSeq)
	}
	if in.FromTime != nil {
		seq, err := s.natsService.SeqAtTime(ctx, in.ConnectionID, in.StreamName, plan.fetchMethod, *in.FromTime)
		if err != nil {
			return nil, err
		}
		lo = max(lo, seq)
	}
	if in.ToTime != nil {
		seq, err := s.natsService.SeqAtTime(ctx, in.ConnectionID, in.StreamName, plan.fetchMethod, in.ToTime.Add(time.Nanosecond))
		if err != nil {
			return nil, err
		}
		if seq == 0 {
			return plan, nil
		}
		hi = min(hi, seq-1)
	}
	if lo == 0 || lo > hi {
		return plan, nil
	}

	plan.rangeFirst, plan.rangeLast = lo, hi
	plan.from, plan.to = lo, hi
	if in.CursorSeq != nil {
		if plan.backward {
			plan.to = min(hi, *in.CursorSeq)
		} else {
			plan.from = max(lo, *in.CursorSeq)
		}
	}
	return plan, nil
}

// searchRun tracks one run and streams its events.
type searchRun struct {
	s       *Service
	ctx     context.Context
	in      *entities.MessageSearchRequest
	plan    *searchPlan
	matcher *searchMatcher
	emit    func(*entities.MessageSearchEvent) error

	started      time.Time
	lastProgress time.Time
	scanned      uint64
	matched      uint64
	current      uint64
	resume       uint64
	pending      []*entities.Message
	unmapped     map[string]bool
}

// Search reads the stream in budgeted runs and streams progress, batches of matches and a summary through emit.
func (s *Service) Search(ctx context.Context, in *entities.MessageSearchRequest, emit func(*entities.MessageSearchEvent) error) error {
	matcher, err := newSearchMatcher(in)
	if err != nil {
		return err
	}
	plan, err := s.planSearch(ctx, in)
	if err != nil {
		return err
	}

	run := &searchRun{s: s, ctx: ctx, in: in, plan: plan, matcher: matcher, emit: emit, started: searchClock(), unmapped: map[string]bool{}}
	run.resume = plan.from
	if plan.backward {
		run.resume = plan.to
	}
	if err := run.progress(); err != nil {
		return err
	}
	if plan.rangeFirst == 0 || plan.from > plan.to {
		return run.done(entities.SearchStopComplete, 0)
	}
	if plan.backward {
		return run.backward()
	}
	return run.forward()
}

func (r *searchRun) scan(from, to uint64, visit func(*entities.Message) bool) error {
	opts := entities.ScanOptions{SubjectFilter: r.in.SubjectFilter, FromSeq: from, ToSeq: to, FetchMethod: r.plan.fetchMethod}
	for msg, err := range r.s.natsService.ScanMessages(r.ctx, r.in.ConnectionID, r.in.StreamName, opts) {
		if err != nil {
			return err
		}
		r.scanned++
		r.current = msg.Sequence
		if !visit(msg) {
			return nil
		}
	}
	return r.ctx.Err()
}

// accept reports whether msg matches, decoding it for the check only when the stored payload does not.
func (r *searchRun) accept(msg *entities.Message) bool {
	if !r.matcher.matchHeaders(msg) {
		return false
	}
	return r.matcher.matchPayload(msg, func() string {
		if r.unmapped[msg.Subject] {
			return ""
		}
		r.s.protoService.DecodeMessages(r.ctx, []*entities.Message{msg}, false)
		if msg.Decoded == nil && msg.DecodeError == "" && len(r.unmapped) < searchSubjectCache {
			r.unmapped[msg.Subject] = true
		}
		return string(msg.Decoded)
	})
}

// prepare decodes and trims a match for display.
func (r *searchRun) prepare(msg *entities.Message) *entities.Message {
	if msg.Decoded == nil && (r.plan.maxPayload <= 0 || int64(msg.DataSize) <= int64(r.plan.maxPayload)*skipDecodeMultiplier) {
		r.s.protoService.DecodeMessages(r.ctx, []*entities.Message{msg}, r.plan.detect)
	}
	if r.plan.maxPayload > 0 {
		truncateMessages([]*entities.Message{msg}, int(r.plan.maxPayload))
	}
	return msg
}

func (r *searchRun) maxMatches() uint64 {
	if r.in.MaxMatches > 0 && r.in.MaxMatches < searchMaxMatches {
		return uint64(r.in.MaxMatches)
	}
	return searchMaxMatches
}

func (r *searchRun) overBudget() entities.SearchStopReason {
	switch {
	case r.scanned >= searchMaxScanned:
		return entities.SearchStopScanLimit
	case searchClock().Sub(r.started) >= searchMaxDuration:
		return entities.SearchStopTimeLimit
	default:
		return entities.SearchStopUnspecified
	}
}

func (r *searchRun) forward() error {
	reason := entities.SearchStopComplete
	var next uint64
	var emitErr error
	err := r.scan(r.plan.from, r.plan.to, func(msg *entities.Message) bool {
		if r.accept(msg) {
			r.pending = append(r.pending, r.prepare(msg))
			r.matched++
		}
		r.resume = msg.Sequence + 1
		if emitErr = r.flush(false); emitErr != nil {
			return false
		}
		stop := r.overBudget()
		if r.matched >= r.maxMatches() {
			stop = entities.SearchStopMatchLimit
		}
		if stop == entities.SearchStopUnspecified {
			return true
		}
		reason = stop
		if msg.Sequence < r.plan.to {
			next = msg.Sequence + 1
		} else {
			reason = entities.SearchStopComplete
		}
		return false
	})
	if emitErr != nil {
		return emitErr
	}
	if err != nil {
		return err
	}
	return r.done(reason, next)
}

// backward reads descending windows, each oldest first, and sends a window's matches newest first once it is read.
// A budget reached inside a window drops that window, which the next run reads again; the first window of a run
// always completes, so every run makes progress.
func (r *searchRun) backward() error {
	window := uint64(searchFirstWindow)
	for winHi, first := r.plan.to, true; ; first = false {
		winLo := r.plan.from
		if winHi-r.plan.from+1 > window {
			winLo = winHi - window + 1
		}
		r.resume = winHi

		keep := int(r.maxMatches() - r.matched)
		var found []*entities.Message
		dropped := false
		windowStarted := searchClock()
		stop := entities.SearchStopUnspecified
		var emitErr error
		err := r.scan(winLo, winHi, func(msg *entities.Message) bool {
			if r.accept(msg) {
				found = append(found, r.prepare(msg))
				if len(found) > keep {
					found = found[1:]
					dropped = true
				}
			}
			if !first {
				if stop = r.overBudget(); stop != entities.SearchStopUnspecified {
					return false
				}
			}
			emitErr = r.flush(false)
			return emitErr == nil
		})
		if emitErr != nil {
			return emitErr
		}
		if err != nil {
			return err
		}
		if stop != entities.SearchStopUnspecified {
			return r.done(stop, winHi)
		}

		slices.Reverse(found)
		r.pending = append(r.pending, found...)
		r.matched += uint64(len(found))
		r.resume = winLo - 1

		if dropped {
			return r.done(entities.SearchStopMatchLimit, found[len(found)-1].Sequence-1)
		}
		if winLo == r.plan.from {
			return r.done(entities.SearchStopComplete, 0)
		}
		if r.matched >= r.maxMatches() {
			return r.done(entities.SearchStopMatchLimit, winLo-1)
		}
		if stop := r.overBudget(); stop != entities.SearchStopUnspecified {
			return r.done(stop, winLo-1)
		}
		if err := r.flush(true); err != nil {
			return err
		}
		window = r.nextWindow(window, windowStarted, winHi-winLo+1)
		winHi = winLo - 1
	}
}

// nextWindow grows a window that read fast and keeps the next one within what is left of both budgets.
func (r *searchRun) nextWindow(window uint64, started time.Time, covered uint64) uint64 {
	took := searchClock().Sub(started)
	if took < searchFastWindow {
		window = min(window*searchWindowGrowth, searchMaxWindow)
	}
	if r.scanned < searchMaxScanned {
		window = min(window, searchMaxScanned-r.scanned)
	}
	if perSeq := took / time.Duration(covered); perSeq > 0 {
		left := searchMaxDuration - searchClock().Sub(r.started)
		window = min(window, max(uint64(left/perSeq), 1))
	}
	return max(window, 1)
}

// flush sends the pending matches in full batches and, when due or forced, all of them followed by progress.
func (r *searchRun) flush(force bool) error {
	for len(r.pending) >= searchMatchBatch {
		if err := r.emit(&entities.MessageSearchEvent{Matches: r.pending[:searchMatchBatch]}); err != nil {
			return err
		}
		r.pending = r.pending[searchMatchBatch:]
	}
	if !force && searchClock().Sub(r.lastProgress) < searchProgressTick {
		return nil
	}
	return r.progress()
}

// progress sends every pending match, then where the search stands, so a stop resumes without gaps or repeats.
func (r *searchRun) progress() error {
	if len(r.pending) > 0 {
		if err := r.emit(&entities.MessageSearchEvent{Matches: r.pending}); err != nil {
			return err
		}
		r.pending = nil
	}
	r.lastProgress = searchClock()
	resume := r.resume
	if resume < r.plan.rangeFirst || resume > r.plan.rangeLast {
		resume = 0
	}
	return r.emit(&entities.MessageSearchEvent{Progress: &entities.MessageSearchProgress{
		Scanned:    r.scanned,
		Matched:    r.matched,
		CurrentSeq: r.current,
		RangeFirst: r.plan.rangeFirst,
		RangeLast:  r.plan.rangeLast,
		ResumeSeq:  resume,
	}})
}

func (r *searchRun) done(reason entities.SearchStopReason, next uint64) error {
	if next < r.plan.rangeFirst || next > r.plan.rangeLast {
		next = 0
	}
	r.resume = next
	if err := r.flush(true); err != nil {
		return err
	}
	return r.emit(&entities.MessageSearchEvent{Done: &entities.MessageSearchDone{
		Scanned:    r.scanned,
		Matched:    r.matched,
		Reason:     reason,
		RangeFirst: r.plan.rangeFirst,
		RangeLast:  r.plan.rangeLast,
		NextSeq:    next,
	}})
}
