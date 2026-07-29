// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// RetentionPolicy defines the message retention policy for a stream.
type RetentionPolicy int

const (
	RetentionLimits    RetentionPolicy = iota // limits
	RetentionInterest                         // interest
	RetentionWorkQueue                        // workqueue
)

// IsValid checks if the retention policy is a known value.
func (p RetentionPolicy) IsValid() bool {
	switch p {
	case RetentionLimits, RetentionInterest, RetentionWorkQueue:
		return true
	default:
		return false
	}
}

// StorageType defines the storage backend type.
type StorageType int

const (
	StorageFile   StorageType = iota // file
	StorageMemory                    // memory
)

// IsValid checks if the storage type is a known value.
func (t StorageType) IsValid() bool {
	switch t {
	case StorageFile, StorageMemory:
		return true
	default:
		return false
	}
}

// DiscardPolicy defines what happens when stream limits are reached.
type DiscardPolicy int

const (
	DiscardOld DiscardPolicy = iota // old
	DiscardNew                      // new
)

// IsValid checks if the discard policy is a known value.
func (p DiscardPolicy) IsValid() bool {
	switch p {
	case DiscardOld, DiscardNew:
		return true
	default:
		return false
	}
}

// StoreCompression defines the compression algorithm for stream storage.
type StoreCompression int

const (
	CompressionNone StoreCompression = iota // none
	CompressionS2                           // s2
)

// IsValid checks if the compression type is a known value.
func (c StoreCompression) IsValid() bool {
	switch c {
	case CompressionNone, CompressionS2:
		return true
	default:
		return false
	}
}

// DeliverPolicy defines where to start delivering messages from.
type DeliverPolicy int

const (
	DeliverAll             DeliverPolicy = iota // all
	DeliverLast                                 // last
	DeliverNew                                  // new
	DeliverByStartSequence                      // by_start_sequence
	DeliverByStartTime                          // by_start_time
	DeliverLastPerSubject                       // last_per_subject
)

// IsValid checks if the deliver policy is a known value.
func (p DeliverPolicy) IsValid() bool {
	switch p {
	case DeliverAll, DeliverLast, DeliverNew, DeliverByStartSequence, DeliverByStartTime, DeliverLastPerSubject:
		return true
	default:
		return false
	}
}

// AckPolicy defines the acknowledgment policy for consumers.
type AckPolicy int

const (
	AckExplicit AckPolicy = iota // explicit
	AckAll                       // all
	AckNone                      // none
)

// IsValid checks if the ack policy is a known value.
func (p AckPolicy) IsValid() bool {
	switch p {
	case AckExplicit, AckNone, AckAll:
		return true
	default:
		return false
	}
}

// ReplayPolicy defines how messages are replayed for consumers.
type ReplayPolicy int

const (
	ReplayInstant  ReplayPolicy = iota // instant
	ReplayOriginal                     // original
)

// IsValid checks if the replay policy is a known value.
func (p ReplayPolicy) IsValid() bool {
	switch p {
	case ReplayInstant, ReplayOriginal:
		return true
	default:
		return false
	}
}
