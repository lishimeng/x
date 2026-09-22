package s3

import (
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
)

var errFactoryDenied = errors.New("factory denied")

// 模拟 Factory：按 category+region 返回不同 Provider
func lookupTableFactory(roots map[Region]Region) ProviderFactory {
	return func(category ProviderCategory, region Region) (Provider, error) {
		switch category {
		case Fs:
			root, ok := roots[region]
			if !ok {
				return nil, ErrFsRootNotAllowed
			}
			return NewProviderFs(root)
		default:
			return nil, ErrUnsupportedCategory
		}
	}
}

func TestNewGateway_ProviderFactoryNil(t *testing.T) {
	_, err := NewGateway(nil)
	if err != ErrBuilderNil {
		t.Fatalf("err=%v want %v", err, ErrBuilderNil)
	}
}

func TestGateway_ProviderFactory(t *testing.T) {
	rootA := Region(filepath.Join(t.TempDir(), "uploads-a"))
	rootB := Region(filepath.Join(t.TempDir(), "uploads-b"))
	aliasA := Region("tenant-a")
	aliasB := Region("tenant-b")

	var calls int32
	factory := lookupTableFactory(map[Region]Region{
		aliasA: rootA,
		aliasB: rootB,
	})
	wrapped := ProviderFactory(func(category ProviderCategory, region Region) (Provider, error) {
		atomic.AddInt32(&calls, 1)
		return factory(category, region)
	})

	gw, err := NewGateway(wrapped)
	if err != nil {
		t.Fatal(err)
	}

	pA1, err := gw.Provider(Fs, aliasA)
	if err != nil {
		t.Fatal(err)
	}
	if pA1.Category() != Fs {
		t.Fatalf("category=%s want %s", pA1.Category(), Fs)
	}
	// Provider.Region() 为 Fs 根目录，与网关查询键 aliasA 不同
	if string(pA1.Region()) != string(rootA) {
		t.Fatalf("fs root=%s want %s", pA1.Region(), rootA)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("factory calls=%d want 1", calls)
	}

	// 同 category+region 命中缓存，不再调 factory
	pA2, err := gw.Provider(Fs, aliasA)
	if err != nil {
		t.Fatal(err)
	}
	if pA2 != pA1 {
		t.Fatal("cached provider instance should be reused")
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("factory calls=%d want 1 (cached)", calls)
	}

	// 不同 region 触发新 factory 调用
	pB, err := gw.Provider(Fs, aliasB)
	if err != nil {
		t.Fatal(err)
	}
	if pB == pA1 {
		t.Fatal("different region should yield different provider")
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("factory calls=%d want 2", calls)
	}

	// Factory 未命中：factory 报错，且不写入缓存
	_, err = gw.Provider(Fs, Region("unknown"))
	if err != ErrFsRootNotAllowed {
		t.Fatalf("err=%v want %v", err, ErrFsRootNotAllowed)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("factory calls=%d want 3", calls)
	}
	gw.mu.Lock()
	n := len(gw.providers)
	gw.mu.Unlock()
	if n != 2 {
		t.Fatalf("cached providers=%d want 2 (failed lookup must not cache)", n)
	}

	// 不支持 category
	_, err = gw.Provider(S3, Region("cn-hangzhou"))
	if err != ErrUnsupportedCategory {
		t.Fatalf("err=%v want %v", err, ErrUnsupportedCategory)
	}
}

func TestGateway_ProviderFactory_ErrorNotCached(t *testing.T) {
	var calls int32
	gw, err := NewGateway(func(category ProviderCategory, region Region) (Provider, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errFactoryDenied
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		_, err = gw.Provider(Fs, Region("any"))
		if err != errFactoryDenied {
			t.Fatalf("attempt %d: err=%v want %v", i+1, err, errFactoryDenied)
		}
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("factory calls=%d want 2 (errors are not cached)", calls)
	}
}
