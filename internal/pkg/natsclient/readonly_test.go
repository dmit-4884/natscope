// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsclient

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

var writeMethods = map[string]bool{
	"CreateStream": true, "UpdateStream": true, "DeleteStream": true, "PurgeStream": true, "SealStream": true, "DeleteMessage": true,
	"CreateConsumer": true, "UpdateConsumer": true, "DeleteConsumer": true, "PauseConsumer": true, "ResumeConsumer": true,
	"ResetConsumer": true, "UnpinConsumer": true,
	"Publish": true, "PublishToStream": true, "Request": true,
	"CreateKVBucket": true, "UpdateKVBucket": true, "PurgeKVBucket": true, "DeleteKVBucket": true, "PutKVKey": true, "CreateKVKey": true, "DeleteKVKey": true, "PurgeKVKey": true,
	"CreateObjectBucket": true, "DeleteObjectBucket": true, "PutObject": true, "PutObjectStream": true, "DeleteObject": true,
	"SealObjectBucket": true,
}

type panicClient struct{ Client }

func pooledClient(t *testing.T, saved *entities.SavedConnection, inner Client) Client {
	t.Helper()
	pool := NewPool(&fakeDialer{client: inner}, func(context.Context, string) (*entities.SavedConnection, error) {
		return saved, nil
	})
	c, err := pool.Client(t.Context(), "conn-1")
	if err != nil {
		t.Fatalf("Client() error = %v", err)
	}
	return c
}

func callMethod(m reflect.Value) (out []reflect.Value, delegated bool) {
	defer func() {
		if recover() != nil {
			delegated = true
		}
	}()
	args := make([]reflect.Value, m.Type().NumIn())
	for i := range args {
		args[i] = reflect.Zero(m.Type().In(i))
	}
	return m.Call(args), false
}

func TestPool_ReadOnlyConnection_RefusesEveryWriteAndPassesReads(t *testing.T) {
	t.Parallel()

	c := pooledClient(t, &entities.SavedConnection{ReadOnly: true}, &panicClient{})
	v := reflect.ValueOf(c)
	clientType := reflect.TypeFor[Client]()
	seen := 0
	for method := range clientType.Methods() {
		name := method.Name
		if name == "IsConnected" || name == "IsReconnecting" || name == "Status" || name == "Close" {
			continue
		}
		out, delegated := callMethod(v.MethodByName(name))
		if !writeMethods[name] {
			if !delegated {
				t.Errorf("%s: read did not reach the connection", name)
			}
			continue
		}
		seen++
		if delegated {
			t.Errorf("%s: write reached the connection", name)
			continue
		}
		err, _ := out[len(out)-1].Interface().(error)
		if !errors.Is(err, errs.ErrConnectionReadOnly) {
			t.Errorf("%s: error = %v, want ErrConnectionReadOnly", name, err)
		}
	}
	if seen != len(writeMethods) {
		t.Fatalf("checked %d writes, want %d: the write list names a method Client no longer has", seen, len(writeMethods))
	}
}

func TestPool_WritableConnection_PassesWrites(t *testing.T) {
	t.Parallel()

	c := pooledClient(t, &entities.SavedConnection{}, &panicClient{})
	if _, delegated := callMethod(reflect.ValueOf(c).MethodByName("CreateStream")); !delegated {
		t.Fatalf("CreateStream did not reach the connection")
	}
}

type gatedDialer struct {
	fakeDialer
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (d *gatedDialer) Dial(context.Context, *entities.SavedConnection) (Client, error) {
	if d.calls.Add(1) == 1 {
		close(d.entered)
		<-d.release
	}
	return &fakeClient{connected: true}, nil
}

func TestPool_ReadOnlySwitchedOnDuringDial_IsNotLost(t *testing.T) {
	t.Parallel()

	var readOnly atomic.Bool
	d := &gatedDialer{entered: make(chan struct{}), release: make(chan struct{})}
	pool := NewPool(d, func(context.Context, string) (*entities.SavedConnection, error) {
		return &entities.SavedConnection{ReadOnly: readOnly.Load()}, nil
	})

	got := make(chan Client, 1)
	go func() {
		c, err := pool.Client(t.Context(), "conn-1")
		if err != nil {
			t.Errorf("Client() error = %v", err)
		}
		got <- c
	}()
	<-d.entered
	readOnly.Store(true)
	pool.Disconnect("conn-1")
	close(d.release)

	if _, ok := (<-got).(*readOnlyClient); !ok {
		t.Fatalf("the pool handed out a writable client dialed before the connection became read-only")
	}
	if c, _ := pool.Pooled("conn-1"); c != nil {
		if _, ok := c.(*readOnlyClient); !ok {
			t.Fatalf("the pool kept a writable client for a read-only connection")
		}
	}
}
