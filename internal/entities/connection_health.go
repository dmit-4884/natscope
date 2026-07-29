// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ConnectionHealth is a health probe result for a saved connection.
type ConnectionHealth struct {
	Id             string
	URL            string
	Status         string
	RTT            string
	Error          string
	IsConnected    bool
	IsReconnecting bool
	ServerVersion  string
}
