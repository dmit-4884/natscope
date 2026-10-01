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
	all, err := mcptransport.MessageTypes(ctx, t.registry)
	if err != nil {
		return nil, listTypesOutput{}, err
	}
	filter := strings.ToLower(strings.TrimSpace(in.Filter))
	var matched []entities.SchemaType
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.FullName), filter) {
			matched = append(matched, m)
		}
	}
	shown := matched[:min(len(matched), mcptransport.Limit(in.Limit, defaultTypesLimit, maxTypesLimit))]
	return nil, listTypesOutput{
		Types: mcptransport.Items(slices.To(shown, func(m entities.SchemaType) typeView { return *converter.Convert(&m, &typeView{}) })),
		Total: len(matched),
	}, nil
}

func (t *Toolset) describeType(ctx context.Context, _ *mcp.CallToolRequest, in typeInput) (*mcp.CallToolResult, describeOutput, error) {
	info, err := mcptransport.LookupType(ctx, t.registry, in.Type, in.SourceID)
	if err != nil {
		return nil, describeOutput{}, err
	}
	desc, err := t.registry.DescribeType(ctx, info.SourceID, "", info.FullName, true)
	if err != nil {
		return nil, describeOutput{}, err
	}
	root := converter.Convert(desc.Messages[0], &messageView{})
	out := describeOutput{
		typeView: *converter.Convert(&info, &typeView{}),
		Package:  info.Package,
		Fields:   root.Fields,
		Related:  slices.To(desc.Messages[1:], func(m *entities.SchemaMessage) messageView { return *converter.Convert(m, &messageView{}) }),
		Enums:    slices.To(desc.Enums, func(e *entities.SchemaEnum) enumView { return *converter.Convert(e, &enumView{}) }),
	}
	out.Comment = root.Comment
	if example, exErr := t.registry.GenerateExample(ctx, info.SourceID, "", info.FullName); exErr == nil {
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
	out.Mapped, out.Mapping = true, toMappingView(m)
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
		return *toMappingView(m)
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
		dump := t.codec.DecodeWire(data)
		return nil, decodeOutput{Wire: wireViews(dump.Fields), ValidBytes: dump.ValidBytes, Error: dump.Error}, nil
	}
	if err != nil {
		return nil, decodeOutput{}, err
	}
	out := decodeOutput{
		MessageType: cmp.Or(res.MessageType, messageType),
		Decoded:     res.Decoded,
		Error:       res.Error,
		ValidBytes:  res.ValidBytes,
		UnknownFields: slices.To(res.UnknownFields, func(f entities.UnknownField) unknownFieldView {
			return *converter.Convert(&f, &unknownFieldView{})
		}),
	}
	return nil, out, nil
}

func (t *Toolset) detect(ctx context.Context, _ *mcp.CallToolRequest, in detectInput) (*mcp.CallToolResult, detectOutput, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(in.Base64))
	if err != nil {
		return nil, detectOutput{}, mcptransport.Errorf("base64 is not valid standard base64: %v", err)
	}
	if len(data) == 0 {
		return nil, detectOutput{}, mcptransport.Errorf("base64 decodes to an empty payload")
	}
	limit := in.Limit
	if limit <= 0 {
		limit = defaultDetectLimit
	}
	candidates, err := t.codec.DetectTypes(ctx, data, strings.TrimSpace(in.SourceID), min(limit, maxDetectLimit))
	if err != nil {
		return nil, detectOutput{}, err
	}
	return nil, detectOutput{Candidates: mcptransport.Items(slices.To(candidates, func(c entities.TypeCandidate) candidateView {
		return *converter.Convert(&c, &candidateView{})
	}))}, nil
}

func wireViews(fields []*entities.WireField) []wireFieldView {
	return slices.To(fields, func(f *entities.WireField) wireFieldView {
		v := *converter.Convert(f, &wireFieldView{}, converter.WithIgnoreFields("Message"))
		if len(f.Message) > 0 {
			v.Message, _ = json.Marshal(wireViews(f.Message)) //nolint:errcheck // plain structs always marshal
		}
		if v.Text != "" || len(f.Message) > 0 {
			v.Bytes = nil
		}
		return v
	})
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
	status, err := t.registry.SchemaStatus(ctx)
	if err != nil {
		return nil, statusOutput{}, err
	}
	return nil, statusOutput{
		MessageTypes: status.MessageTypes,
		Conflicts:    slices.To(status.Conflicts, func(c *entities.SchemaConflict) conflictView { return *converter.Convert(c, &conflictView{}) }),
	}, nil
}

func toMappingView(m *entities.SubjectMapping) *mappingView {
	v := converter.Convert(m, &mappingView{})
	v.FramingKind = string(m.Framing.Kind)
	return v
}
