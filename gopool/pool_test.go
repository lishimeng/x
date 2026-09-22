package gopool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRuns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := New(ctx, WithMinSize(1), WithMaxSize(2), WithIdleTimeout(time.Minute))
	var n atomic.Int32
	if err := p.Do(context.Background(), func(ctx context.Context) error {
		n.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if n.Load() != 1 {
		t.Fatalf("n=%d", n.Load())
	}
}

func TestExpandToMax(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := New(ctx, WithMinSize(1), WithMaxSize(4), WithIdleTimeout(time.Minute))

	started := make(chan struct{}, 4)
	release := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.Do(context.Background(), func(ctx context.Context) error {
				started <- struct{}{}
				<-release
				return nil
			})
		}()
	}
	for i := 0; i < 4; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("workers did not start")
		}
	}
	if got := p.Size(); got != 4 {
		t.Fatalf("size=%d want 4", got)
	}
	close(release)
	wg.Wait()
}

func TestShrinkIdle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := New(ctx, WithMinSize(1), WithMaxSize(3), WithIdleTimeout(40*time.Millisecond))

	var wg sync.WaitGroup
	block := make(chan struct{})
	started := make(chan struct{}, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = p.Do(context.Background(), func(ctx context.Context) error {
				started <- struct{}{}
				<-block
				return nil
			})
		}()
	}
	for i := 0; i < 3; i++ {
		<-started
	}
	if p.Size() != 3 {
		t.Fatalf("size=%d", p.Size())
	}
	close(block)
	wg.Wait()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if p.Size() <= 1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("did not shrink, size=%d", p.Size())
}

func TestDoContextCancelWhileQueued(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := New(ctx, WithMinSize(1), WithMaxSize(1), WithIdleTimeout(time.Minute))

	hold := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_ = p.Do(context.Background(), func(ctx context.Context) error {
			close(started)
			<-hold
			return nil
		})
	}()
	<-started

	jobCtx, jobCancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer jobCancel()
	err := p.Do(jobCtx, func(ctx context.Context) error { return nil })
	if err == nil {
		t.Fatal("expected error")
	}
	close(hold)
}

func TestNewNilContextPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = New(nil)
}

func TestDoNilContextPanics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := New(ctx, WithMinSize(1), WithMaxSize(1))
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = p.Do(nil, func(context.Context) error { return nil })
}

func TestDoNilFnPanics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := New(ctx, WithMinSize(1), WithMaxSize(1))
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = p.Do(context.Background(), nil)
}

func TestPoolContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := New(ctx, WithMinSize(2), WithMaxSize(2), WithIdleTimeout(time.Minute))
	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := p.Do(context.Background(), func(context.Context) error { return nil }); err == ErrPoolClosed {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := p.Do(context.Background(), func(context.Context) error { return nil }); err != ErrPoolClosed {
		t.Fatalf("Do after cancel: %v", err)
	}
	for time.Now().Before(deadline) {
		if p.Size() == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("workers did not exit after cancel, size=%d", p.Size())
}
