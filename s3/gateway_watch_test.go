package s3

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestGateway_PurgeExpired(t *testing.T) {
	var calls int32
	factory := func(category ProviderCategory, region Region) (Provider, error) {
		atomic.AddInt32(&calls, 1)
		return NewProviderFs(region)
	}
	gw, err := NewGateway(factory)
	if err != nil {
		t.Fatal(err)
	}
	gw.watchInterval = time.Millisecond * 20
	root := Region(t.TempDir())

	p1, err := gw.Provider(Fs, root)
	if err != nil || p1 == nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("factory calls=%d want 1", calls)
	}

	p2, err := gw.Provider(Fs, root)
	if err != nil || p2 == nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("factory calls=%d want 1 (cached)", calls)
	}

	key := gw.genProviderKey(Fs, root)
	gw.mu.Lock()
	gw.providers[key].lastUsed = time.Now().Add(-DefaultProviderExpired - time.Second)
	gw.mu.Unlock()

	gw.PurgeExpired()

	gw.mu.Lock()
	_, ok := gw.providers[key]
	gw.mu.Unlock()
	if ok {
		t.Fatal("expired provider should be removed")
	}

	_, err = gw.Provider(Fs, root)
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Fatalf("factory calls=%d want 2 after purge", calls)
	}
}

func TestGateway_StartWatch(t *testing.T) {
	gw, err := NewGateway(func(category ProviderCategory, region Region) (Provider, error) {
		return NewProviderFs(region)
	})
	if err != nil {
		t.Fatal(err)
	}
	gw.watchInterval = time.Millisecond * 30
	root := Region(t.TempDir())
	if _, err = gw.Provider(Fs, root); err != nil {
		t.Fatal(err)
	}
	key := gw.genProviderKey(Fs, root)
	gw.mu.Lock()
	gw.providers[key].lastUsed = time.Now().Add(-DefaultProviderExpired - time.Second)
	gw.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gw.StartWatch(ctx)
	time.Sleep(time.Millisecond * 80)

	gw.mu.Lock()
	_, ok := gw.providers[key]
	gw.mu.Unlock()
	if ok {
		t.Fatal("watch should remove expired provider")
	}
}
