// Package event 提供跨进程事件总线抽象：发布 / 按消费者组订阅。
//
// 本包不含业务事件名或载荷定义；业务目录放在使用方。
//
// Handler 两类结果：
//   - 业务结果：自行打日志后 return nil；若 return 普通 error，框架仅打印后 ACK（不重试）
//   - 通信/基础设施失败：return event.NoAck(err)，框架不 ACK，保留 pending 重试
//
// Handler 须幂等。
package event

import (
	"context"
	"time"
)

// Envelope 事件信封（传输层）。
type Envelope struct {
	ID         string            // 后端生成的消息 ID（发布时可空）
	Type       string            // 事件类型（由业务约定）
	OccurredAt time.Time         // 业务发生时间
	Payload    []byte            // 业务载荷（通常 JSON）
	Headers    map[string]string // 可选扩展头
	MaxLen     int64             // 该次写入流近似上限；0 表示跟随 Bus 默认
}

// Def 业务事件目录项（名称与该 topic 留存策略，由业务包在常量旁声明）。
type Def struct {
	Type   string // Envelope.Type / topic
	MaxLen int64  // Publish 时写入流的近似上限；0 跟随 Bus 默认
}

// Topic 默认与 Type 同名。
func (d Def) Topic() string {
	return d.Type
}

// Handler 处理一条事件。
// 业务异常：日志用，return nil 或普通 error（框架 ACK）。
// 通信异常：return NoAck(err)（框架不 ACK）。
type Handler func(ctx context.Context, env Envelope) error

// Publisher 发布事件到 topic。
type Publisher interface {
	Publish(ctx context.Context, topic string, env Envelope) error
}

// Subscriber 以独立消费者组订阅 topic（同 topic 上每个 group 都会收到副本）。
type Subscriber interface {
	// Subscribe 阻塞直到 ctx 取消。group 区分监听者；consumer 为组内实例名。
	Subscribe(ctx context.Context, topic, group, consumer string, h Handler) error
}

// StreamMessage Backend 读到的原始消息。
type StreamMessage struct {
	ID     string
	Stream string // 所属 stream key（多流读取时用于路由）
	Values map[string]string
}

// Backend 底层流操作（由基础设施适配 Redis 等，本包不绑定具体驱动）。
type Backend interface {
	// XAdd 写入流。maxLen>0 时近似封顶（如 Redis MAXLEN ~）；0 不裁剪。
	XAdd(ctx context.Context, stream string, values map[string]string, maxLen int64) (id string, err error)
	XGroupCreateMkStream(ctx context.Context, stream, group, start string) error
	// XReadGroup id 为 ">"（仅新消息）或 "0"/"0-0"（本 consumer 的 pending）。
	XReadGroup(ctx context.Context, stream, group, consumer string, count int64, block time.Duration, id string) ([]StreamMessage, error)
	// XReadGroupMulti 一次读取多个 stream；ids 与 streams 等长（多为 ">" 或 "0-0"）。
	// block>0 且 ids 均为 ">" 时带 BLOCK。返回的 StreamMessage.Stream 为对应 stream key。
	XReadGroupMulti(ctx context.Context, streams []string, group, consumer string, count int64, block time.Duration, ids []string) ([]StreamMessage, error)
	// XAutoClaim 认领组内空闲过久的 pending（含其他 consumer）。
	XAutoClaim(ctx context.Context, stream, group, consumer string, minIdle time.Duration, start string, count int64) (nextStart string, msgs []StreamMessage, err error)
	// XPendingIDs 列出组内 pending 的消息 ID（最多 count 条）。
	XPendingIDs(ctx context.Context, stream, group string, count int64) (ids []string, err error)
	// XClaim 强制认领指定 pending（minIdle=0 表示立刻认领）。
	XClaim(ctx context.Context, stream, group, consumer string, minIdle time.Duration, ids ...string) ([]StreamMessage, error)
	XAck(ctx context.Context, stream, group string, ids ...string) error
}

var Debug = false

func SetDebug(debug bool) {
	Debug = debug
}
