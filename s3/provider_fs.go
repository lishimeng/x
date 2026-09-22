package s3

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// NewProviderFs 创建本地文件系统 provider；region 为根目录，每个实例独立配置不共享
func NewProviderFs(root Region) (*ProviderFs, error) {
	root = Region(strings.TrimSpace(string(root)))
	if root == "" || root == "." {
		return nil, ErrFsRootRequired
	}
	root = Region(filepath.Clean(string(root)))
	if err := os.MkdirAll(string(root), 0o755); err != nil {
		return nil, err
	}
	return &ProviderFs{
		BaseProvider: BaseProvider{config: NewFsProviderConfig(root)},
	}, nil
}

func (p *ProviderFs) absPath(key string) (string, error) {
	rel, err := normalizeObjectKey(key)
	if err != nil {
		return "", err
	}
	root := filepath.Clean(string(p.config.region))
	abs := filepath.Join(root, filepath.FromSlash(rel))
	abs = filepath.Clean(abs)
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", ErrInvalidObjectKey
	}
	return abs, nil
}

func (p *ProviderFs) Save(ctx context.Context, key string, r io.Reader) (written int64, err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	abs, err := p.absPath(key)
	if err != nil {
		return
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	written, err = io.Copy(f, r)
	return
}

func (p *ProviderFs) Open(ctx context.Context, key string, fn ReadFn) error {
	if fn == nil {
		return ErrReadFnNil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	abs, err := p.absPath(key)
	if err != nil {
		return err
	}
	f, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()
	return fn(f)
}

func (p *ProviderFs) Remove(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	abs, err := p.absPath(key)
	if err != nil {
		return err
	}
	return os.Remove(abs)
}
