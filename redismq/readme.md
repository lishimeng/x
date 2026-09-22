# redismq

基于 Redis List 的轻量队列（`LPUSH` / `BRPOP`），客户端走 app-starter Redis。

## 用法

```go
mq := redismq.NewFromApp("mq:billing") // 或 redismq.New(app.GetRedisClient(), prefix)
_ = mq.Push(ctx, "jobs", payload)
body, err := mq.Pop(ctx, "jobs", time.Second)
```

队列 key：`{prefix}:{queue}`。
