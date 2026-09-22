package s3

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lishimeng/app-starter/log"
)

// ProviderCategory 供应商类型
type ProviderCategory string

type Region string // 地域（fs 类型下为根目录）

type Endpoint string

type ProviderWrapper struct {
	provider Provider
	region   Region
	category ProviderCategory
	lastUsed time.Time
}

func (pw *ProviderWrapper) Expired() bool {
	return time.Since(pw.lastUsed) > DefaultProviderExpired
}

func (pw *ProviderWrapper) RefreshExpired() {
	pw.lastUsed = time.Now()
}

type Auth struct {
	AccessKey string
	SecretKey string
}

type Sts struct {
	Token string
	Ttl   int64
}

// ProviderFactory 按 category+region 构建 Provider 实例。
type ProviderFactory func(category ProviderCategory, region Region) (Provider, error)

// Gateway 按 category+region 缓存 Provider 实例。
type Gateway struct {
	mu            sync.Mutex
	providers     map[string]*ProviderWrapper
	builder       ProviderFactory
	watchInterval time.Duration
}

// NewGateway 使用 factory 创建网关。
func NewGateway(factory ProviderFactory) (*Gateway, error) {
	if factory == nil {
		return nil, ErrBuilderNil
	}
	return &Gateway{
		providers:     make(map[string]*ProviderWrapper),
		builder:       factory,
		watchInterval: DefaultWatchInterval,
	}, nil
}

func (g *Gateway) genProviderKey(category ProviderCategory, region Region) string {
	return fmt.Sprintf("provider_%s_%s", category, region)
}

// Provider 按 category+region 获取 Provider；缓存未命中时调用 Factory 创建并缓存。
func (g *Gateway) Provider(category ProviderCategory, region Region) (Provider, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.getProviderLocked(category, region)
}

func (g *Gateway) getProviderLocked(category ProviderCategory, region Region) (Provider, error) {
	key := g.genProviderKey(category, region)
	if pw, ok := g.providers[key]; ok {
		pw.RefreshExpired()
		return pw.provider, nil
	}
	p, err := g.builder(category, region)
	if err != nil {
		log.Debugf("provider factory failed category=%s region=%s: %v", category, region, err)
		return nil, err
	}
	g.providers[key] = &ProviderWrapper{
		provider: p,
		region:   region,
		category: category,
		lastUsed: time.Now(),
	}
	return p, nil
}

// StartWatch 后台巡检：ProviderWrapper.Expired() 为 true 的实例从缓存移除
func (g *Gateway) StartWatch(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	interval := g.watchInterval
	if interval <= 0 {
		interval = DefaultWatchInterval
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g.purgeExpired()
			}
		}
	}()
}

func (g *Gateway) purgeExpired() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for key, pw := range g.providers {
		if pw.Expired() {
			delete(g.providers, key)
			log.Debugf("provider expired removed: %s", key)
		}
	}
}

// PurgeExpired 供测试或手动触发过期清理
func (g *Gateway) PurgeExpired() {
	g.purgeExpired()
}
