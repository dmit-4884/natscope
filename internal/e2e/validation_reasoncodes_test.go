// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"

	badrequestv1 "github.com/altessa-s/proto-gen-go/io/altessa/badrequest/v1"
	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
)

// TestValidationReasonCodes proves the reasoncodes resolver is wired end-to-end:
// a rejected request carries a canonical, client-facing reason code in its
// BadRequest detail — not the raw protovalidate rule ID.
func TestValidationReasonCodes(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	cases := []struct {
		name     string
		call     func() error
		wantCode string
	}{
		{
			"standard required rule → FIELD_REQUIRED",
			func() error {
				_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
					Urls: []string{"nats://127.0.0.1:1"},
				}))
				return err
			},
			"NAME_REQUIRED",
		},
		{
			"blank name → same code as a missing name",
			func() error {
				_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
					Name: "   ", Urls: []string{"nats://127.0.0.1:1"},
				}))
				return err
			},
			"NAME_REQUIRED",
		},
		{
			"empty name on update → same code as a missing name",
			func() error {
				_, err := env.connections.UpdateConnection(ctx, connect.NewRequest(&connectionspb.UpdateConnectionRequest{
					Id: unknownUUID, Name: new(""),
				}))
				return err
			},
			"NAME_REQUIRED",
		},
		{
			"message_type without source_id → catalog code",
			func() error {
				_, err := env.publish.PublishMessage(ctx, connect.NewRequest(&publishpb.PublishMessageRequest{
					ConnectionId: "x", Subject: "a", Data: "{}", MessageType: new("pkg.T"),
				}))
				return err
			},
			"SOURCE_ID_REQUIRED",
		},
		{
			"domain CEL rule → catalog code",
			func() error {
				_, err := env.messages.ListMessages(ctx, connect.NewRequest(&messagespb.ListMessagesRequest{
					ConnectionId: "x", StreamName: "x",
					StartSeq:  new(uint64(1)),
					StartTime: timestamppb.New(time.Now()),
				}))
				return err
			},
			"MUTUALLY_EXCLUSIVE_FIELDS",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			assert.Contains(t, violationCodes(t, err), tc.wantCode,
				"canonical reason code should reach the client")
		})
	}
}

// violationCodes extracts every FieldViolation.Code from a connect error's
// BadRequest details.
func violationCodes(t *testing.T, err error) []string {
	t.Helper()
	var connErr *connect.Error
	require.ErrorAs(t, err, &connErr)

	var codes []string
	for _, detail := range connErr.Details() {
		msg, valErr := detail.Value()
		if valErr != nil {
			continue
		}
		br, ok := msg.(*badrequestv1.BadRequest)
		if !ok {
			continue
		}
		for _, fv := range br.GetFieldViolations() {
			if code := fv.GetCode(); code != "" {
				codes = append(codes, code)
			}
		}
	}
	return codes
}
