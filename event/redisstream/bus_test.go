package redisstream

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lishimeng/x/event"
)

type memBackend struct {
	mu      sync.Mutex
	streams map[string][]event.StreamMessage
	groups  map[string]map[string]int // stream -> group -> next index
	seq     int
}

func newMemBackend() *memBackend {
	return &memBackend{
		streams: map[string][]event.StreamMessage{},
		groups:  map[string]map[string]int{},
	}
}

func (m *memBackend) XAdd(_ context.Context, stream string, values map[string]string, maxLen int64) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	id := strconvID(m.seq)
	cp := map[string]string{}
	for k, v := range values {
		cp[k] = v
	}
	m.streams[stream] = append(m.streams[stream], event.StreamMessage{ID: id, Stream: stream, Values: cp})
	if maxLen > 0 {
		msgs := m.streams[stream]
		if int64(len(msgs)) > maxLen {
			m.streams[stream] = append([]event.StreamMessage(nil), msgs[len(msgs)-int(maxLen):]...)
		}
	}
	return id, nil
}

func (m *memBackend) XGroupCreateMkStream(_ context.Context, stream, group, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.groups[stream] == nil {
		m.groups[stream] = map[string]int{}
	}
	if _, ok := m.groups[stream][group]; ok {
		return errors.New("BUSYGROUP Consumer Group name already exists")
	}
	m.groups[stream][group] = 0
	if m.streams[stream] == nil {
		m.streams[stream] = nil
	}
	return nil
}

func (m *memBackend) XReadGroup(ctx context.Context, stream, group, _ string, count int64, block time.Duration, id string) ([]event.StreamMessage, error) {
	if id == "0" || id == "0-0" {
		// mem backend 无独立 PEL，视为无 pending
		return nil, errors.New("redis: nil")
	}
	deadline := time.Now().Add(block)
	for {
		m.mu.Lock()
		g := m.groups[stream]
		if g == nil {
			m.mu.Unlock()
			return nil, errors.New("NOGROUP")
		}
		idx := g[group]
		msgs := m.streams[stream]
		if idx < len(msgs) {
			end := idx + int(count)
			if end > len(msgs) {
				end = len(msgs)
			}
			out := append([]event.StreamMessage(nil), msgs[idx:end]...)
			for i := range out {
				out[i].Stream = stream
			}
			g[group] = end
			m.mu.Unlock()
			return out, nil
		}
		m.mu.Unlock()
		if block <= 0 || time.Now().After(deadline) {
			return nil, errors.New("redis: nil")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func (m *memBackend) XReadGroupMulti(ctx context.Context, streams []string, group, consumer string, count int64, block time.Duration, ids []string) ([]event.StreamMessage, error) {
	if len(streams) == 0 {
		return nil, nil
	}
	if len(ids) != len(streams) {
		return nil, errors.New("streams/ids length mismatch")
	}
	deadline := time.Now().Add(block)
	for {
		var out []event.StreamMessage
		for i, stream := range streams {
			id := ids[i]
			if id == "0" || id == "0-0" {
				continue
			}
			msgs, err := m.XReadGroup(ctx, stream, group, consumer, count, 0, id)
			if err != nil {
				if stringsContainsNil(err) {
					continue
				}
				return nil, err
			}
			out = append(out, msgs...)
			if int64(len(out)) >= count {
				return out[:count], nil
			}
		}
		if len(out) > 0 {
			return out, nil
		}
		if block <= 0 || time.Now().After(deadline) {
			return nil, errors.New("redis: nil")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func stringsContainsNil(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return s == "redis: nil" || strings.Contains(s, "nil")
}

func (m *memBackend) XAutoClaim(context.Context, string, string, string, time.Duration, string, int64) (string, []event.StreamMessage, error) {
	return "0-0", nil, nil
}

func (m *memBackend) XPendingIDs(context.Context, string, string, int64) ([]string, error) {
	return nil, nil
}

func (m *memBackend) XClaim(context.Context, string, string, string, time.Duration, ...string) ([]event.StreamMessage, error) {
	return nil, nil
}

func (m *memBackend) XAck(context.Context, string, string, ...string) error { return nil }

func strconvID(n int) string {
	return time.Now().UTC().Format("20060102150405") + "-" + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestBusPublishSubscribeFanout(t *testing.T) {
	be := newMemBackend()
	bus := New(be, "evt")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gotA := make(chan string, 1)
	gotB := make(chan string, 1)

	go func() {
		_ = bus.Subscribe(ctx, "demo", "group-a", "c1", func(_ context.Context, env event.Envelope) error {
			gotA <- string(env.Payload)
			return nil
		})
	}()
	go func() {
		_ = bus.Subscribe(ctx, "demo", "group-b", "c1", func(_ context.Context, env event.Envelope) error {
			gotB <- string(env.Payload)
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)
	if err := bus.Publish(ctx, "demo", event.Envelope{Type: "demo.t", Payload: []byte("hello")}); err != nil {
		t.Fatal(err)
	}

	select {
	case v := <-gotA:
		if v != "hello" {
			t.Fatalf("group-a got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("group-a timeout")
	}
	select {
	case v := <-gotB:
		if v != "hello" {
			t.Fatalf("group-b got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("group-b timeout")
	}
}

func TestConsumerOnRunMultiTopic(t *testing.T) {
	be := newMemBackend()
	bus := New(be, "evt")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gotA := make(chan string, 1)
	gotB := make(chan string, 1)

	go func() {
		_ = bus.Consumer("g1", "c1").
			On("topic.a", func(_ context.Context, env event.Envelope) error {
				gotA <- string(env.Payload)
				return nil
			}).
			On("topic.b", func(_ context.Context, env event.Envelope) error {
				gotB <- string(env.Payload)
				return nil
			}).
			Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	if err := bus.Publish(ctx, "topic.a", event.Envelope{Type: "topic.a", Payload: []byte("A")}); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish(ctx, "topic.b", event.Envelope{Type: "topic.b", Payload: []byte("B")}); err != nil {
		t.Fatal(err)
	}

	select {
	case v := <-gotA:
		if v != "A" {
			t.Fatalf("topic.a got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("topic.a timeout")
	}
	select {
	case v := <-gotB:
		if v != "B" {
			t.Fatalf("topic.b got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("topic.b timeout")
	}
	cancel()
}
