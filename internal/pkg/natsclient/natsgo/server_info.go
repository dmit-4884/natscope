// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"log/slog"
	"net"
	"strconv"

	"github.com/dmit-4884/natscope/internal/entities"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// GetServerInfo returns detailed NATS server information.
func (c *Client) GetServerInfo(ctx context.Context) (*entities.ServerInfo, error) {
	info := &entities.ServerInfo{
		ServerId:     c.conn.ConnectedServerId(),
		ServerName:   c.conn.ConnectedServerName(),
		Version:      c.conn.ConnectedServerVersion(),
		ClusterName:  c.conn.ConnectedClusterName(),
		MaxPayload:   c.conn.MaxPayload(),
		ConnectedUrl: c.conn.ConnectedUrl(),
		AuthRequired: c.conn.AuthRequired(),
		TlsRequired:  c.conn.TLSRequired(),
		ConnectUrls:  c.conn.DiscoveredServers(),
	}

	// Parse host:port from connected address
	addr := c.conn.ConnectedAddr()
	if addr != "" {
		host, portStr, splitErr := net.SplitHostPort(addr)
		if splitErr == nil {
			info.Host = host
			port, convErr := strconv.Atoi(portStr)
			if convErr == nil {
				info.Port = int32(port)
			}
		} else {
			info.Host = addr
		}
	}

	// Client connection statistics
	stats := c.conn.Stats()
	info.ClientStats = &entities.ClientStatistics{
		InMsgs:     stats.InMsgs,
		OutMsgs:    stats.OutMsgs,
		InBytes:    stats.InBytes,
		OutBytes:   stats.OutBytes,
		Reconnects: stats.Reconnects,
	}

	// Get JetStream account info
	accountInfo, err := c.jetStream.AccountInfo(ctx)
	if err == nil && accountInfo != nil {
		info.Jetstream = true
		info.JsAccount = &entities.JetStreamAccountInfo{
			Memory:        int64(accountInfo.Memory),
			Storage:       int64(accountInfo.Store),
			Streams:       int64(accountInfo.Streams),
			Consumers:     int64(accountInfo.Consumers),
			MemoryLimit:   accountInfo.Limits.MaxMemory,
			StorageLimit:  accountInfo.Limits.MaxStore,
			StreamLimit:   int64(accountInfo.Limits.MaxStreams),
			ConsumerLimit: int64(accountInfo.Limits.MaxConsumers),
			ApiTotal:      accountInfo.API.Total,
			ApiErrors:     accountInfo.API.Errors,
			Domain:        accountInfo.Domain,
		}
	} else {
		c.logger.Debug("JetStream account info unavailable",
			slog.String("connection_id", c.id), slogx.Error(err))
	}

	_, apiLevel := c.conn.ConnectedServerJetStream()
	info.Capabilities = computeCapabilities(info.Version, info.Jetstream, apiLevel)

	return info, nil
}
