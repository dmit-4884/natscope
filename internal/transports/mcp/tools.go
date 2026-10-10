// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package mcptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/domain/normalizer"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"

	grpchelpers "github.com/dmit-4884/natscope/internal/transports/grpc/helpers"
)

var schemaOptions = &jsonschema.ForOptions{
	TypeSchemas: map[reflect.Type]*jsonschema.Schema{
		reflect.TypeFor[json.RawMessage](): {},
	},
}

// AddTool registers a typed tool; its input is normalized by `normalize` tags, json.RawMessage fields accept any JSON and handler errors reach
// the agent as sanitized, reason-coded messages.
func AddTool[In, Out any](s *mcp.Server, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	t.InputSchema = mustSchema[In]()
	t.OutputSchema = mustSchema[Out]()
	mcp.AddTool(s, t, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var zero Out
		if err := normalizer.Normalize(&in); err != nil {
			return nil, zero, toolError(ctx, err)
		}
		res, out, err := h(ctx, req, in)
		if err != nil {
			return nil, zero, toolError(ctx, err)
		}
		return res, out, nil
	})
}

func mustSchema[T any]() *jsonschema.Schema {
	s, err := jsonschema.For[T](schemaOptions)
	if err != nil {
		panic(fmt.Sprintf("mcp: schema for %s: %v", reflect.TypeFor[T](), err))
	}
	return s
}

type agentError struct {
	msg string
}

func (e *agentError) Error() string { return e.msg }

// Errorf builds an error the agent sees verbatim; any other handler error is sanitized first.
func Errorf(format string, args ...any) error {
	return &agentError{msg: fmt.Sprintf(format, args...)}
}

func toolError(ctx context.Context, err error) error {
	if ae, ok := errors.AsType[*agentError](err); ok {
		return ae
	}
	st := status.Convert(grpchelpers.StatusErrorConvert(ctx, err))
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info.GetReason() != "" {
			return fmt.Errorf("%s [%s]", st.Message(), info.GetReason())
		}
	}
	return errors.New(st.Message())
}
