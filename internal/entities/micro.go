// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// MicroReport is one NATS Micro instance's answer to $SRV.INFO or $SRV.STATS.
type MicroReport struct {
	Name        string
	ID          string
	Version     string
	Description string
	Metadata    map[string]string
	Started     time.Time
	Endpoints   []MicroEndpoint
}

// MicroDiscovery is the discovered services with the user's access to their info and stats.
type MicroDiscovery struct {
	InfoAccess  AccessCheck
	StatsAccess *AccessCheck
	Services    []MicroService
}

// MicroService is a NATS Micro service grouped across its instances.
type MicroService struct {
	Name        string
	Description string
	Versions    []string
	Instances   []MicroInstance
	Endpoints   []MicroEndpoint
}

// MicroInstance is one running instance of a NATS Micro service.
type MicroInstance struct {
	ID        string
	Version   string
	Metadata  map[string]string
	Started   *time.Time
	Endpoints []MicroEndpoint
}

// MicroEndpoint is an endpoint of a NATS Micro service.
type MicroEndpoint struct {
	Name        string
	Subject     string
	QueueGroup  string
	Metadata    map[string]string
	Stats       *MicroEndpointStats
	ProtoMethod *ProtoMethodMatch
}

// MicroEndpointStats are the request statistics of a NATS Micro endpoint.
type MicroEndpointStats struct {
	NumRequests           int64
	NumErrors             int64
	LastError             string
	ProcessingTime        time.Duration
	AverageProcessingTime time.Duration
}

// ProtoMethodMatch is the .proto method a NATS Micro endpoint implements.
type ProtoMethodMatch struct {
	SourceID   string
	Service    string
	Method     string
	InputType  string
	OutputType string
}
