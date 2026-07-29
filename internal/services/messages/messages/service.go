// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	messagessvc "github.com/dmit-4884/natscope/internal/services/messages"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
)

// Service implements messagessvc.Service.
type Service struct {
	natsService     natssvc.StreamReader
	protoService    protosvc.Codec
	settingsService settingssvc.Service
	logger          *slog.Logger
}

// New creates a new messages coordinator service.
func New(
	natsService natssvc.StreamReader,
	protoService protosvc.Codec,
	settingsService settingssvc.Service,
) *Service {
	return &Service{
		natsService:     natsService,
		protoService:    protoService,
		settingsService: settingsService,
		logger:          slog.Default().With(slogx.Module("service:messages")),
	}
}

// skipDecodeMultiplier skips proto decode when raw payload exceeds this many ×
// MaxPayloadBytes — decoded JSON would surely blow the cap too (waste).
const skipDecodeMultiplier = 8

// List applies settings defaults, fetches, decodes, content-filters, then
// truncates payload + decoded JSON to the cap.
func (s *Service) List(ctx context.Context, in *entities.MessageListRequest) (*entities.MessagesResponse, error) {
	opts := s.buildOptions(ctx, in)

	resp, err := s.natsService.GetMessages(ctx, in.ConnectionID, in.StreamName, opts)
	if err != nil {
		return nil, err
	}

	// Decode only messages small enough to yield a useful preview; oversized
	// ones get a tiny clip anyway.
	toDecodeList := resp.Messages
	if opts.MaxPayloadBytes > 0 {
		toDecodeList = decodeableMessages(resp.Messages, opts.MaxPayloadBytes)
	}

	s.protoService.DecodeMessages(ctx, toDecodeList)

	if opts.ContentFilter != "" {
		resp.Messages = filterByContent(resp.Messages, opts.ContentFilter)
	}

	if opts.MaxPayloadBytes > 0 {
		truncateMessages(resp.Messages, int(opts.MaxPayloadBytes))
	}

	return resp, nil
}

// decodeableMessages filters messages whose raw size might still fit
// maxBytes after decode; pointer identity propagates results with no copy.
func decodeableMessages(msgs []*entities.Message, maxBytes int32) []*entities.Message {
	out := make([]*entities.Message, 0, len(msgs))
	threshold := int64(maxBytes) * skipDecodeMultiplier
	for _, m := range msgs {
		if int64(m.DataSize) <= threshold {
			out = append(out, m)
		}
	}
	return out
}

// Get fetches and decodes a single message. Always returns the full payload —
// it's the "load full" path, so no cap applies.
func (s *Service) Get(ctx context.Context, in *entities.MessageGetRequest) (*entities.Message, error) {
	msg, err := s.natsService.GetMessage(ctx, in.ConnectionID, in.StreamName, in.Sequence)
	if err != nil {
		return nil, err
	}
	s.protoService.DecodeMessages(ctx, []*entities.Message{msg})
	return msg, nil
}

// buildOptions builds NATS-service options, applying user-settings
// fall-throughs for unset fields. Unreadable settings fall back to defaults.
func (s *Service) buildOptions(ctx context.Context, in *entities.MessageListRequest) entities.GetMessagesOptions {
	opts := entities.GetMessagesOptions{
		Direction:   in.Direction,
		FetchMethod: in.FetchMethod,
	}
	if in.SubjectFilter != nil {
		opts.SubjectFilter = *in.SubjectFilter
	}
	if in.StartSeq != nil {
		opts.StartSeq = *in.StartSeq
	}
	if in.StartTime != nil {
		opts.StartTime = in.StartTime
	}
	if in.Limit != nil {
		opts.Limit = int(*in.Limit)
	}
	if in.ContentFilter != nil {
		opts.ContentFilter = *in.ContentFilter
	}
	// Request-level override has highest precedence; we still apply settings
	// fall-through below when nil.
	if in.MaxPayloadBytes != nil {
		opts.MaxPayloadBytes = *in.MaxPayloadBytes
	}

	cfg, err := s.settingsService.Get(ctx)
	if err != nil || cfg == nil || cfg.Messages == nil {
		// Apply server defaults only if the caller did not pin a value.
		if in.MaxPayloadBytes == nil {
			opts.MaxPayloadBytes = entities.DefaultMaxPayloadBytesInList
		}
		if opts.FetchMethod == "" {
			opts.FetchMethod = entities.DefaultFetchMethod
		}
		return opts
	}
	if opts.Limit == 0 && cfg.Messages.DefaultPageSize != nil {
		opts.Limit = int(*cfg.Messages.DefaultPageSize)
	}
	if opts.Direction == "" && cfg.Messages.DefaultDirection != nil {
		opts.Direction = *cfg.Messages.DefaultDirection
	}
	if opts.FetchMethod == "" && cfg.Messages.FetchMethod != nil {
		opts.FetchMethod = *cfg.Messages.FetchMethod
	}
	if opts.FetchMethod == "" {
		opts.FetchMethod = entities.DefaultFetchMethod
	}
	if in.MaxPayloadBytes == nil {
		switch {
		case cfg.Messages.MaxPayloadBytesInList != nil:
			opts.MaxPayloadBytes = *cfg.Messages.MaxPayloadBytesInList
		default:
			opts.MaxPayloadBytes = entities.DefaultMaxPayloadBytesInList
		}
	}
	return opts
}

// truncateMessages caps each base64 payload to maxBytes and previews (not
// clips) decoded JSON, since byte-cutting JSON would produce invalid output.
func truncateMessages(messages []*entities.Message, maxBytes int) {
	if maxBytes <= 0 {
		return
	}
	base64Cap := (maxBytes / base64RawGroup) * base64GroupSize
	if base64Cap <= 0 {
		base64Cap = base64GroupSize
	}
	for _, m := range messages {
		if m.DataSize == 0 && m.DataBase64 != "" {
			m.DataSize = base64DecodedLen(m.DataBase64)
		}
		dataTooBig := m.DataSize > maxBytes && len(m.DataBase64) > base64Cap
		decodedTooBig := len(m.Decoded) > maxBytes
		if !dataTooBig && !decodedTooBig {
			continue
		}
		m.Truncated = true
		if dataTooBig {
			m.DataBase64 = m.DataBase64[:base64Cap]
		}
		if decodedTooBig {
			// Small fixed-size text preview: byte-cutting JSON is invalid
			// and dropping it hides whether the proto matched at all.
			m.Decoded = decodedPreview(m.Decoded)
		}
	}
}

// decodedPreviewLines / decodedPreviewBytes bound the preview regardless of
// payload shape; the byte cap guards against one-line minified JSON.
const (
	decodedPreviewLines = 20
	decodedPreviewBytes = 2 * 1024
)

// decodedPreview returns a small textual preview, itself JSON-encoded so it
// stays a valid json.RawMessage for the frontend, suffixed with "…".
func decodedPreview(raw json.RawMessage) json.RawMessage {
	preview := buildPreviewText(raw)
	out, err := json.Marshal(preview)
	if err != nil {
		return nil
	}
	return out
}

// indentInputCap caps bytes fed to json.Indent (only ~decodedPreviewLines of
// output needed). A mid-token truncation fails Indent → compact raw fallback.
const indentInputCap = 64 * 1024

func buildPreviewText(raw json.RawMessage) string {
	input := raw
	if len(input) > indentInputCap {
		input = raw[:indentInputCap]
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, input, "", "  "); err != nil {
		// Input was truncated mid-token — fall back to compact clip.
		return clipText(string(raw)) + "\n…"
	}
	text := indented.String()
	if idx := nthIndex(text, '\n', decodedPreviewLines); idx > 0 {
		text = text[:idx]
	}
	return clipText(text) + "\n…"
}

func clipText(s string) string {
	if len(s) <= decodedPreviewBytes {
		return s
	}
	return s[:decodedPreviewBytes]
}

func nthIndex(s string, c byte, n int) int {
	for i, count := 0, 0; i < len(s); i++ {
		if s[i] == c {
			count++
			if count == n {
				return i
			}
		}
	}
	return -1
}

// base64 emits 4 chars per 3 raw bytes; group sizes for length math.
const (
	base64GroupSize = 4
	base64RawGroup  = 3
)

// base64DecodedLen returns the decoded byte length of a (possibly padded)
// base64 string. Unparsable input returns 0, matching DetectContentType.
func base64DecodedLen(s string) int {
	n := len(s) / base64GroupSize * base64RawGroup
	if n == 0 {
		return 0
	}
	const (
		base64DoublePad = 2 // "==" trailer → 2 fewer raw bytes
		base64SinglePad = 1 // "=" trailer → 1 fewer raw byte
	)
	switch {
	case strings.HasSuffix(s, "=="):
		return n - base64DoublePad
	case strings.HasSuffix(s, "="):
		return n - base64SinglePad
	default:
		return n
	}
}

// filterByContent returns the subset of messages whose decoded JSON or
// base64-decoded raw payload contains the lowercased filter substring.
func filterByContent(messages []*entities.Message, filter string) []*entities.Message {
	if filter == "" {
		return messages
	}
	lowerFilter := strings.ToLower(filter)
	out := messages[:0]
	for _, msg := range messages {
		if matchesContent(msg, lowerFilter) {
			out = append(out, msg)
		}
	}
	return out
}

func matchesContent(msg *entities.Message, lowerFilter string) bool {
	if msg.Decoded != nil {
		if strings.Contains(strings.ToLower(string(msg.Decoded)), lowerFilter) {
			return true
		}
	}
	if msg.DataBase64 != "" {
		if raw, err := base64.StdEncoding.DecodeString(msg.DataBase64); err == nil {
			if strings.Contains(strings.ToLower(string(raw)), lowerFilter) {
				return true
			}
		}
	}
	return false
}

var _ messagessvc.Service = (*Service)(nil)
