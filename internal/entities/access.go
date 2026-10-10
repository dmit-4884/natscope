// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// AccessStatus is whether the server let the NATS user perform an operation.
type AccessStatus int

const (
	AccessUnspecified AccessStatus = iota
	AccessAllowed
	AccessDenied
)

// AccessCheck is the outcome of an access-sensitive NATS operation.
type AccessCheck struct {
	Status    AccessStatus
	Operation string
	Subject   string
}
