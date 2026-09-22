package s3

import (
	"context"
	"io"
)

// ReadFn Open 回调：在 fn 执行期间可读 r，fn 返回后由 Open 关闭底层 reader
type ReadFn func(r io.Reader) error

// Provider 文件网关存储后端（本地 fs 或 S3 兼容对象存储）
type Provider interface {
	Save(ctx context.Context, key string, r io.Reader) (written int64, err error)
	Open(ctx context.Context, key string, fn ReadFn) error
	Remove(ctx context.Context, key string) error
	Category() ProviderCategory
	Region() Region
}

type BaseProvider struct {
	config ProviderConfig // 实现都必须在当前包中,保证不被外部修改
}

func (b *BaseProvider) Category() ProviderCategory { return b.config.category }
func (b *BaseProvider) Region() Region             { return b.config.region }

// ProviderFs 文件系统, region就是根目录, 每一个fs类型的实例必须显式配置region(不共享)
type ProviderFs struct {
	BaseProvider
}
