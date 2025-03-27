package goroutines

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// NewPool creates a new pool with the given name, cap and config.
func NewPool(capacity int32) *pool {
	return &pool{Capacity: capacity}
}

type pool struct {
	sync.Mutex

	// linked list of tasks (tail push head pop) ; 默认处理的是并发处理任务不多的情况
	head *task
	tail *task

	// Capacity of the pool, the maximum number of goroutines that are actually working
	Capacity int32
	// Record the number of running workers
	worker int32
}

// task list
type task struct {
	// context is trade 执行链的上下文, 主要用于 panic handler 追查链
	c    context.Context
	fn   func(ctx context.Context)
	next *task
}

func (p *pool) Push(tail *task) {
	// tail push
	p.Lock()
	if p.head != nil {
		p.tail.next = tail
	} else {
		p.head = tail
	}
	p.tail = tail
	p.Unlock()
}

func (p *pool) Pop() (head *task) {
	p.Lock()
	if p.head != nil {
		head = p.head
		p.head = head.next
	}
	p.Unlock()
	return
}

// This method will be called when the worker panic.
var PanicHandler = func(ctx context.Context, cover any) {
	// 业务统一格式的日志 or PanicHandler 注册业务的全局自定义 panic 告警
	slog.ErrorContext(ctx,
		"tasks worker run panic error",
		slog.Any("error", cover),
		slog.String("stack", string(debug.Stack())),
	)
}

// Go pool add task before run
// ctx 多数需要 chain.CopyTrace(ctx, keys), // context 脱敏 & 延长生命周期
func (p *pool) Go(ctx context.Context, fn func(c context.Context)) {
	tail := &task{
		c:  ctx, // 需要自行进行 context 脱敏 & 延长生命周期
		fn: fn,
	}

	// tail push
	p.Push(tail)

	// The current number of workers is less than the upper limit p.cap.
	// not atomic.LoadInt32(&p.capacity) 设计原因是 惰性的动态修改 worker limit, 但大多数情况用不上
	worker := atomic.LoadInt32(&p.worker)
	if worker < p.Capacity && atomic.CompareAndSwapInt32(&p.worker, worker, worker+1) {
		go p.running()
	}
}

func (p *pool) running() {
	for p.head != nil {
		// pop head after run task
		head := p.Pop()
		if head == nil {
			break
		}

		func() {
			defer func() {
				if cover := recover(); cover != nil {
					PanicHandler(head.c, cover)
				}
			}()

			head.fn(head.c)
		}()
	}

	atomic.AddInt32(&p.worker, -1)
}

// p.Capacity p.Worker() p.Len() 属于运行时内部监控

func (p *pool) Worker() int32 {
	return atomic.LoadInt32(&p.worker)
}

func (p *pool) Len() int {
	p.Lock()
	defer p.Unlock()

	n := 0
	for iter := p.head; iter != nil; iter = iter.next {
		n++
	}
	return n
}
