package redisstream

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lishimeng/x/event"
)

const (
	fieldType       = "type"
	fieldOccurredAt = "occurred_at"
	fieldPayload    = "payload"
	headerPrefix    = "h:"
)

// Bus 基于 Backend 的 Redis Stream 风格事件总线（每 group 独立消费）。
type Bus struct {
	be     event.Backend
	prefix string
	maxLen int64
}

// New 创建总线。prefix 用于 stream key：{prefix}:{topic}；空则用 "event"。
// 默认 WithMaxLen(DefaultMaxLen)；可用 Option 覆盖。
func New(be event.Backend, prefix string, opts ...Option) *Bus {
	if prefix == "" {
		prefix = "event"
	}
	b := &Bus{
		be:     be,
		prefix: strings.TrimRight(prefix, ":"),
		maxLen: DefaultMaxLen,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

// MaxLen 当前单流近似上限（0 表示不裁剪）。
func (b *Bus) MaxLen() int64 {
	if b == nil {
		return 0
	}
	return b.maxLen
}

func (b *Bus) streamKey(topic string) string {
	topic = strings.TrimSpace(topic)
	return fmt.Sprintf("%s:%s", b.prefix, topic)
}

// Publish 实现 event.Publisher。
func (b *Bus) Publish(ctx context.Context, topic string, env event.Envelope) error {
	if b == nil || b.be == nil {
		return errors.New("event/redisstream: nil bus")
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return errors.New("event/redisstream: topic required")
	}
	typ := strings.TrimSpace(env.Type)
	if typ == "" {
		return errors.New("event/redisstream: envelope type required")
	}
	occurred := env.OccurredAt
	if occurred.IsZero() {
		occurred = time.Now()
	}
	values := map[string]string{
		fieldType:       typ,
		fieldOccurredAt: occurred.UTC().Format(time.RFC3339Nano),
		fieldPayload:    string(env.Payload),
	}
	for k, v := range env.Headers {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		values[headerPrefix+k] = v
	}
	_, err := b.be.XAdd(ctx, b.streamKey(topic), values, resolveMaxLen(b.maxLen, env.MaxLen))
	return err
}

func resolveMaxLen(busDefault, envelope int64) int64 {
	if envelope > 0 {
		return envelope
	}
	return busDefault
}

// Subscribe 实现 event.Subscriber。
func (b *Bus) Subscribe(ctx context.Context, topic, group, consumer string, h event.Handler) error {
	if b == nil || b.be == nil {
		return errors.New("event/redisstream: nil bus")
	}
	if h == nil {
		return errors.New("event/redisstream: handler required")
	}
	topic = strings.TrimSpace(topic)
	group = strings.TrimSpace(group)
	consumer = strings.TrimSpace(consumer)
	if topic == "" || group == "" || consumer == "" {
		return errors.New("event/redisstream: topic, group, consumer required")
	}
	stream := b.streamKey(topic)
	Infof("event/redisstream subscribe enter stream=%s group=%s consumer=%s", stream, group, consumer)
	// 从开头消费历史（含离线期间消息）；已存在的 group 忽略 BUSYGROUP。
	if err := b.be.XGroupCreateMkStream(ctx, stream, group, "0"); err != nil {
		if !isBusyGroup(err) {
			Infof("event/redisstream XGROUP CREATE fail stream=%s group=%s: %v", stream, group, err)
			return err
		}
		Infof("event/redisstream group exists stream=%s group=%s", stream, group)
	} else {
		Infof("event/redisstream group created stream=%s group=%s start=0", stream, group)
	}
	Infof("event/redisstream read loop start stream=%s (pending+claim then >)", stream)
	var idleRounds int
	claimStart := "0-0"
	for {
		select {
		case <-ctx.Done():
			Infof("event/redisstream subscribe ctx done stream=%s: %v", stream, ctx.Err())
			return ctx.Err()
		default:
		}

		msgs, err := b.pullBatch(ctx, stream, group, consumer, &claimStart)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				continue
			}
			if isNilOrTimeout(err) {
				idleRounds++
				if idleRounds == 1 || idleRounds%30 == 0 {
					Infof("event/redisstream waiting stream=%s group=%s idle~%ds", stream, group, idleRounds)
				}
				continue
			}
			Infof("event/redisstream pull err stream=%s: %v", stream, err)
			return err
		}
		if len(msgs) == 0 {
			idleRounds++
			if idleRounds == 1 || idleRounds%30 == 0 {
				Infof("event/redisstream waiting stream=%s group=%s idle~%ds", stream, group, idleRounds)
			}
			continue
		}
		idleRounds = 0
		Infof("event/redisstream batch stream=%s n=%d", stream, len(msgs))
		needBackoff := false
		for _, m := range msgs {
			env, e := decodeEnvelope(m)
			if e != nil {
				Infof("event/redisstream bad envelope id=%s: %v (ack)", m.ID, e)
				_ = b.be.XAck(ctx, stream, group, m.ID)
				continue
			}
			if e = h(ctx, env); e != nil {
				if event.IsNoAck(e) {
					Infof("event/redisstream handler no-ack id=%s type=%s: %v (hold pending, backoff)", m.ID, env.Type, e)
					needBackoff = true
					continue
				}
				// 业务异常：仅日志，ACK
				Infof("event/redisstream handler business err id=%s type=%s: %v (ack)", m.ID, env.Type, e)
				if ackErr := b.be.XAck(ctx, stream, group, m.ID); ackErr != nil {
					Infof("event/redisstream XACK fail id=%s: %v", m.ID, ackErr)
				}
				continue
			}
			if e = b.be.XAck(ctx, stream, group, m.ID); e != nil {
				Infof("event/redisstream XACK fail id=%s: %v", m.ID, e)
			}
		}
		if needBackoff {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
		}
	}
}

// pullBatch 优先 pending / claim，最后阻塞读新消息。
func (b *Bus) pullBatch(ctx context.Context, stream, group, consumer string, claimStart *string) ([]event.StreamMessage, error) {
	// 1) 本 consumer 未 ACK 的 pending
	msgs, err := b.be.XReadGroup(ctx, stream, group, consumer, 8, 0, "0-0")
	if err != nil && !isNilOrTimeout(err) {
		Infof("event/redisstream pending read err stream=%s: %v", stream, err)
		return nil, err
	}
	if len(msgs) > 0 {
		Infof("event/redisstream pending(self) stream=%s n=%d first=%s", stream, len(msgs), msgs[0].ID)
		return msgs, nil
	}

	// 2) XAUTOCLAIM：仅认领空闲 >=30s 的 pending，避免本进程刚失败又立刻重抢
	next, claimed, err := b.be.XAutoClaim(ctx, stream, group, consumer, 30*time.Second, *claimStart, 8)
	if err != nil && !isNilOrTimeout(err) {
		Infof("event/redisstream XAUTOCLAIM err stream=%s: %v", stream, err)
	} else {
		if next != "" {
			*claimStart = next
		}
		if len(claimed) > 0 {
			Infof("event/redisstream autoclaim stream=%s n=%d first=%s next=%s", stream, len(claimed), claimed[0].ID, next)
			return claimed, nil
		}
		if next == "0-0" || next == "" {
			*claimStart = "0-0"
		}
	}

	// 3) XPENDING + XCLAIM（minIdle=30s）兜底
	ids, err := b.be.XPendingIDs(ctx, stream, group, 8)
	if err != nil && !isNilOrTimeout(err) {
		Infof("event/redisstream XPENDING err stream=%s: %v", stream, err)
	} else if len(ids) > 0 {
		claimed, err = b.be.XClaim(ctx, stream, group, consumer, 30*time.Second, ids...)
		if err != nil {
			Infof("event/redisstream XCLAIM err stream=%s: %v", stream, err)
		} else if len(claimed) > 0 {
			Infof("event/redisstream XCLAIM stream=%s n=%d first=%s", stream, len(claimed), claimed[0].ID)
			return claimed, nil
		}
	}

	// 4) 阻塞等新消息
	return b.be.XReadGroup(ctx, stream, group, consumer, 8, time.Second, ">")
}

func decodeEnvelope(m event.StreamMessage) (event.Envelope, error) {
	typ := strings.TrimSpace(m.Values[fieldType])
	if typ == "" {
		return event.Envelope{}, errors.New("missing type")
	}
	env := event.Envelope{
		ID:      m.ID,
		Type:    typ,
		Payload: []byte(m.Values[fieldPayload]),
		Headers: map[string]string{},
	}
	if raw := strings.TrimSpace(m.Values[fieldOccurredAt]); raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			env.OccurredAt = t
		} else if t, err := time.Parse(time.RFC3339, raw); err == nil {
			env.OccurredAt = t
		} else if ms, err := strconv.ParseInt(raw, 10, 64); err == nil {
			env.OccurredAt = time.UnixMilli(ms)
		}
	}
	for k, v := range m.Values {
		if strings.HasPrefix(k, headerPrefix) {
			env.Headers[strings.TrimPrefix(k, headerPrefix)] = v
		}
	}
	return env, nil
}

func isBusyGroup(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "BUSYGROUP")
}

func isNilOrTimeout(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return s == "redis: nil" || strings.Contains(s, "i/o timeout") || strings.Contains(s, "nil")
}
