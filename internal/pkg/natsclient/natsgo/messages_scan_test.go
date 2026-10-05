// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"bytes"
	"context"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

// throttle forwards a TCP port, sending server-to-client bytes at roughly bytesPerSecond.
func throttle(t *testing.T, upstream string, bytesPerSecond int) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })
	go func() {
		for {
			client, err := lis.Accept()
			if err != nil {
				return
			}
			srv, err := net.Dial("tcp", upstream)
			if err != nil {
				_ = client.Close()
				return
			}
			go func() { _, _ = io.Copy(srv, client); _ = srv.Close() }()
			go func() {
				defer client.Close()
				const chunk = 4096
				buf := make([]byte, chunk)
				for {
					n, err := srv.Read(buf)
					if n > 0 {
						if _, werr := client.Write(buf[:n]); werr != nil {
							return
						}
						time.Sleep(time.Duration(n) * time.Second / time.Duration(bytesPerSecond))
					}
					if err != nil {
						return
					}
				}
			}()
		}
	}()
	return "nats://" + lis.Addr().String()
}

func jetStreamServer(t *testing.T) (*server.Server, string) {
	t.Helper()
	opts := &server.Options{Host: "127.0.0.1", Port: -1, NoLog: true, NoSigs: true, JetStream: true, StoreDir: t.TempDir()}
	srv, err := server.NewServer(opts)
	require.NoError(t, err)
	go srv.Start()
	require.True(t, srv.ReadyForConnections(10*time.Second))
	t.Cleanup(srv.Shutdown)
	return srv, srv.ClientURL()
}

func dialClient(t *testing.T, url string) *Client {
	t.Helper()
	c, err := NewDialer().Dial(t.Context(), &entities.SavedConnection{URLs: []string{url}})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	client, ok := c.(*Client)
	require.True(t, ok)
	return client
}

func scanAll(t *testing.T, c *Client, stream string, opts entities.ScanOptions) []*entities.Message {
	t.Helper()
	var out []*entities.Message
	for msg, err := range c.ScanMessages(t.Context(), stream, opts) {
		require.NoError(t, err)
		out = append(out, msg)
	}
	return out
}

func sequences(msgs []*entities.Message) []uint64 {
	out := make([]uint64, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, m.Sequence)
	}
	return out
}

func TestScanViaConsumer_SlowLinkLosesNothing(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "BIG", Subjects: []string{"big.>"}})
	require.NoError(t, err)
	payload := bytes.Repeat([]byte("x"), 300_000)
	for i := range 3 {
		_, err = js.Publish(t.Context(), "big.small", []byte("small "+strconv.Itoa(i)))
		require.NoError(t, err)
	}
	for range 3 {
		_, err = js.Publish(t.Context(), "big.large", payload)
		require.NoError(t, err)
	}

	c := dialClient(t, throttle(t, strings.TrimPrefix(url, "nats://"), 200_000))
	got := scanAll(t, c, "BIG", entities.ScanOptions{FromSeq: 1, ToSeq: 6, FetchMethod: fetchMethodConsumer})

	assert.Equal(t, []uint64{1, 2, 3, 4, 5, 6}, sequences(got))
}

func TestScanViaConsumer_StopsAtTheEndWithoutWaiting(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	c := dialClient(t, url)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "S", Subjects: []string{"s.>"}})
	require.NoError(t, err)
	for range 3 {
		_, err = js.Publish(t.Context(), "s.a", []byte("x"))
		require.NoError(t, err)
	}

	start := time.Now()
	got := scanAll(t, c, "S", entities.ScanOptions{FromSeq: 1, ToSeq: 100, FetchMethod: fetchMethodConsumer})

	assert.Equal(t, []uint64{1, 2, 3}, sequences(got))
	assert.Less(t, time.Since(start), time.Second)
	empty := scanAll(t, c, "S", entities.ScanOptions{FromSeq: 1, ToSeq: 3, SubjectFilter: "s.none", FetchMethod: fetchMethodConsumer})
	assert.Empty(t, empty)
	assert.Less(t, time.Since(start), 2*time.Second)
}

func TestScanDirect_JumpsOverGaps(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name: "ROLL", Subjects: []string{"roll.>"}, MaxMsgsPerSubject: 1,
	})
	require.NoError(t, err)
	_, err = js.Publish(t.Context(), "roll.first", []byte("first"), jetstream.WithMsgID("first"))
	require.NoError(t, err)
	for range 30 {
		for range 1000 {
			_, err = js.PublishAsync("roll.state", []byte("s"))
			require.NoError(t, err)
		}
		select {
		case <-js.PublishAsyncComplete():
		case <-time.After(30 * time.Second):
			t.Fatal("publishing did not finish")
		}
	}
	_, err = js.Publish(t.Context(), "roll.last", []byte("last"))
	require.NoError(t, err)

	c := dialClient(t, url)
	before, err := js.AccountInfo(t.Context())
	require.NoError(t, err)
	got := scanAll(t, c, "ROLL", entities.ScanOptions{FromSeq: 1, ToSeq: 30_002, FetchMethod: "direct"})
	after, err := js.AccountInfo(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []uint64{1, 30_001, 30_002}, sequences(got))
	assert.Less(t, after.API.Total-before.API.Total, uint64(1000), "a gap costs a jump, not a read per sequence")
}

func TestScanDirect_DropsTheHeadersDirectReadsAdd(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "DIR", Subjects: []string{"dir.>"}, AllowDirect: true})
	require.NoError(t, err)
	hdr := nats.Header{}
	hdr.Set("X-Mine", "yes")
	_, err = js.PublishMsg(t.Context(), &nats.Msg{Subject: "dir.a", Data: []byte("a"), Header: hdr})
	require.NoError(t, err)

	c := dialClient(t, url)
	for _, filter := range []string{"", "dir.a"} {
		got := scanAll(t, c, "DIR", entities.ScanOptions{FromSeq: 1, ToSeq: 1, SubjectFilter: filter, FetchMethod: "direct"})
		require.Len(t, got, 1)
		assert.Equal(t, map[string]string{"X-Mine": "yes"}, got[0].Headers, "filter %q", filter)
	}
}

func TestBrowseViaConsumer_SlowLinkKeepsTheWholePage(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "PAGE", Subjects: []string{"page.>"}})
	require.NoError(t, err)
	payload := bytes.Repeat([]byte("y"), 300_000)
	for range 4 {
		_, err = js.Publish(t.Context(), "page.large", payload)
		require.NoError(t, err)
	}

	c := dialClient(t, throttle(t, strings.TrimPrefix(url, "nats://"), 200_000))
	for _, dir := range []string{"forward", "backward"} {
		resp, err := c.GetMessages(t.Context(), "PAGE", entities.GetMessagesOptions{Limit: 10, Direction: dir, FetchMethod: fetchMethodConsumer})
		require.NoError(t, err)
		assert.Len(t, resp.Messages, 4, dir)
	}
}

func TestSeqAtTimeViaConsumer_SlowLinkStillFindsTheMessage(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "TIME", Subjects: []string{"time.>"}})
	require.NoError(t, err)
	var between time.Time
	for i := range 3 {
		if i == 1 {
			time.Sleep(50 * time.Millisecond)
			between = time.Now()
			time.Sleep(50 * time.Millisecond)
		}
		_, err = js.Publish(t.Context(), "time.large", bytes.Repeat([]byte("z"), 300_000))
		require.NoError(t, err)
	}

	c := dialClient(t, throttle(t, strings.TrimPrefix(url, "nats://"), 200_000))
	seq, err := c.SeqAtTime(t.Context(), "TIME", fetchMethodConsumer, between)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), seq)
}

func slowFirstMessageStream(t *testing.T, name string) string {
	t.Helper()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: name, Subjects: []string{"slow.>"}})
	require.NoError(t, err)
	_, err = js.Publish(t.Context(), "slow.large", bytes.Repeat([]byte("w"), 700_000))
	require.NoError(t, err)
	for i := range 100 {
		_, err = js.Publish(t.Context(), "slow.small", []byte("small "+strconv.Itoa(i)))
		require.NoError(t, err)
	}
	return throttle(t, strings.TrimPrefix(url, "nats://"), 100_000)
}

func TestScanViaConsumer_SlowFirstMessageLosesNothing(t *testing.T) {
	t.Parallel()
	c := dialClient(t, slowFirstMessageStream(t, "SLOWSCAN"))

	got := scanAll(t, c, "SLOWSCAN", entities.ScanOptions{FromSeq: 1, ToSeq: 101, FetchMethod: fetchMethodConsumer})

	require.Len(t, got, 101)
	assert.Equal(t, uint64(1), got[0].Sequence)
	assert.Equal(t, uint64(101), got[100].Sequence)
}

func TestBrowseViaConsumer_SlowFirstMessageIsNotAnEmptyPage(t *testing.T) {
	t.Parallel()
	c := dialClient(t, slowFirstMessageStream(t, "SLOWPAGE"))

	resp, err := c.GetMessages(t.Context(), "SLOWPAGE", entities.GetMessagesOptions{Limit: 10, Direction: "forward", FetchMethod: fetchMethodConsumer})

	require.NoError(t, err)
	assert.Equal(t, []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, sequences(resp.Messages))
}

func republishedStream(t *testing.T) *Client {
	t.Helper()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "B", Subjects: []string{"b.>"}, AllowDirect: true})
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name: "A", Subjects: []string{"a.>"}, RePublish: &jetstream.RePublish{Source: "a.>", Destination: "b.>"},
	})
	require.NoError(t, err)
	for range 10 {
		_, err = js.Publish(t.Context(), "b.pre", []byte("pre"))
		require.NoError(t, err)
	}
	for range 5 {
		_, err = js.Publish(t.Context(), "a.x", []byte("x"))
		require.NoError(t, err)
	}
	require.Eventually(t, func() bool {
		info, err := js.Stream(t.Context(), "B")
		return err == nil && info.CachedInfo().State.Msgs == 15
	}, 5*time.Second, 20*time.Millisecond)
	return dialClient(t, url)
}

func TestDirectRead_KeepsTheStoredNatsHeadersApartFromTheReadOnes(t *testing.T) {
	t.Parallel()
	c := republishedStream(t)

	msg, err := c.GetMessage(t.Context(), "B", 12)
	require.NoError(t, err)
	assert.Equal(t, uint64(12), msg.Sequence)
	assert.Equal(t, "b.x", msg.Subject)
	assert.Equal(t, "2", msg.Headers["Nats-Sequence"])
	assert.Equal(t, "A", msg.Headers["Nats-Stream"])
	assert.Equal(t, "a.x", msg.Headers["Nats-Subject"])
	assert.Equal(t, "1", msg.Headers["Nats-Last-Sequence"])

	for _, filter := range []string{"", "b.>"} {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		var got []*entities.Message
		for m, err := range c.ScanMessages(ctx, "B", entities.ScanOptions{FromSeq: 1, ToSeq: 15, SubjectFilter: filter, FetchMethod: "direct"}) {
			require.NoError(t, err)
			got = append(got, m)
		}
		cancel()
		require.Len(t, got, 15, "filter %q", filter)
		assert.Equal(t, uint64(15), got[14].Sequence, "filter %q", filter)
		assert.Equal(t, "b.x", got[14].Subject, "filter %q", filter)
		assert.Equal(t, "5", got[14].Headers["Nats-Sequence"], "filter %q", filter)
	}
}
