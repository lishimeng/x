package s3

import (
	"sync"
)

// Registry 供应器配置注册表，供内置 ProviderFactory 使用
type Registry struct {
	mu       sync.RWMutex
	fsRoots  map[Region]struct{}
	s3ByKey  map[string]ProviderConfig // key = category_region
}

func NewRegistry() *Registry {
	return &Registry{
		fsRoots: make(map[Region]struct{}),
		s3ByKey: make(map[string]ProviderConfig),
	}
}

// RegisterFsRoot 注册本地 fs 根目录（region）；每个根目录须显式注册
func (reg *Registry) RegisterFsRoot(root Region) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.fsRoots[root] = struct{}{}
}

// RegisterObject 注册 S3 兼容对象存储配置
func (reg *Registry) RegisterObject(cfg ProviderConfig) error {
	if !IsObjectCategory(cfg.category) {
		return ErrUnsupportedCategory
	}
	if cfg.bucket == "" {
		return ErrS3BucketRequired
	}
	reg.mu.Lock()
	defer reg.mu.Unlock()
	reg.s3ByKey[reg.objectKey(cfg.category, cfg.region)] = cfg
	return nil
}

func (reg *Registry) objectKey(category ProviderCategory, region Region) string {
	return string(category) + "_" + string(region)
}

func (reg *Registry) Factory() ProviderFactory {
	return func(category ProviderCategory, region Region) (Provider, error) {
		switch category {
		case Fs:
			reg.mu.RLock()
			_, ok := reg.fsRoots[region]
			reg.mu.RUnlock()
			if !ok {
				return nil, ErrFsRootNotAllowed
			}
			return NewProviderFs(region)
		default:
			if !IsObjectCategory(category) {
				return nil, ErrUnsupportedCategory
			}
			reg.mu.RLock()
			cfg, ok := reg.s3ByKey[reg.objectKey(category, region)]
			reg.mu.RUnlock()
			if !ok {
				return nil, ErrS3ConfigNotFound
			}
			return NewProviderS3(cfg)
		}
	}
}
