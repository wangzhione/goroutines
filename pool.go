package goroutines

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

// task list
type task struct {
	// context is trade 执行链的上下文, 主要用于 panic handler 追查链
	ctx context.Context
	f   func()

	next *task
}

type pool struct {
	// linked list of tasks (tail push head pop)
	// 默认处理的是并发处理任务不多的情况
	sync.Mutex
	head *task
	tail *task

	// capacity of the pool, the maximum number of goroutines that are actually working
	capacity int32
	// Record the number of running workers
	worker int32
}

// NewPool creates a new pool with the given name, cap and config.
func NewPool(capacity int32) *pool {
	return &pool{capacity: capacity}
}

// This method will be called when the worker panic.
var panicHandler = func(ctx context.Context, cover any) {
	// replace 业务统一格式的日志 or InitPanicHandler 注册业务的全局自定义 panic 告警
	log.Printf("Goroutines_panic_error worker run panic : %v : %v : %s\n", ctx, cover, debug.Stack())
}

// InitPanicHandler the func here will be called after the panic has been recovered.
func InitPanicHandler(handler func(context.Context, any)) {
	panicHandler = handler
}

func (p *pool) Go(ctx context.Context, f func()) {
	ask := &task{ctx: ctx, f: f}

	// tail push
	p.Lock()
	if p.head != nil {
		p.tail.next = ask
	} else {
		p.head = ask
	}
	p.tail = ask
	p.Unlock()

	// The current number of workers is less than the upper limit p.cap.
	// not atomic.LoadInt32(&p.capacity) 设计原因是 不希望提供运行时动态的修改 worker limit 能力, 因为用不上
	if atomic.LoadInt32(&p.worker) < p.capacity {
		atomic.AddInt32(&p.worker, 1)
		go p.running()
	}
}

func (p *pool) running() {
	for p.head != nil {
		// pop head after run task
		var now *task

		p.Lock()
		if p.head == nil {
			// if there's no task to do, exit
			p.Unlock()
			break
		}
		now = p.head
		p.head = now.next
		p.Unlock()

		func() {
			defer func() {
				if cover := recover(); cover != nil {
					panicHandler(now.ctx, cover)
				}
			}()

			now.f()
		}()
	}

	atomic.AddInt32(&p.worker, -1)
}
