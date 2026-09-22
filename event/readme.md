# event

跨进程事件总线抽象与 Redis Stream 风格实现。

## 语义

- **Publish**：写入流（topic）
- **Subscribe**：按 **consumer group** 独立消费；同一 topic 上每个 group 都会收到副本（fan-out）
- Handler **须幂等**

### 业务异常 vs 通信异常

| 类型 | 用法 | 框架 ACK |
|------|------|----------|
| **业务异常** | 业务自行 `log` 后 `return nil`；或 `return err` 仅供框架打印 | **ACK**（不重试） |
| **通信异常** | `return event.NoAck(err)`（DB/Redis/超时等） | **不 ACK**，pending 重试 |

```go
// 业务：缺套餐 —— 只记日志，流程算处理完
log.Infof("skip: plan missing")
return nil

// 通信：写库失败 —— 控制 ACK
return event.NoAck(fmt.Errorf("db: %w", err))
```

## 用法

```go
bus := redisstream.New(backend, "event") // Bus 默认 MAXLEN ~ 10000（Envelope.MaxLen==0 时）
_ = bus.Publish(ctx, "org.certified", event.Envelope{
	Type: "org.certified", Payload: payloadJSON, MaxLen: 10000,
})

// 单 topic（每订阅一个 goroutine / 一条 BLOCK 连接）
_ = bus.Subscribe(ctx, "org.certified", "my-service", "instance-1", handler)

// 多 topic：链式 On + Run；单循环多 Stream XREADGROUP BLOCK，handler 走 gopool
_ = bus.Consumer("my-service", "instance-1").
	On("org.certified", handleOrgCertified).
	On("order.purchased", handleOrderPurchased).
	Run(ctx)
```

`Consumer` 默认 handler 池 min=2 max=8（`WithHandlerWorkers` / `WithHandlerPool` 可改）。

业务目录用 `event.Def` 把类型与 MaxLen 写在一起（见 `internal/ddd/events`）：

```go
EventOrgCertified = event.Def{Type: "org.certified", MaxLen: 10000}
```

`Backend` 由基础设施适配具体 Redis 客户端，本包不绑定驱动。

## 留存

Publish 经 `Backend.XAdd(..., maxLen)` 近似封顶（Redis：`MAXLEN ~`）：

| 优先级 | 来源 |
|--------|------|
| 1 | `Envelope.MaxLen`（>0，通常来自 `event.Def`） |
| 2 | Bus：`DefaultMaxLen` / `WithMaxLen` / `EVENT_STREAM_MAXLEN` |

Stream 是投递缓冲，不是业务存档；业务结果落库。
