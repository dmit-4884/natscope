// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/dmit-4884/natscope/internal/entities"
)

// TestConnection tests a NATS connection without saving or pooling it.
func (d *Dialer) TestConnection(
	_ context.Context,
	in *entities.TestConnectionRequest,
) (*entities.TestConnectionResult, error) {
	result := &entities.TestConnectionResult{}

	natsOpts, err := buildTestOptions(in)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	url := strings.Join(in.URLs, ",")

	conn, err := nats.Connect(url, natsOpts...)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}
	defer conn.Close()

	result.Success = true
	result.ConnectedURL = conn.ConnectedUrl()
	result.ServerVersion = conn.ConnectedServerVersion()
	result.ServerName = conn.ConnectedServerName()
	result.ServerID = conn.ConnectedServerId()
	result.ClusterName = conn.ConnectedClusterName()
	result.MaxPayload = conn.MaxPayload()
	result.DiscoveredServers = conn.DiscoveredServers()

	// Measure RTT
	start := time.Now()
	if flushErr := conn.Flush(); flushErr == nil {
		result.RTTMs = time.Since(start).Milliseconds()
	}

	// Check JetStream availability
	js, err := conn.JetStream()
	if err == nil {
		_, err = js.AccountInfo()
		result.JetstreamEnabled = err == nil
	}

	return result, nil
}
