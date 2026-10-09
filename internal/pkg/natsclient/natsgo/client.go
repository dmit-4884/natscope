// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"cmp"
	"context"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/pkg/natsclient"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

const (
	DefaultTimeout           = 30 * time.Second
	DefaultMessageLimit      = 50
	DefaultMaxMessageLimit   = 500
	DefaultStreamConcurrency = 10
	DefaultSearchRange       = 5000
	DefaultDirection         = "backward"

	// Server's ephemeral-consumer idle TTL; reused as the post-CreateConsumer
	// async-error wait so a permissions violation isn't missed.
	ephemeralConsumerInactiveThreshold = 30 * time.Second

	// kvOperationTimeout caps single KV reads/list ops as a safety net against an
	// unresponsive server.
	kvOperationTimeout = 3 * time.Second

	// browseAsyncErrorWait is head-room after a fetch timeout for the async handler
	// to surface an out-of-band failure (permissions) before reporting timeout.
	browseAsyncErrorWait = 10 * time.Second

	// Bounds clamping the auto-derived 2×NumCPU parallel-GetMsg concurrency.
	fetchConcurrencyMin = 8
	fetchConcurrencyMax = 32
)

// ioBoundConcurrencyRatio is workers-per-CPU for I/O-bound jobs.
const ioBoundConcurrencyRatio = 2

// optimalFetchConcurrency returns the parallelism for parallel GetMsg fetches.
func optimalFetchConcurrency() int {
	n := runtime.NumCPU() * ioBoundConcurrencyRatio
	if n < fetchConcurrencyMin {
		return fetchConcurrencyMin
	}
	if n > fetchConcurrencyMax {
		return fetchConcurrencyMax
	}
	return n
}

// Client is a single live NATS connection wrapping both the legacy and new
// JetStream APIs. It implements natsclient.Client; every method returns errors
// already translated to internal/errs domain sentinels via wrapErr (JetStream
// paths through the watched decorator, other paths explicitly).
type Client struct {
	id        string
	conn      *nats.Conn
	jetStream jetstream.JetStream
	url       string
	api       string

	// permWatch correlates out-of-band async errors (e.g. permissions
	// violations) with in-flight requests so they fail fast instead of
	// blocking until their deadline.
	permWatch *PermissionWatcher

	// logger is the structured logger with client context.
	logger *slog.Logger

	defaultTimeout time.Duration

	// putObjectLocks serializes PutObject per (bucket, name).
	putObjectLocks sync.Map

	// ownConsumers holds the names of the short-lived consumers this client created to read streams, and
	// ownConsumerSeries the name prefixes of the ordered consumers it reads live streams with.
	ownConsumers      sync.Map
	ownConsumerSeries sync.Map

	// remoteLevel is the API level of the JetStream behind api, once an AccountInfo has reported it; levelRefused
	// records that the server refused the AccountInfo, so it is not asked again.
	levelMu      sync.Mutex
	remoteLevel  *int32
	levelRefused bool

	// link is the connection state the connection's handlers report, so a status read never waits for the lock a
	// reconnect attempt holds while it dials.
	link atomic.Int32
}

// Connection states a Client keeps in link; linkUnknown until the first handler or the dial reports one.
const (
	linkUnknown int32 = iota
	linkUp
	linkDown
	linkClosed
)

var _ natsclient.Client = (*Client)(nil)

// URL returns the NATS server URL the client dialed.
func (c *Client) URL() string { return c.url }

// IsConnected reports whether the underlying connection is currently up.
func (c *Client) IsConnected() bool {
	if state := c.link.Load(); state != linkUnknown {
		return state == linkUp
	}
	return c.conn.IsConnected()
}

// IsReconnecting reports whether the client is mid-reconnect.
func (c *Client) IsReconnecting() bool {
	if state := c.link.Load(); state != linkUnknown {
		return state == linkDown
	}
	return c.conn.IsReconnecting()
}

// Status returns the connection status as a string.
func (c *Client) Status() string {
	switch c.link.Load() {
	case linkUp:
		return nats.CONNECTED.String()
	case linkDown:
		return nats.RECONNECTING.String()
	case linkClosed:
		return nats.CLOSED.String()
	default:
		return c.conn.Status().String()
	}
}

// Close closes the underlying connection; idempotent.
func (c *Client) Close() { c.conn.Close() }

// RTT measures round-trip time to the server via a flush.
func (c *Client) RTT() (time.Duration, error) {
	start := time.Now()
	if err := c.conn.Flush(); err != nil {
		return 0, wrapErr(err)
	}
	return time.Since(start), nil
}

// takeAsyncError returns the last uncorrelated async error if it occurred
// within the given window, wrapped to domain sentinels; the fallback for
// operations not going through the watched JetStream client.
func (c *Client) takeAsyncError(within time.Duration) error {
	if err := c.permWatch.TakeRecent(within); err != nil {
		return wrapErr(err)
	}
	return nil
}

// apiSubject returns a JetStream API subject on the client's API prefix, e.g. apiSubject("STREAM.LIST").
func (c *Client) apiSubject(suffix string) string {
	return cmp.Or(c.api, defaultAPIPrefix) + "." + suffix
}

// request performs a raw JetStream API request bounded by the default timeout
// (core NATS requests have no SDK-side default) and watched for out-of-band
// permissions violations.
func (c *Client) request(ctx context.Context, subject string, data []byte) (*nats.Msg, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, DefaultTimeout)
	defer cancel()
	return c.permWatch.Request(ctx, c.conn, subject, data, apiSubjects(c.api, subject))
}
