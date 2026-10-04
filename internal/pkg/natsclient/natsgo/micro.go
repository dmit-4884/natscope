// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/nats-io/nats.go/micro"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
)

const (
	microDiscoveryTimeout = 2 * time.Second
	microDiscoveryIdle    = 300 * time.Millisecond

	statusHeader       = "Status"
	noRespondersStatus = "503"
)

// MicroInfo collects every NATS Micro instance's answer to $SRV.INFO.
func (c *Client) MicroInfo(ctx context.Context) ([]entities.MicroReport, error) {
	return c.gatherMicro(ctx, micro.InfoVerb, parseMicroInfo)
}

// MicroStats collects every NATS Micro instance's answer to $SRV.STATS.
func (c *Client) MicroStats(ctx context.Context) ([]entities.MicroReport, error) {
	return c.gatherMicro(ctx, micro.StatsVerb, parseMicroStats)
}

func (c *Client) gatherMicro(
	ctx context.Context,
	verb micro.Verb,
	parse func([]byte) (entities.MicroReport, bool),
) ([]entities.MicroReport, error) {
	subject, err := micro.ControlSubject(verb, "", "")
	if err != nil {
		return nil, wrapErr(err)
	}
	replies, err := c.gather(ctx, subject)
	if err != nil {
		return nil, widenInboxDenial(wrapErr(err), c.inboxPrefix())
	}
	reports := make([]entities.MicroReport, 0, len(replies))
	for _, data := range replies {
		if report, ok := parse(data); ok {
			reports = append(reports, report)
		}
	}
	return reports, nil
}

func (c *Client) gather(ctx context.Context, subject string) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, microDiscoveryTimeout)
	defer cancel()

	var replies [][]byte
	err := c.permWatch.Watch(ctx, []string{subject, c.inboxPrefix()}, func(ctx context.Context) error {
		inbox := c.conn.NewRespInbox()
		sub, err := c.conn.SubscribeSync(inbox)
		if err != nil {
			return err
		}
		defer sub.Unsubscribe() //nolint:errcheck // one-shot inbox

		if err := c.conn.PublishRequest(subject, inbox, nil); err != nil {
			return err
		}
		for {
			waitCtx, waitCancel := ctx, context.CancelFunc(func() {})
			if len(replies) > 0 {
				waitCtx, waitCancel = context.WithTimeout(ctx, microDiscoveryIdle)
			}
			msg, err := sub.NextMsgWithContext(waitCtx)
			waitCancel()
			if err != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return ctx.Err()
				}
				return nil
			}
			if msg.Header.Get(statusHeader) == noRespondersStatus {
				return nil
			}
			replies = append(replies, msg.Data)
		}
	})
	if err != nil {
		return nil, err
	}
	return replies, nil
}

func parseMicroInfo(data []byte) (entities.MicroReport, bool) {
	var info micro.Info
	if err := json.Unmarshal(data, &info); err != nil || info.Name == "" {
		return entities.MicroReport{}, false
	}
	report := *converter.Convert(info.ServiceIdentity, &entities.MicroReport{})
	report.Description = info.Description
	report.Endpoints = slices.To(info.Endpoints, func(e micro.EndpointInfo) entities.MicroEndpoint {
		return *converter.Convert(e, &entities.MicroEndpoint{})
	})
	return report, true
}

func parseMicroStats(data []byte) (entities.MicroReport, bool) {
	var stats micro.Stats
	if err := json.Unmarshal(data, &stats); err != nil || stats.Name == "" {
		return entities.MicroReport{}, false
	}
	report := *converter.Convert(stats.ServiceIdentity, &entities.MicroReport{})
	report.Started = stats.Started
	for _, e := range stats.Endpoints {
		if e == nil {
			continue
		}
		endpoint := converter.Convert(e, &entities.MicroEndpoint{})
		endpoint.Stats = converter.Convert(e, &entities.MicroEndpointStats{})
		report.Endpoints = append(report.Endpoints, *endpoint)
	}
	return report, true
}
