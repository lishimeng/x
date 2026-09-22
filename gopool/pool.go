// Package gopool 提供可动态扩缩的有界工人池。
//
// Do 同步等待任务结束（便于调用方在结束后做 ACK 等收尾）。
// 工人数在 [MinSize, MaxSize] 之间：忙时扩容，空闲超时缩回 MinSize。
// 停池：取消 New 传入的 ctx。
package gopool

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrPoolClosed = errors.New("gopool: closed")
)

const defaultIdleTimeout = 30 * time.Second

// Pool 动态工人池。
type Pool struct {
	minSize     int
	maxSize     int
	idleTimeout time.Duration
	ctx         context.Context

	mu      sync.Mutex
	workers int
	idle    int
	jobCh   chan *job
}

type job struct {
	ctx context.Context
	fn  func(context.Context) error
	res chan error
}

// Option 配置 Pool。
type Option func(*Pool)

// WithMinSize 设置最小工人数（至少 1）。
func WithMinSize(n int) Option {
	return func(p *Pool) { p.minSize = n }
}

// WithMaxSize 设置最大工人数（至少等于 MinSize）。
func WithMaxSize(n int) Option {
	return func(p *Pool) { p.maxSize = n }
}

// WithIdleTimeout 空闲工人超过此时长且人数 > MinSize 时可回收。
func WithIdleTimeout(d time.Duration) Option {
	return func(p *Pool) { p.idleTimeout = d }
}

// New 创建池；生命周期跟随 ctx，取消即停池。ctx 必填（不可为 nil）。默认 min=2 max=8 idleTimeout=30s。
func New(ctx context.Context, opts ...Option) *Pool {
	if ctx == nil {
		panic("gopool: nil context")
	}
	p := &Pool{
		minSize:     2,
		maxSize:     8,
		idleTimeout: defaultIdleTimeout,
		ctx:         ctx,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	if p.minSize < 1 {
		p.minSize = 1
	}
	if p.maxSize < p.minSize {
		p.maxSize = p.minSize
	}
	if p.idleTimeout <= 0 {
		p.idleTimeout = defaultIdleTimeout
	}
	p.jobCh = make(chan *job)
	for i := 0; i < p.minSize; i++ {
		p.startWorkerLocked()
	}
	return p
}

// MinSize 最小工人数。
func (p *Pool) MinSize() int { return p.minSize }

// MaxSize 最大工人数。
func (p *Pool) MaxSize() int { return p.maxSize }

// Size 当前工人数。
func (p *Pool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workers
}

func (p *Pool) stopped() bool {
	return p.ctx.Err() != nil
}

func (p *Pool) startWorkerLocked() {
	p.workers++
	go p.worker()
}

func (p *Pool) worker() {
	idleTimer := time.NewTimer(p.idleTimeout)
	defer idleTimer.Stop()

	for {
		if p.stopped() {
			p.mu.Lock()
			p.workers--
			p.mu.Unlock()
			return
		}

		p.mu.Lock()
		p.idle++
		p.mu.Unlock()

		if !idleTimer.Stop() {
			select {
			case <-idleTimer.C:
			default:
			}
		}
		idleTimer.Reset(p.idleTimeout)

		select {
		case <-p.ctx.Done():
			p.mu.Lock()
			p.idle--
			p.workers--
			p.mu.Unlock()
			return
		case j := <-p.jobCh:
			p.mu.Lock()
			p.idle--
			p.mu.Unlock()
			if j == nil {
				p.mu.Lock()
				p.workers--
				p.mu.Unlock()
				return
			}
			if p.stopped() {
				select {
				case j.res <- ErrPoolClosed:
				default:
				}
				p.mu.Lock()
				p.workers--
				p.mu.Unlock()
				return
			}
			err := j.fn(j.ctx)
			j.res <- err
		case <-idleTimer.C:
			p.mu.Lock()
			p.idle--
			if p.ctx.Err() != nil {
				p.workers--
				p.mu.Unlock()
				return
			}
			if p.workers > p.minSize {
				p.workers--
				p.mu.Unlock()
				return
			}
			p.mu.Unlock()
		}
	}
}

// Do 在工人上执行 fn 并等待结果。无空闲且未达 max 时扩容；已满则阻塞等待工人。
// ctx、fn 必填；为 nil 时 panic（用法错误）。
func (p *Pool) Do(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		panic("gopool: nil context")
	}
	if fn == nil {
		panic("gopool: nil fn")
	}
	if p.stopped() {
		return ErrPoolClosed
	}

	j := &job{ctx: ctx, fn: fn, res: make(chan error, 1)}

	for {
		if p.stopped() {
			return ErrPoolClosed
		}

		p.mu.Lock()
		if p.ctx.Err() != nil {
			p.mu.Unlock()
			return ErrPoolClosed
		}
		if p.idle == 0 && p.workers < p.maxSize {
			p.startWorkerLocked()
		}
		atMaxBusy := p.idle == 0 && p.workers >= p.maxSize
		p.mu.Unlock()

		if atMaxBusy {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-p.ctx.Done():
				return ErrPoolClosed
			case p.jobCh <- j:
				return p.waitResult(ctx, j)
			case <-time.After(5 * time.Millisecond):
				continue
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.ctx.Done():
			return ErrPoolClosed
		case p.jobCh <- j:
			return p.waitResult(ctx, j)
		}
	}
}

func (p *Pool) waitResult(ctx context.Context, j *job) error {
	select {
	case err := <-j.res:
		return err
	case <-p.ctx.Done():
		select {
		case err := <-j.res:
			if err != nil {
				return err
			}
			return ErrPoolClosed
		case <-time.After(time.Second):
			return ErrPoolClosed
		}
	case <-ctx.Done():
		select {
		case err := <-j.res:
			return err
		case <-time.After(time.Second):
			return ctx.Err()
		}
	}
}
