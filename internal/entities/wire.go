// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// WireType is how a field is encoded on the wire.
type WireType string

const (
	WireVarint  WireType = "varint"
	WireFixed64 WireType = "fixed64"
	WireBytes   WireType = "bytes"
	WireGroup   WireType = "group"
	WireFixed32 WireType = "fixed32"
)

// WireField is a payload field read without a schema; Message holds the fields of bytes that parse as a message.
type WireField struct {
	Number   int32
	WireType WireType
	Offset   int
	Length   int
	Varint   uint64
	Fixed    uint64
	Bytes    []byte
	Text     string
	Message  []*WireField
}

// WireDump is a payload read without a schema; Error says why reading stopped before the end.
type WireDump struct {
	Fields     []*WireField
	ValidBytes int
	Error      string
}

// UnknownField is a field the payload carries but its schema does not declare.
type UnknownField struct {
	Path     string
	Number   int32
	WireType WireType
	Size     int
}
