// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package bsr reads modules from a Buf Schema Registry over its Connect JSON API.
package bsr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/dmit-4884/natscope/internal/errs"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	coreio "github.com/altessa-s/go-atlas/core/io"
)

const (
	labelPageSize    = 250
	maxLabelPages    = 8
	maxResponseBytes = 64 << 20
)

var commitID = regexp.MustCompile(`^[0-9a-f]{32}$`)

// Module is a BSR module reference such as buf.build/acme/payments.
type Module struct {
	BaseURL string
	Owner   string
	Name    string
}

// ParseModule parses "registry/owner/module"; an http:// or https:// prefix picks the scheme, https by default.
func ParseModule(ref string) (Module, error) {
	ref = strings.TrimSuffix(strings.TrimSpace(ref), "/")
	scheme := "https://"
	for _, s := range []string{"https://", "http://"} {
		if rest, ok := strings.CutPrefix(ref, s); ok {
			scheme, ref = s, rest
		}
	}
	parts := strings.Split(ref, "/")
	if len(parts) != 3 || slices.Contains(parts, "") {
		return Module{}, fmt.Errorf("%w: BSR module must look like buf.build/owner/module", errs.ErrInvalidRequest)
	}
	return Module{BaseURL: scheme + parts[0], Owner: parts[1], Name: parts[2]}, nil
}

// IsCommitID reports whether ref is a BSR commit id rather than a label.
func IsCommitID(ref string) bool {
	return commitID.MatchString(ref)
}

// Label is a named, movable pointer to a commit.
type Label struct {
	Name     string
	CommitID string
}

// Schema is a commit's descriptor set, imports included, with the paths of the module's own files.
type Schema struct {
	CommitID      string
	DescriptorSet []byte
	OwnFiles      []string
}

// Registry is the part of the BSR API natscope uses.
type Registry interface {
	// DefaultLabel returns the module's default label and fails when the module is not reachable.
	DefaultLabel(ctx context.Context, m Module, token string) (string, error)
	// ListLabels returns the module's labels, most recently updated first.
	ListLabels(ctx context.Context, m Module, token string) ([]Label, error)
	// ResolveRef returns the commit a label or commit id points to; errs.ErrProtoRefNotFound when none.
	ResolveRef(ctx context.Context, m Module, token, ref string) (string, error)
	// Schema downloads the descriptor set of a commit.
	Schema(ctx context.Context, m Module, token, commit string) (*Schema, error)
}

// Client implements Registry over HTTP.
type Client struct {
	http *http.Client
}

// New creates a BSR client.
func New(httpClient *http.Client) *Client {
	return &Client{http: httpClient}
}

type moduleName struct {
	Owner  string `json:"owner"`
	Module string `json:"module"`
	Ref    string `json:"ref,omitempty"`
}

type resourceRef struct {
	ID   string      `json:"id,omitempty"`
	Name *moduleName `json:"name,omitempty"`
}

// DefaultLabel returns the module's default label and fails when the module is not reachable.
func (c *Client) DefaultLabel(ctx context.Context, m Module, token string) (string, error) {
	var out struct {
		Modules []struct {
			DefaultLabelName string `json:"defaultLabelName"`
		} `json:"modules"`
	}
	req := map[string]any{"moduleRefs": []resourceRef{{Name: &moduleName{Owner: m.Owner, Module: m.Name}}}}
	if err := c.call(ctx, m, token, "buf.registry.module.v1.ModuleService/GetModules", req, &out); err != nil {
		return "", err
	}
	if len(out.Modules) == 0 {
		return "", &errs.RegistryError{Code: "not_found", Message: "module " + m.Owner + "/" + m.Name + " not found"}
	}
	return out.Modules[0].DefaultLabelName, nil
}

// ListLabels returns the module's labels, most recently updated first.
func (c *Client) ListLabels(ctx context.Context, m Module, token string) ([]Label, error) {
	var labels []Label
	pageToken := ""
	for range maxLabelPages {
		var out struct {
			NextPageToken string `json:"nextPageToken"`
			Labels        []struct {
				Name        string `json:"name"`
				CommitID    string `json:"commitId"`
				ArchiveTime string `json:"archiveTime"`
			} `json:"labels"`
		}
		req := map[string]any{
			"pageSize":    labelPageSize,
			"pageToken":   pageToken,
			"resourceRef": resourceRef{Name: &moduleName{Owner: m.Owner, Module: m.Name}},
			"order":       "ORDER_UPDATE_TIME_DESC",
		}
		if err := c.call(ctx, m, token, "buf.registry.module.v1.LabelService/ListLabels", req, &out); err != nil {
			return nil, err
		}
		for _, l := range out.Labels {
			if l.ArchiveTime == "" {
				labels = append(labels, Label{Name: l.Name, CommitID: l.CommitID})
			}
		}
		if out.NextPageToken == "" {
			break
		}
		pageToken = out.NextPageToken
	}
	return labels, nil
}

// ResolveRef returns the commit a label or commit id points to; errs.ErrProtoRefNotFound when none.
func (c *Client) ResolveRef(ctx context.Context, m Module, token, ref string) (string, error) {
	var out struct {
		Commits []struct {
			ID string `json:"id"`
		} `json:"commits"`
	}
	req := map[string]any{"resourceRefs": []resourceRef{{Name: &moduleName{Owner: m.Owner, Module: m.Name, Ref: ref}}}}
	err := c.call(ctx, m, token, "buf.registry.module.v1.CommitService/GetCommits", req, &out)
	if regErr, ok := errors.AsType[*errs.RegistryError](err); ok && regErr.Code == "not_found" {
		return "", fmt.Errorf("%w: %q is not a label or commit of %s/%s", errs.ErrProtoRefNotFound, ref, m.Owner, m.Name)
	}
	if err != nil {
		return "", err
	}
	if len(out.Commits) == 0 {
		return "", fmt.Errorf("%w: %q", errs.ErrProtoRefNotFound, ref)
	}
	return out.Commits[0].ID, nil
}

// Schema downloads the descriptor set of a commit.
func (c *Client) Schema(ctx context.Context, m Module, token, commit string) (*Schema, error) {
	full, err := c.descriptorSet(ctx, m, token, commit, false)
	if err != nil {
		return nil, err
	}
	own, err := c.descriptorSet(ctx, m, token, commit, true)
	if err != nil {
		return nil, err
	}
	data, err := proto.Marshal(full)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "marshal descriptor set")
	}
	files := make([]string, 0, len(own.GetFile()))
	for _, f := range own.GetFile() {
		files = append(files, f.GetName())
	}
	slices.Sort(files)
	return &Schema{CommitID: commit, DescriptorSet: data, OwnFiles: files}, nil
}

func (c *Client) descriptorSet(ctx context.Context, m Module, token, commit string, ownOnly bool) (*descriptorpb.FileDescriptorSet, error) {
	var out struct {
		FileDescriptorSet json.RawMessage `json:"fileDescriptorSet"`
	}
	req := map[string]any{
		"resourceRef":           resourceRef{ID: commit},
		"excludeImports":        ownOnly,
		"excludeSourceCodeInfo": ownOnly,
	}
	if err := c.call(ctx, m, token, "buf.registry.module.v1.FileDescriptorSetService/GetFileDescriptorSet", req, &out); err != nil {
		return nil, err
	}
	set := &descriptorpb.FileDescriptorSet{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(out.FileDescriptorSet, set); err != nil {
		return nil, coreerrs.WrapOperation(err, "decode descriptor set")
	}
	return set, nil
}

func (c *Client) call(ctx context.Context, m Module, token, procedure string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return coreerrs.WrapOperation(err, "encode BSR request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.BaseURL+"/"+procedure, bytes.NewReader(body))
	if err != nil {
		return coreerrs.WrapOperation(err, "build BSR request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return &errs.RegistryError{Code: "unavailable", Message: "cannot reach " + m.BaseURL}
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	data, err := io.ReadAll(coreio.NewLimitedReadCloser(resp.Body, maxResponseBytes))
	if err != nil {
		return coreerrs.WrapOperation(err, "read BSR response")
	}
	if resp.StatusCode != http.StatusOK {
		regErr := &errs.RegistryError{Code: "unknown", Message: resp.Status}
		var connectErr struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &connectErr) == nil && connectErr.Code != "" {
			regErr.Code, regErr.Message = connectErr.Code, connectErr.Message
		}
		return regErr
	}
	if err := json.Unmarshal(data, out); err != nil {
		return coreerrs.WrapOperation(err, "decode BSR response")
	}
	return nil
}
