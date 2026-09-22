# pkg/s3 — 文件存储网关

统一抽象**本地文件系统**与**S3 兼容对象存储**（AWS S3、阿里云 OSS、腾讯云 COS 等），通过 `Gateway` + `Provider` 读写文件，而不直接依赖具体 SDK。

---

## 架构

```
ProviderFactory
        │
        ▼
   Gateway（按 category+region 缓存 Provider）
        │
        ├── ProviderFs   本地目录
        └── ProviderS3   minio-go 客户端（S3 API 兼容）
```

| 类型 | 说明 |
|------|------|
| `Gateway` | 入口；`Provider(category, region)` 获取客户端，未命中时调 Factory 创建并缓存 |
| `ProviderFactory` | `func(category, region) (Provider, error)` |
| `Provider` | `Save` / `Open` / `Remove` |
| `Registry` | 可选参考实现：内存注册表 + 内置 `Factory()`，适合单测或简单部署 |
| `Region` | 对象存储为云厂商地域码；`fs` 类型为**本地根目录**（绝对或相对路径） |
| `Endpoint` | 访问地址；对象存储由 `SupportedCategory` 模板 + `region` 生成 |

---

## ProviderCategory（供应商类型）

常量定义见 `common.go`，`SupportedCategory` 在 `init()` 中注册各厂商 **Endpoint 模板**（`{region}` 占位符）。

| Category | 含义 | Endpoint 模板（示例） |
|----------|------|------------------------|
| `fs` | 本地文件系统 | `{region}`（region = 根目录路径） |
| `s3` | AWS S3 | `https://s3.{region}.amazonaws.com` |
| `oss` | 阿里云 OSS | `https://s3.oss-{region}.aliyuncs.com` |
| `cos` | 腾讯云 COS | `https://cos.{region}.myqcloud.com` |
| `obs` | 华为云 OBS | `https://obs.{region}.myhuaweicloud.com` |
| `tos` | 火山引擎 TOS | `https://tos-s3-{region}.volces.com` |
| `jdos` | 京东云 OSS | `https://s3.{region}.jdcloud-oss.com` |
| `kodos` | 七牛云 Kodo | `https://s3-{region}.qiniucs.com` |
| `bos` | 百度云 BOS | `https://s3.{region}.bcebos.com` |

- `IsObjectCategory(c)`：`s3` / `oss` / `cos` 等返回 `true`；`fs` 为 `false`。
- 自定义 Endpoint：通过 `NewObjectProviderConfig` 写入配置；模板仅用于按地域生成默认 host。

---

## Provider 接口

```go
type Provider interface {
    Save(ctx context.Context, key string, r io.Reader) (written int64, err error)
    Open(ctx context.Context, key string, fn ReadFn) error   // fn 返回后关闭底层 reader
    Remove(ctx context.Context, key string) error
    Category() ProviderCategory
    Region() Region
}
```

### 对象键 `key`

`normalizeObjectKey` 统一规则（`fs` 与对象存储一致）：

- 相对路径，使用 `/` 作为分隔符
- 禁止 `..`、绝对路径、以 `/` 开头
- 空 key → `ErrInvalidObjectKey`

`ProviderFs` 在 `region` 根目录下拼接路径，并校验结果仍在根目录内（防目录穿越）。

---

## 快速开始

### 方式一：Registry（参考实现）

```go
reg := s3.NewRegistry()
reg.RegisterFsRoot(s3.Region("/data/uploads"))

cfg, _ := s3.NewObjectProviderConfig(s3.COS, s3.Region("ap-guangzhou"), "my-bucket", s3.Auth{
    AccessKey: "AK...",
    SecretKey: "SK...",
})
_ = reg.RegisterObject(cfg)

gw, _ := s3.NewGateway(reg.Factory())
gw.StartWatch(ctx)

p, _ := gw.Provider(s3.Fs, s3.Region("/data/uploads"))
_, _ = p.Save(ctx, "2026/05/report/a.jpg", reader)
_ = p.Open(ctx, "2026/05/report/a.jpg", func(r io.Reader) error {
    _, err := io.ReadAll(r)
    return err
})
```

注意：

- **每个 `fs` 根目录须 `RegisterFsRoot` 显式注册**，未注册返回 `ErrFsRootNotAllowed`。
- **对象存储须 `RegisterObject`**，按 `category + region` 查表，未命中返回 `ErrS3ConfigNotFound`。

### 方式二：自定义 Factory

```go
factory := func(category s3.ProviderCategory, region s3.Region) (s3.Provider, error) {
    cfg, err := s3.NewObjectProviderConfig(category, region, bucket, auth)
    if err != nil {
        return nil, err
    }
    return s3.NewProviderS3(cfg)
}
gw, _ := s3.NewGateway(factory)
gw.StartWatch(ctx)
```

---

## 预签名（PresignProvider）

`ProviderS3` 额外实现 `PresignProvider`（`presign.go`）：

| 方法 | 用途 |
|------|------|
| `PresignPost` | 表单 POST 直传（如小程序 `wx.uploadFile` + `formData`） |
| `PresignGet` | 临时读 URL（私有 bucket 展示图片） |

默认有效期：POST 15 分钟，GET 1 小时（`expiry <= 0` 时）。

```go
if ps, ok := p.(s3.PresignProvider); ok {
    url, _ := ps.PresignGet(ctx, key, time.Hour)
}
```

---

## Gateway 缓存与过期

| 常量 | 默认值 | 含义 |
|------|--------|------|
| `DefaultProviderExpired` | 1 小时 | 实例上次使用后超过该时间视为过期 |
| `DefaultWatchInterval` | 5 分钟 | 后台巡检间隔 |

`StartWatch(ctx)` 定期 `purgeExpired()`，从 map 移除过期 `ProviderWrapper`（不关闭底层连接，仅丢弃缓存引用；下次 `Provider()` 会重新 Factory）。

测试或运维可手动调用 `gw.PurgeExpired()`。

---

## 错误一览

| 错误 | 含义 |
|------|------|
| `ErrBuilderNil` | `NewGateway(nil)` |
| `ErrFsRootRequired` | fs 根目录为空或 `.` |
| `ErrFsRootNotAllowed` | Registry 未注册该 fs 根目录 |
| `ErrS3ConfigNotFound` | Registry / Factory 无对应对象存储配置 |
| `ErrS3BucketRequired` | bucket 为空 |
| `ErrUnsupportedCategory` | 不支持的 category |
| `ErrInvalidObjectKey` | key 非法（空、绝对路径、`..` 等） |
| `ErrReadFnNil` | `Open` 未传回调 |

---

## 依赖与实现说明

- **对象存储客户端**：[`minio-go/v7`](https://github.com/minio/minio-go)（S3 兼容 API）
- **COS**：`NewProviderS3` 对 `cos` 设置 `BucketLookup = DNS`
- **STS**：`security.token.service.go` 中 `GenSts()` 为占位，尚未实现临时凭证下发

---

## 测试

```bash
go test ./pkg/s3/...
```

覆盖：Registry / Gateway 缓存与过期、Factory 注入、fs 路径安全、`normalizeObjectKey`、Endpoint 模板格式化等。

---

## 源码索引

| 文件 | 说明 |
|------|------|
| `gateway.go` | Gateway、ProviderFactory、缓存与 Watch |
| `provider.go` | Provider 接口、BaseProvider、ProviderFs 类型 |
| `provider_fs.go` | 本地文件实现 |
| `provider_s3.go` | S3 兼容实现 + Presign |
| `registry.go` | 内存注册表 + 参考 Factory |
| `config.go` | `ProviderConfig`、`NewFsProviderConfig`、`NewObjectProviderConfig` |
| `common.go` | Category 常量、`SupportedCategory`、公共错误 |
| `key.go` | 对象键规范化 |
| `presign.go` | Presign 接口与结果类型 |
| `endpoint.tpl.go` | Endpoint 模板 `{region}` 替换 |
| `doc.go` | 包级 godoc 摘要 |
