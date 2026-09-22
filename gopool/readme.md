# gopool

有界动态工人池：`Do` 同步等待任务结束（适合事件 ACK）。

## 行为

- 工人数在 `[MinSize, MaxSize]`：无空闲且未达上限时扩容；空闲超过 `IdleTimeout` 且高于下限时缩减。
- 停池：取消 `New(ctx)` 的 `ctx`；工人退出，`Do` 返回 `ErrPoolClosed`。
- 用法错误（`New`/`Do` 的 `ctx` 或 `fn` 为 nil）直接 panic，不允许带错运行。

## 示例

```go
ctx, cancel := context.WithCancel(parent)
p := gopool.New(ctx, gopool.WithMinSize(2), gopool.WithMaxSize(8))
defer cancel()

err := p.Do(ctx, func(ctx context.Context) error {
    return handle(ctx)
})
```
