// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

func (t *Toolset) listTypes(ctx context.Context, _ *mcp.CallToolRequest, in listTypesInput) (*mcp.CallToolResult, listTypesOutput, error) {
	filter := strings.ToLower(strings.TrimSpace(in.Filter))
	var matched []entities.ProtoMessageInfo
	for _, m := range t.registry.ListMessages(ctx) {
		if strings.Contains(strings.ToLower(m.FullName), filter) {
			matched = append(matched, m)
		}
	}
	shown := matched[:min(len(matched), mcptransport.Limit(in.Limit, defaultTypesLimit, maxTypesLimit))]
	return nil, listTypesOutput{
		Types: mcptransport.Items(slices.To(shown, func(m entities.ProtoMessageInfo) typeView { return *converter.Convert(&m, &typeView{}) })),
		Total: len(matched),
	}, nil
}

func (t *Toolset) describeType(ctx context.Context, _ *mcp.CallToolRequest, in typeInput) (*mcp.CallToolResult, describeOutput, error) {
	info, err := mcptransport.LookupType(ctx, t.registry, in.Type, in.SourceID)
	if err != nil {
		return nil, describeOutput{}, err
	}
	detail, err := t.registry.GetMessage(ctx, info.SourceID, info.FullName)
	if err != nil {
		return nil, describeOutput{}, err
	}
	out := describeOutput{
		typeView: *converter.Convert(detail, &typeView{}),
		Package:  detail.Package,
		Fields:   slices.To(detail.Fields, func(f *entities.ProtoField) fieldView { return *converter.Convert(f, &fieldView{}) }),
	}
	if example, exErr := t.registry.GenerateExample(ctx, info.SourceID, info.FullName); exErr == nil {
		out.Example = example
	}
	return nil, out, nil
}

func (t *Toolset) resolveSubject(ctx context.Context, _ *mcp.CallToolRequest, in subjectInput) (*mcp.CallToolResult, resolveOutput, error) {
	subject := strings.TrimSpace(in.Subject)
	out := resolveOutput{Subject: subject}
	m := t.mappings.Resolver(ctx).Resolve(subject)
	if m == nil {
		return nil, out, nil
	}
	out.Mapped, out.Mapping = true, converter.Convert(m, &mappingView{})
	health, err := t.registry.MappingHealth(ctx, []string{m.Id})
	if err != nil {
		return nil, resolveOutput{}, err
	}
	if len(health) > 0 {
		out.Health, out.HealthDetail = string(health[0].Health), health[0].Detail
	}
	return nil, out, nil
}

func (t *Toolset) listMappings(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, listMappingsOutput, error) {
	all, err := t.mappings.GetAll(ctx)
	if err != nil {
		return nil, listMappingsOutput{}, err
	}
	return nil, listMappingsOutput{Mappings: mcptransport.Items(slices.To(all, func(m *entities.SubjectMapping) mappingView {
		return *converter.Convert(m, &mappingView{})
	}))}, nil
}

func (t *Toolset) decode(ctx context.Context, _ *mcp.CallToolRequest, in decodeInput) (*mcp.CallToolResult, decodeOutput, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(in.Base64))
	if err != nil {
		return nil, decodeOutput{}, mcptransport.Errorf("base64 is not valid standard base64: %v", err)
	}

	var (
		res         *entities.DecodeResult
		messageType string
	)
	switch {
	case strings.TrimSpace(in.Type) != "":
		info, lookupErr := mcptransport.LookupType(ctx, t.registry, in.Type, in.SourceID)
		if lookupErr != nil {
			return nil, decodeOutput{}, lookupErr
		}
		messageType = info.FullName
		res, err = t.codec.Decode(ctx, entities.CodecRequest{Data: data, SourceID: info.SourceID, MessageType: info.FullName})
	case strings.TrimSpace(in.Subject) != "":
		m := t.mappings.Resolver(ctx).Resolve(strings.TrimSpace(in.Subject))
		if m == nil {
			return nil, decodeOutput{}, mcptransport.Errorf("subject %q has no mapping; pass type instead", in.Subject)
		}
		messageType = m.MessageType
		res, err = t.codec.DecodeForMapping(ctx, data, m)
	default:
		return nil, decodeOutput{}, mcptransport.Errorf("pass type or subject")
	}
	if err != nil {
		return nil, decodeOutput{}, err
	}
	out := decodeOutput{MessageType: cmp.Or(res.MessageType, messageType), Decoded: res.Decoded, Error: res.Error}
	return nil, out, nil
}

func (t *Toolset) validate(ctx context.Context, _ *mcp.CallToolRequest, in validateInput) (*mcp.CallToolResult, validateOutput, error) {
	info, err := mcptransport.LookupType(ctx, t.registry, in.Type, in.SourceID)
	if err != nil {
		return nil, validateOutput{}, err
	}
	payload, err := json.Marshal(in.Payload)
	if err != nil {
		return nil, validateOutput{}, mcptransport.Errorf("payload is not serializable JSON: %v", err)
	}
	res, err := t.codec.ValidateJSON(ctx, entities.CodecRequest{JSON: payload, SourceID: info.SourceID, MessageType: info.FullName})
	if err != nil {
		return nil, validateOutput{}, err
	}
	return nil, *converter.Convert(res, &validateOutput{}), nil
}

func (t *Toolset) status(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, statusOutput, error) {
	conflicts, err := t.registry.ListSchemaConflicts(ctx)
	if err != nil {
		return nil, statusOutput{}, err
	}
	out := statusOutput{
		Conflicts: slices.To(conflicts, func(c *entities.SchemaConflict) conflictView { return *converter.Convert(c, &conflictView{}) }),
	}
	if stats := t.registry.Stats(ctx); stats != nil {
		out.MessageTypes, out.Error = stats.MessagesCount, stats.Error
	}
	return nil, out, nil
}
