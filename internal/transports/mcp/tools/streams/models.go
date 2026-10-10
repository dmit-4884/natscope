// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package streams

import (
	"time"

	mcptransport "github.com/dmit-4884/natscope/internal/transports/mcp"
)

type streamSummary struct {
	Config  streamSummaryConfig `json:"config"`
	State   *streamStateView    `json:"state,omitempty"`
	Created time.Time           `json:"created"`
}

type streamSummaryConfig struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Subjects    []string `json:"subjects"`
	Retention   string   `json:"retention"`
	Storage     string   `json:"storage"`
	Replicas    int      `json:"replicas"`
	Sealed      bool     `json:"sealed,omitempty"`
}

type streamStateView struct {
	Msgs        uint64    `json:"messages"`
	Bytes       uint64    `json:"bytes"`
	FirstSeq    uint64    `json:"firstSeq"`
	LastSeq     uint64    `json:"lastSeq"`
	FirstTime   time.Time `json:"firstTime"`
	LastTime    time.Time `json:"lastTime"`
	Consumers   int       `json:"consumers"`
	NumDeleted  int       `json:"deleted,omitempty"`
	NumSubjects uint64    `json:"subjects"`
}

type listStreamsOutput struct {
	Streams []streamSummary `json:"streams"`
}

type streamInput struct {
	mcptransport.ConnectionArg
	Stream string `json:"stream" jsonschema:"stream name" normalize:"trim"`
}

type streamView struct {
	Config  streamConfigView `json:"config"`
	State   *streamStateView `json:"state,omitempty"`
	Created time.Time        `json:"created"`
	Cluster *clusterView     `json:"cluster,omitempty"`
}

type streamConfigView struct {
	Name                   string            `json:"name"`
	Description            string            `json:"description,omitempty"`
	Subjects               []string          `json:"subjects"`
	Retention              string            `json:"retention"`
	Storage                string            `json:"storage"`
	Replicas               int               `json:"replicas"`
	Discard                string            `json:"discard"`
	MaxMsgs                int64             `json:"maxMsgs" jsonschema:"-1 means unlimited"`
	MaxBytes               int64             `json:"maxBytes" jsonschema:"-1 means unlimited"`
	MaxAge                 string            `json:"maxAge,omitempty"`
	MaxMsgSize             int32             `json:"maxMsgSize" jsonschema:"-1 means unlimited"`
	MaxMsgsPerSubject      int64             `json:"maxMsgsPerSubject" jsonschema:"-1 means unlimited"`
	MaxConsumers           int               `json:"maxConsumers" jsonschema:"-1 means unlimited"`
	Duplicates             string            `json:"duplicateWindow,omitempty"`
	Compression            string            `json:"compression"`
	Sealed                 bool              `json:"sealed"`
	DenyDelete             bool              `json:"denyDelete"`
	DenyPurge              bool              `json:"denyPurge"`
	AllowRollup            bool              `json:"allowRollup"`
	AllowDirect            bool              `json:"allowDirect"`
	AllowMsgTTL            bool              `json:"allowMsgTtl"`
	AllowAtomicPublish     bool              `json:"allowAtomicPublish"`
	AllowMsgCounter        bool              `json:"allowMsgCounter"`
	AllowMsgSchedules      bool              `json:"allowMsgSchedules"`
	AllowBatchPublish      bool              `json:"allowBatchPublish"`
	SubjectDeleteMarkerTTL string            `json:"subjectDeleteMarkerTtl,omitempty"`
	PersistMode            string            `json:"persistMode"`
	Mirror                 *sourceView       `json:"mirror,omitempty"`
	Sources                []*sourceView     `json:"sources,omitempty"`
	Metadata               map[string]string `json:"metadata,omitempty"`
}

type sourceView struct {
	Name          string `json:"name"`
	FilterSubject string `json:"filterSubject,omitempty"`
	OptStartSeq   uint64 `json:"startSeq,omitempty"`
}

type clusterView struct {
	Name     string      `json:"name,omitempty"`
	Leader   string      `json:"leader,omitempty"`
	Replicas []*peerView `json:"replicas,omitempty"`
}

type peerView struct {
	Name    string `json:"name"`
	Current bool   `json:"current"`
	Offline bool   `json:"offline,omitempty"`
	Active  string `json:"lastActive,omitempty"`
	Lag     uint64 `json:"lag,omitempty"`
}

type listConsumersInput struct {
	mcptransport.ConnectionArg
	Stream string `json:"stream,omitempty" jsonschema:"stream name; omit to list the consumers of every stream" normalize:"trim"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum consumers to return, most pending first, 1-1000 (default 200)"`
}

type consumerView struct {
	Stream         string              `json:"stream"`
	Name           string              `json:"name"`
	Description    string              `json:"description,omitempty"`
	Durable        string              `json:"durable,omitempty"`
	FilterSubject  string              `json:"filterSubject,omitempty"`
	FilterSubjects []string            `json:"filterSubjects,omitempty"`
	DeliverPolicy  string              `json:"deliverPolicy"`
	AckPolicy      string              `json:"ackPolicy"`
	AckWait        string              `json:"ackWait,omitempty"`
	MaxDeliver     int                 `json:"maxDeliver"`
	MaxAckPending  int                 `json:"maxAckPending"`
	PushBound      bool                `json:"pushBound,omitempty"`
	NumPending     uint64              `json:"numPending"`
	NumAckPending  int                 `json:"numAckPending"`
	NumRedelivered int                 `json:"numRedelivered"`
	NumWaiting     int                 `json:"numWaiting"`
	Delivered      sequenceView        `json:"delivered"`
	AckFloor       sequenceView        `json:"ackFloor"`
	Created        *time.Time          `json:"created,omitempty"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
	PriorityPolicy string              `json:"priorityPolicy"`
	PinnedTTL      string              `json:"pinnedTtl,omitempty"`
	PriorityGroups []string            `json:"priorityGroups,omitempty"`
	Pinned         []priorityGroupView `json:"pinned,omitempty" jsonschema:"priority groups whose pull requests are currently pinned to one client"`
}

type priorityGroupView struct {
	Group          string    `json:"group"`
	PinnedClientID string    `json:"pinnedClientId,omitempty"`
	PinnedTS       time.Time `json:"pinnedAt,omitzero"`
}

type sequenceView struct {
	Consumer uint64 `json:"consumerSeq"`
	Stream   uint64 `json:"streamSeq"`
}

type listConsumersOutput struct {
	Consumers         []consumerView         `json:"consumers"`
	Total             int                    `json:"total" jsonschema:"consumers found before the limit"`
	UnreadableStreams []unreadableStreamView `json:"unreadableStreams,omitempty" jsonschema:"streams whose consumers could not be listed"`
}

type unreadableStreamView struct {
	Stream string `json:"stream"`
	Reason string `json:"reason"`
}

type relationsInput struct {
	mcptransport.ConnectionArg
	Stream string `json:"stream,omitempty" jsonschema:"stream name; omit for every relation of the connection" normalize:"trim"`
}

type relationsOutput struct {
	Nodes     []relationNodeView `json:"nodes"`
	Relations []relationView     `json:"relations"`
}

type relationNodeView struct {
	ID       string        `json:"id" jsonschema:"the stream name for local streams; placeholders carry a prefix"`
	Name     string        `json:"name"`
	Kind     string        `json:"kind" jsonschema:"stream, kv, object_store, external (other account or domain), missing, or subject (uncaptured republish)"`
	External *externalView `json:"external,omitempty"`
}

type externalView struct {
	APIPrefix     string `json:"apiPrefix"`
	DeliverPrefix string `json:"deliverPrefix,omitempty"`
}

type relationView struct {
	Kind      string         `json:"kind" jsonschema:"source, mirror or republish"`
	From      string         `json:"from" jsonschema:"id of the upstream node"`
	To        string         `json:"to" jsonschema:"id of the downstream node"`
	Source    *linkView      `json:"source,omitempty"`
	State     *linkStateView `json:"state,omitempty"`
	Republish *republishView `json:"republish,omitempty"`
}

type linkView struct {
	FilterSubject     string          `json:"filterSubject,omitempty"`
	SubjectTransforms []transformView `json:"subjectTransforms,omitempty"`
	OptStartSeq       uint64          `json:"startSeq,omitempty"`
	OptStartTime      *time.Time      `json:"startTime,omitempty"`
	External          *externalView   `json:"external,omitempty"`
}

type linkStateView struct {
	Lag    uint64 `json:"lag"`
	Active string `json:"lastActive" jsonschema:"time since the upstream was last heard from; negative when it never was"`
	Error  string `json:"error,omitempty"`
}

type transformView struct {
	Source      string `json:"src"`
	Destination string `json:"dest,omitempty" jsonschema:"empty when the subject is kept as is"`
}

type republishView struct {
	Src         string `json:"src"`
	Dest        string `json:"dest"`
	HeadersOnly bool   `json:"headersOnly,omitempty"`
}
