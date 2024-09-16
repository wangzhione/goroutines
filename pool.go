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
	ctx context.Context
	f   func()

	next *task
}

type pool struct {
	// linked list of tasks (tail push head pop)
	sync.Mutex
	head *task
	tail *task

	// capacity of the pool, the maximum number of goroutines that are actually working
	capacity int32
	// Record the number of running workers
	worker int32

	// This method will be called when the worker panic. rec = recover()
	panicHandler func(ctx context.Context, rec any)
}

// NewPool creates a new pool with the given name, cap and config.
func NewPool(capacity int32) *pool {
	return &pool{capacity: capacity}
}

// SetCapacity 可以运行时建议 worker goroutine 数量上限, 惰性采纳
func (p *pool) SetCapacity(capacity int32) {
	atomic.StoreInt32(&p.capacity, capacity)
}

// SetPanicHandler the func here will be called after the panic has been recovered.
func (p *pool) SetPanicHandler(handler func(context.Context, any)) {
	// handler context by into task::context
	p.panicHandler = handler
}

func (p *pool) Go(ctx context.Context, f func()) {
	task := &task{ctx: ctx, f: f}

	// tail push
	p.Lock()
	if p.head == nil {
		p.head = task
	} else {
		p.tail.next = task
	}
	p.tail = task
	p.Unlock()

	// The current number of workers is less than the upper limit p.cap.
	if atomic.LoadInt32(&p.worker) < atomic.LoadInt32(&p.capacity) {
		atomic.AddInt32(&p.worker, 1)

		go p.run()
	}
}

func (p *pool) run() {
	defer atomic.AddInt32(&p.worker, -1)

	for {

		// pop head after run task
		var now *task

		p.Lock()
		if p.head == nil {
			// if there's no task to do, exit
			p.Unlock()
			return
		}

		now = p.head
		p.head = now.next
		p.Unlock()

		func() {
			defer func() {
				rec := recover()
				if rec != nil {
					if p.panicHandler == nil {
						// replace 业务统一格式的日志
						log.Printf("Goroutines worker run panic : %v : %s\n", rec, debug.Stack())
					} else {
						p.panicHandler(now.ctx, rec)
					}
				}
			}()

			now.f()
		}()
	}
}
