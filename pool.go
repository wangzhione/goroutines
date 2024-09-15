package goroutines

import (
	"context"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
)

type task struct {
	ctx context.Context
	f   func()

	next *task
}

var taskZero task

var taskPool = sync.Pool{New: func() any { return new(task) }}

func taskPoolRecycle(t *task) {
	*t = taskZero
	taskPool.Put(t)
}

func taskPoolGet(ctx context.Context, f func()) *task {
	t := taskPool.Get().(*task)
	t.ctx = ctx
	t.f = f
	return t
}

type pool struct {
	// linked list of tasks
	sync.Mutex
	head  *task
	tail  *task
	count int32

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

// Worker 可以用于线上统计 worker goroutine 数量
func (p *pool) Worker() int32 {
	return atomic.LoadInt32(&p.worker)
}

// SetPanicHandler the func here will be called after the panic has been recovered.
func (p *pool) SetPanicHandler(handler func(context.Context, any)) {
	p.panicHandler = handler
}

func (p *pool) Go(ctx context.Context, f func()) {
	t := taskPoolGet(ctx, f)

	p.Lock()
	if p.head == nil {
		p.head = t
		p.tail = t
	} else {
		p.tail.next = t
		p.tail = t
	}
	p.count++
	p.Unlock()

	// The following two conditions are met:
	// 1. the number of tasks is greater than the threshold.
	// 2. The current number of workers is less than the upper limit p.cap.
	// or there are currently no workers.
	if atomic.LoadInt32(&p.count) > 0 {
		worker := p.Worker()
		if worker == 0 || worker < atomic.LoadInt32(&p.capacity) {
			go p.run()
		}
	}
}

func (p *pool) run() {
	atomic.AddInt32(&p.worker, 1)

	for {

		// pop head after run task
		var now *task

		p.Lock()
		if p.head == nil {
			// if there's no task to do, exit
			p.Unlock()
			atomic.AddInt32(&p.worker, -1)
			return
		}

		now = p.head
		p.head = now.next
		p.count--
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

		taskPoolRecycle(now)
	}
}
