// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"time"
)

type connectionView struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	URLs        []string  `json:"urls"`
	Auth        *authView `json:"auth,omitempty"`
	Meta        *testView `json:"lastTest,omitempty"`
}

type authView struct {
	Method string `json:"method" jsonschema:"none, user_pass, token, nkey or credentials"`
}

type testView struct {
	LastTestedAt     time.Time `json:"testedAt"`
	LastSuccess      bool      `json:"success"`
	LastRTTMs        *int64    `json:"rttMs,omitempty"`
	ServerVersion    *string   `json:"serverVersion,omitempty"`
	ServerName       *string   `json:"serverName,omitempty"`
	ClusterName      *string   `json:"clusterName,omitempty"`
	JetstreamEnabled *bool     `json:"jetstream,omitempty"`
	LastError        *string   `json:"error,omitempty"`
}

type listConnectionsOutput struct {
	Connections []connectionView `json:"connections"`
}

type serverView struct {
	ServerName   string         `json:"serverName"`
	Version      string         `json:"version"`
	Host         string         `json:"host"`
	Port         int32          `json:"port"`
	ClusterName  string         `json:"clusterName,omitempty"`
	MaxPayload   int64          `json:"maxPayload"`
	Jetstream    bool           `json:"jetstream"`
	AuthRequired bool           `json:"authRequired"`
	TlsRequired  bool           `json:"tlsRequired"`
	ConnectUrls  []string       `json:"clusterUrls,omitempty"`
	JsAccount    *jsAccountView `json:"jetstreamAccount,omitempty"`
}

type jsAccountView struct {
	Memory        int64  `json:"memoryBytes"`
	Storage       int64  `json:"storageBytes"`
	Streams       int64  `json:"streams"`
	Consumers     int64  `json:"consumers"`
	MemoryLimit   int64  `json:"memoryLimit" jsonschema:"-1 means unlimited"`
	StorageLimit  int64  `json:"storageLimit" jsonschema:"-1 means unlimited"`
	StreamLimit   int64  `json:"streamLimit" jsonschema:"-1 means unlimited"`
	ConsumerLimit int64  `json:"consumerLimit" jsonschema:"-1 means unlimited"`
	Domain        string `json:"domain,omitempty"`
}
