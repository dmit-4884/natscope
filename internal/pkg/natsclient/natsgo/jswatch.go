// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// fallbackWindow is how long an uncorrelated async error is still considered
// the real cause of a timed-out operation.
const fallbackWindow = 10 * time.Second

// JetStream API subject templates the watcher listens on, per operation.
// Templates ending in "." are token-boundary prefixes (the consumer name or
// filter subject follows); the rest match exactly.
const (
	subjStreamCreate          = "$JS.API.STREAM.CREATE.%s"
	subjStreamUpdate          = "$JS.API.STREAM.UPDATE.%s"
	subjStreamDelete          = "$JS.API.STREAM.DELETE.%s"
	subjStreamInfo            = "$JS.API.STREAM.INFO.%s"
	subjStreamPurge           = "$JS.API.STREAM.PURGE.%s"
	subjStreamNames           = "$JS.API.STREAM.NAMES"
	subjStreamList            = "$JS.API.STREAM.LIST"
	subjConsumerCreate        = "$JS.API.CONSUMER.CREATE.%s."
	subjConsumerDurableCreate = "$JS.API.CONSUMER.DURABLE.CREATE.%s."
	subjConsumerInfo          = "$JS.API.CONSUMER.INFO.%s."
	subjConsumerDelete        = "$JS.API.CONSUMER.DELETE.%s."
	subjConsumerNames         = "$JS.API.CONSUMER.NAMES.%s"
	subjConsumerList          = "$JS.API.CONSUMER.LIST.%s"
	subjMsgGet                = "$JS.API.MSG.GET.%s"
	subjMsgDelete             = "$JS.API.MSG.DELETE.%s"
	subjDirectGet             = "$JS.API.DIRECT.GET.%s"
	subjDirectGetPrefix       = "$JS.API.DIRECT.GET.%s."
)

// subjectsFor renders subject templates for one stream name.
func subjectsFor(name string, templates ...string) []string {
	subjects := make([]string, 0, len(templates))
	for _, t := range templates {
		subjects = append(subjects, fmt.Sprintf(t, name))
	}
	return subjects
}

// watchJetStream wraps js so that every JetStream API call fails fast on an
// out-of-band permissions violation for its API subject, and every error is
// translated through wrapErr exactly once, at this boundary. Streams,
// consumers, listers and fetch batches returned by the wrapped client are
// wrapped transitively.
func watchJetStream(js jetstream.JetStream, pw *PermissionWatcher) jetstream.JetStream {
	return &jetStreamWatch{JetStream: js, pw: pw}
}

// jetStreamWatch decorates [jetstream.JetStream]; methods not overridden pass
// through unwatched.
type jetStreamWatch struct {
	jetstream.JetStream
	pw *PermissionWatcher
}

var _ jetstream.JetStream = (*jetStreamWatch)(nil)

// translate resolves an operation error to its real cause: a timeout whose
// origin is an uncorrelated recent permissions violation is replaced by that
// violation, and everything passes through the configured error wrapper.
func (w *jetStreamWatch) translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout) {
		if async := w.pw.TakeRecent(fallbackWindow); async != nil {
			return wrapErr(async)
		}
	}
	return wrapErr(err)
}

// watchCall runs fn under a violation watch for subjects and translates the
// outcome.
func watchCall[T any](ctx context.Context, w *jetStreamWatch, subjects []string, fn func(ctx context.Context) (T, error)) (T, error) {
	var out T
	err := w.pw.Watch(ctx, subjects, func(ctx context.Context) error {
		var fnErr error
		out, fnErr = fn(ctx)
		return fnErr
	})
	return out, w.translate(err)
}

// stream wraps a returned stream handle so its own API calls are watched too.
func (w *jetStreamWatch) stream(s jetstream.Stream, name string) jetstream.Stream {
	if s == nil {
		return nil
	}
	return &streamWatch{Stream: s, name: name, w: w}
}

// consumer wraps a returned consumer handle so its fetches are translated.
func (w *jetStreamWatch) consumer(c jetstream.Consumer, stream string) jetstream.Consumer {
	if c == nil {
		return nil
	}
	return &consumerWatch{Consumer: c, stream: stream, w: w}
}

func (w *jetStreamWatch) CreateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	s, err := watchCall(ctx, w, subjectsFor(cfg.Name, subjStreamCreate), func(ctx context.Context) (jetstream.Stream, error) {
		return w.JetStream.CreateStream(ctx, cfg)
	})
	return w.stream(s, cfg.Name), err
}

func (w *jetStreamWatch) UpdateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	s, err := watchCall(ctx, w, subjectsFor(cfg.Name, subjStreamUpdate), func(ctx context.Context) (jetstream.Stream, error) {
		return w.JetStream.UpdateStream(ctx, cfg)
	})
	return w.stream(s, cfg.Name), err
}

func (w *jetStreamWatch) CreateOrUpdateStream(ctx context.Context, cfg jetstream.StreamConfig) (jetstream.Stream, error) {
	s, err := watchCall(ctx, w, subjectsFor(cfg.Name, subjStreamCreate, subjStreamUpdate), func(ctx context.Context) (jetstream.Stream, error) {
		return w.JetStream.CreateOrUpdateStream(ctx, cfg)
	})
	return w.stream(s, cfg.Name), err
}

func (w *jetStreamWatch) Stream(ctx context.Context, name string) (jetstream.Stream, error) {
	s, err := watchCall(ctx, w, subjectsFor(name, subjStreamInfo), func(ctx context.Context) (jetstream.Stream, error) {
		return w.JetStream.Stream(ctx, name)
	})
	return w.stream(s, name), err
}

func (w *jetStreamWatch) DeleteStream(ctx context.Context, name string) error {
	err := w.pw.Watch(ctx, subjectsFor(name, subjStreamDelete), func(ctx context.Context) error {
		return w.JetStream.DeleteStream(ctx, name)
	})
	return w.translate(err)
}

func (w *jetStreamWatch) ListStreams(ctx context.Context, opts ...jetstream.StreamListOpt) jetstream.StreamInfoLister {
	return &streamInfoLister{StreamInfoLister: w.JetStream.ListStreams(ctx, opts...), w: w}
}

func (w *jetStreamWatch) StreamNames(ctx context.Context, opts ...jetstream.StreamListOpt) jetstream.StreamNameLister {
	return &streamNameLister{StreamNameLister: w.JetStream.StreamNames(ctx, opts...), w: w}
}

func (w *jetStreamWatch) Publish(ctx context.Context, subject string, payload []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	return watchCall(ctx, w, []string{subject}, func(ctx context.Context) (*jetstream.PubAck, error) {
		return w.JetStream.Publish(ctx, subject, payload, opts...)
	})
}

func (w *jetStreamWatch) PublishMsg(ctx context.Context, msg *nats.Msg, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	return watchCall(ctx, w, []string{msg.Subject}, func(ctx context.Context) (*jetstream.PubAck, error) {
		return w.JetStream.PublishMsg(ctx, msg, opts...)
	})
}

// streamWatch decorates [jetstream.Stream] for one named stream.
type streamWatch struct {
	jetstream.Stream
	name string
	w    *jetStreamWatch
}

var _ jetstream.Stream = (*streamWatch)(nil)

func (s *streamWatch) Info(ctx context.Context, opts ...jetstream.StreamInfoOpt) (*jetstream.StreamInfo, error) {
	return watchCall(ctx, s.w, subjectsFor(s.name, subjStreamInfo), func(ctx context.Context) (*jetstream.StreamInfo, error) {
		return s.Stream.Info(ctx, opts...)
	})
}

func (s *streamWatch) Purge(ctx context.Context, opts ...jetstream.StreamPurgeOpt) error {
	err := s.w.pw.Watch(ctx, subjectsFor(s.name, subjStreamPurge), func(ctx context.Context) error {
		return s.Stream.Purge(ctx, opts...)
	})
	return s.w.translate(err)
}

func (s *streamWatch) CreateConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	c, err := watchCall(ctx, s.w, subjectsFor(s.name, subjConsumerCreate, subjConsumerDurableCreate), func(ctx context.Context) (jetstream.Consumer, error) {
		return s.Stream.CreateConsumer(ctx, cfg)
	})
	return s.w.consumer(c, s.name), err
}

func (s *streamWatch) CreateOrUpdateConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	c, err := watchCall(ctx, s.w, subjectsFor(s.name, subjConsumerCreate, subjConsumerDurableCreate), func(ctx context.Context) (jetstream.Consumer, error) {
		return s.Stream.CreateOrUpdateConsumer(ctx, cfg)
	})
	return s.w.consumer(c, s.name), err
}

func (s *streamWatch) UpdateConsumer(ctx context.Context, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	c, err := watchCall(ctx, s.w, subjectsFor(s.name, subjConsumerCreate, subjConsumerDurableCreate), func(ctx context.Context) (jetstream.Consumer, error) {
		return s.Stream.UpdateConsumer(ctx, cfg)
	})
	return s.w.consumer(c, s.name), err
}

func (s *streamWatch) OrderedConsumer(ctx context.Context, cfg jetstream.OrderedConsumerConfig) (jetstream.Consumer, error) {
	c, err := watchCall(ctx, s.w, subjectsFor(s.name, subjConsumerCreate, subjConsumerDurableCreate), func(ctx context.Context) (jetstream.Consumer, error) {
		return s.Stream.OrderedConsumer(ctx, cfg)
	})
	return s.w.consumer(c, s.name), err
}

func (s *streamWatch) Consumer(ctx context.Context, name string) (jetstream.Consumer, error) {
	c, err := watchCall(ctx, s.w, subjectsFor(s.name, subjConsumerInfo), func(ctx context.Context) (jetstream.Consumer, error) {
		return s.Stream.Consumer(ctx, name)
	})
	return s.w.consumer(c, s.name), err
}

func (s *streamWatch) DeleteConsumer(ctx context.Context, name string) error {
	err := s.w.pw.Watch(ctx, subjectsFor(s.name, subjConsumerDelete), func(ctx context.Context) error {
		return s.Stream.DeleteConsumer(ctx, name)
	})
	return s.w.translate(err)
}

func (s *streamWatch) ConsumerNames(ctx context.Context) jetstream.ConsumerNameLister {
	return &consumerNameLister{ConsumerNameLister: s.Stream.ConsumerNames(ctx), w: s.w}
}

func (s *streamWatch) ListConsumers(ctx context.Context) jetstream.ConsumerInfoLister {
	return &consumerInfoLister{ConsumerInfoLister: s.Stream.ListConsumers(ctx), w: s.w}
}

func (s *streamWatch) GetMsg(ctx context.Context, seq uint64, opts ...jetstream.GetMsgOpt) (*jetstream.RawStreamMsg, error) {
	return watchCall(ctx, s.w, subjectsFor(s.name, subjMsgGet, subjDirectGet, subjDirectGetPrefix), func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
		return s.Stream.GetMsg(ctx, seq, opts...)
	})
}

func (s *streamWatch) GetLastMsgForSubject(ctx context.Context, subject string) (*jetstream.RawStreamMsg, error) {
	return watchCall(ctx, s.w, subjectsFor(s.name, subjMsgGet, subjDirectGet, subjDirectGetPrefix), func(ctx context.Context) (*jetstream.RawStreamMsg, error) {
		return s.Stream.GetLastMsgForSubject(ctx, subject)
	})
}

func (s *streamWatch) DeleteMsg(ctx context.Context, seq uint64) error {
	err := s.w.pw.Watch(ctx, subjectsFor(s.name, subjMsgDelete), func(ctx context.Context) error {
		return s.Stream.DeleteMsg(ctx, seq)
	})
	return s.w.translate(err)
}

func (s *streamWatch) SecureDeleteMsg(ctx context.Context, seq uint64) error {
	err := s.w.pw.Watch(ctx, subjectsFor(s.name, subjMsgDelete), func(ctx context.Context) error {
		return s.Stream.SecureDeleteMsg(ctx, seq)
	})
	return s.w.translate(err)
}

// consumerWatch decorates [jetstream.Consumer]; fetch batches report the real
// failure cause instead of a bare timeout.
type consumerWatch struct {
	jetstream.Consumer
	stream string
	w      *jetStreamWatch
}

var _ jetstream.Consumer = (*consumerWatch)(nil)

func (c *consumerWatch) Fetch(batch int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	b, err := c.Consumer.Fetch(batch, opts...)
	return c.wrapBatch(b), c.w.translate(err)
}

func (c *consumerWatch) FetchBytes(maxBytes int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	b, err := c.Consumer.FetchBytes(maxBytes, opts...)
	return c.wrapBatch(b), c.w.translate(err)
}

func (c *consumerWatch) FetchNoWait(batch int) (jetstream.MessageBatch, error) {
	b, err := c.Consumer.FetchNoWait(batch)
	return c.wrapBatch(b), c.w.translate(err)
}

func (c *consumerWatch) Info(ctx context.Context) (*jetstream.ConsumerInfo, error) {
	return watchCall(ctx, c.w, subjectsFor(c.stream, subjConsumerInfo), func(ctx context.Context) (*jetstream.ConsumerInfo, error) {
		return c.Consumer.Info(ctx)
	})
}

func (c *consumerWatch) wrapBatch(b jetstream.MessageBatch) jetstream.MessageBatch {
	if b == nil {
		return nil
	}
	return &batchWatch{MessageBatch: b, w: c.w}
}

// batchWatch translates the terminal batch error so a fetch that silently
// timed out on a permissions violation reports the violation.
type batchWatch struct {
	jetstream.MessageBatch
	w *jetStreamWatch
}

func (b *batchWatch) Error() error {
	return b.w.translate(b.MessageBatch.Error())
}

// Lister decorators: iteration happens after the constructor returns, so the
// watch cannot bracket it; translating Err() still recovers the violation
// retained by the fallback slot.
type streamInfoLister struct {
	jetstream.StreamInfoLister
	w *jetStreamWatch
}

func (l *streamInfoLister) Err() error { return l.w.translate(l.StreamInfoLister.Err()) }

type streamNameLister struct {
	jetstream.StreamNameLister
	w *jetStreamWatch
}

func (l *streamNameLister) Err() error { return l.w.translate(l.StreamNameLister.Err()) }

type consumerNameLister struct {
	jetstream.ConsumerNameLister
	w *jetStreamWatch
}

func (l *consumerNameLister) Err() error { return l.w.translate(l.ConsumerNameLister.Err()) }

type consumerInfoLister struct {
	jetstream.ConsumerInfoLister
	w *jetStreamWatch
}

func (l *consumerInfoLister) Err() error { return l.w.translate(l.ConsumerInfoLister.Err()) }

// Request performs a core NATS request under a permissions watch for its
// subject, so a denied request fails fast instead of waiting out its deadline.
func (pw *PermissionWatcher) Request(ctx context.Context, nc *nats.Conn, subject string, data []byte) (*nats.Msg, error) {
	var msg *nats.Msg
	err := pw.Watch(ctx, []string{subject}, func(ctx context.Context) error {
		var reqErr error
		msg, reqErr = nc.RequestWithContext(ctx, subject, data)
		return reqErr
	})
	return msg, err
}
