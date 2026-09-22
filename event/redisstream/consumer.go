package redisstream

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/lishimeng/app-starter/log"
	"github.com/lishimeng/x/event"
	"github.com/lishimeng/x/gopool"
)

// Consumer 预注册多 topic 消费者：链式 On，Run 单循环多 Stream BLOCK 读。
type Consumer struct {
	bus      *Bus
	group    string
	consumer string
	handlers map[string]event.Handler // topic -> handler
	topics   []string

	pool    *gopool.Pool
	poolMin int
	poolMax int

	mu sync.Mutex
}

// ConsumerOption 配置 Consumer 的 handler 池等。
type ConsumerOption func(*Consumer)

// WithHandlerPool 使用外部工人池（生命周期由该池的 New ctx 管理）。
func WithHandlerPool(p *gopool.Pool) ConsumerOption {
	return func(c *Consumer) {
		if p != nil {
			c.pool = p
		}
	}
}

// WithHandlerWorkers 使用内建动态池 min~max（在 Run 时用其 ctx 创建）。
func WithHandlerWorkers(minSize, maxSize int) ConsumerOption {
	return func(c *Consumer) {
		c.pool = nil
		c.poolMin = minSize
		c.poolMax = maxSize
	}
}

// Consumer 创建链式消费者。默认 handler 池 min=2 max=8（Run 时创建）。
func (b *Bus) Consumer(group, consumer string, opts ...ConsumerOption) *Consumer {
	c := &Consumer{
		bus:      b,
		group:    strings.TrimSpace(group),
		consumer: strings.TrimSpace(consumer),
		handlers: map[string]event.Handler{},
		poolMin:  2,
		poolMax:  8,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// On 注册 topic handler，返回自身以支持链式调用。
func (c *Consumer) On(topic string, h event.Handler) *Consumer {
	if c == nil {
		return c
	}
	topic = strings.TrimSpace(topic)
	if topic == "" || h == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.handlers[topic]; !ok {
		c.topics = append(c.topics, topic)
	}
	c.handlers[topic] = h
	return c
}

// Run 阻塞消费直到 ctx 取消。须至少 On 一个 topic。
func (c *Consumer) Run(ctx context.Context) error {
	if c == nil || c.bus == nil || c.bus.be == nil {
		return errors.New("event/redisstream: nil consumer")
	}
	if c.group == "" || c.consumer == "" {
		return errors.New("event/redisstream: group, consumer required")
	}
	c.mu.Lock()
	topics := append([]string(nil), c.topics...)
	handlers := make(map[string]event.Handler, len(c.handlers))
	for k, v := range c.handlers {
		handlers[k] = v
	}
	c.mu.Unlock()
	if len(topics) == 0 {
		return errors.New("event/redisstream: no topics registered")
	}
	if c.pool == nil {
		min, max := c.poolMin, c.poolMax
		if min < 1 {
			min = 2
		}
		if max < min {
			max = 8
			if max < min {
				max = min
			}
		}
		c.pool = gopool.New(ctx, gopool.WithMinSize(min), gopool.WithMaxSize(max))
	}

	streams := make([]string, 0, len(topics))
	streamToTopic := map[string]string{}
	for _, topic := range topics {
		sk := c.bus.streamKey(topic)
		streams = append(streams, sk)
		streamToTopic[sk] = topic
		log.Infof("event/redisstream consumer On stream=%s group=%s consumer=%s", sk, c.group, c.consumer)
		if err := c.bus.be.XGroupCreateMkStream(ctx, sk, c.group, "0"); err != nil {
			if !isBusyGroup(err) {
				return err
			}
		}
	}

	log.Infof("event/redisstream consumer Run streams=%d group=%s consumer=%s", len(streams), c.group, c.consumer)
	claimStarts := make([]string, len(streams))
	for i := range claimStarts {
		claimStarts[i] = "0-0"
	}
	var idleRounds int
	for {
		select {
		case <-ctx.Done():
			log.Infof("event/redisstream consumer ctx done group=%s: %v", c.group, ctx.Err())
			return ctx.Err()
		default:
		}

		msgs, err := c.pullMulti(ctx, streams, claimStarts)
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
					log.Infof("event/redisstream waiting multi group=%s streams=%d idle~%ds", c.group, len(streams), idleRounds)
				}
				continue
			}
			log.Infof("event/redisstream consumer pull err group=%s: %v", c.group, err)
			return err
		}
		if len(msgs) == 0 {
			idleRounds++
			if idleRounds == 1 || idleRounds%30 == 0 {
				log.Infof("event/redisstream waiting multi group=%s streams=%d idle~%ds", c.group, len(streams), idleRounds)
			}
			continue
		}
		idleRounds = 0
		log.Infof("event/redisstream consumer batch group=%s n=%d", c.group, len(msgs))

		type outcome struct {
			stream  string
			id      string
			envType string
			err     error
		}
		outcomes := make(chan outcome, len(msgs))
		var wg sync.WaitGroup
		for _, m := range msgs {
			stream := m.Stream
			topic := streamToTopic[stream]
			if topic == "" {
				topic = strings.TrimPrefix(stream, c.bus.prefix+":")
			}
			h := handlers[topic]
			env, e := decodeEnvelope(m)
			if e != nil {
				log.Infof("event/redisstream bad envelope id=%s stream=%s: %v (ack)", m.ID, stream, e)
				_ = c.bus.be.XAck(ctx, stream, c.group, m.ID)
				continue
			}
			if h == nil {
				log.Infof("event/redisstream no handler topic=%s id=%s (ack)", topic, m.ID)
				_ = c.bus.be.XAck(ctx, stream, c.group, m.ID)
				continue
			}
			wg.Add(1)
			msgID, envType := m.ID, env.Type
			go func(stream, msgID, envType string, env event.Envelope, h event.Handler) {
				defer wg.Done()
				err := c.pool.Do(ctx, func(wctx context.Context) error {
					return h(wctx, env)
				})
				outcomes <- outcome{stream: stream, id: msgID, envType: envType, err: err}
			}(stream, msgID, envType, env, h)
		}
		go func() {
			wg.Wait()
			close(outcomes)
		}()

		needBackoff := false
		for o := range outcomes {
			if o.err != nil {
				if event.IsNoAck(o.err) {
					log.Infof("event/redisstream handler no-ack id=%s type=%s: %v (hold pending)", o.id, o.envType, o.err)
					needBackoff = true
					continue
				}
				log.Infof("event/redisstream handler business err id=%s type=%s: %v (ack)", o.id, o.envType, o.err)
			}
			if ackErr := c.bus.be.XAck(ctx, o.stream, c.group, o.id); ackErr != nil {
				log.Infof("event/redisstream XACK fail id=%s: %v", o.id, ackErr)
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

func (c *Consumer) pullMulti(ctx context.Context, streams []string, claimStarts []string) ([]event.StreamMessage, error) {
	// 1) 各流本 consumer pending（非 BLOCK）
	for _, stream := range streams {
		msgs, err := c.bus.be.XReadGroup(ctx, stream, c.group, c.consumer, 8, 0, "0-0")
		if err != nil && !isNilOrTimeout(err) {
			return nil, err
		}
		if len(msgs) > 0 {
			for j := range msgs {
				if msgs[j].Stream == "" {
					msgs[j].Stream = stream
				}
			}
			log.Infof("event/redisstream pending(self) stream=%s n=%d", stream, len(msgs))
			return msgs, nil
		}
	}

	// 2) 逐流 autoclaim
	for i, stream := range streams {
		next, claimed, err := c.bus.be.XAutoClaim(ctx, stream, c.group, c.consumer, 30*time.Second, claimStarts[i], 8)
		if err != nil && !isNilOrTimeout(err) {
			log.Infof("event/redisstream XAUTOCLAIM err stream=%s: %v", stream, err)
			continue
		}
		if next != "" {
			claimStarts[i] = next
		}
		if len(claimed) > 0 {
			for j := range claimed {
				if claimed[j].Stream == "" {
					claimed[j].Stream = stream
				}
			}
			return claimed, nil
		}
		if next == "0-0" || next == "" {
			claimStarts[i] = "0-0"
		}
	}

	// 3) 多流 BLOCK 等新消息
	ids := make([]string, len(streams))
	for i := range ids {
		ids[i] = ">"
	}
	return c.bus.be.XReadGroupMulti(ctx, streams, c.group, c.consumer, 8, time.Second, ids)
}
