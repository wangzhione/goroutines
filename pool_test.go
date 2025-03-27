package goroutines

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func DoCopyStack(a, b int) int {
	if b < 100 {
		return DoCopyStack(0, b+1)
	}
	return 0
}

func testFunc() {
	_ = DoCopyStack(0, 0)
}

var ctx = context.Background()

const bechmarkCount = 10000000

func TestPool(t *testing.T) {
	p := NewPool(8)
	var n int32

	var wg sync.WaitGroup
	wg.Add(bechmarkCount)
	for range bechmarkCount {
		p.Go(ctx, func(context.Context) {
			defer wg.Done()
			atomic.AddInt32(&n, 1)
		})
	}
	wg.Wait()

	if n != bechmarkCount {
		t.Error(n)
	}
}

// TestPool 相对 TestGo 普通基准测试, 性能损失 1 倍, 但随着复杂业务, 二者差距没有想象那么大

func TestGo(t *testing.T) {
	var n int32

	var wg sync.WaitGroup
	wg.Add(bechmarkCount)
	for range bechmarkCount {
		go func() {
			defer wg.Done()
			atomic.AddInt32(&n, 1)
		}()
	}
	wg.Wait()

	if n != bechmarkCount {
		t.Error(n)
	}
}

func TestPoolPanic(t *testing.T) {
	testPanicFunc := func(context.Context) {
		panic("test")
	}

	p := NewPool(128)
	p.Go(ctx, testPanicFunc)

	slog.InfoContext(ctx, "Success")
}

const benchmarkTimes = 10000

func BenchmarkPool(b *testing.B) {
	p := NewPool(int32(runtime.GOMAXPROCS(0)))

	var wg sync.WaitGroup
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		wg.Add(benchmarkTimes)
		for range benchmarkTimes {
			p.Go(ctx, func(context.Context) {
				testFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}

// BenchmarkPool1.895s 性能比 BenchmarkGo 1.473s

/*
goos: windows
goarch: amd64
pkg: github.com/wangzhione/sbp/helper/safego/tasks
cpu: AMD Ryzen 9 7945HX3D with Radeon Graphics

BenchmarkPool-32
     206	   6692212 ns/op	 1255430 B/op	   48432 allocs/op
PASS
ok  	github.com/wangzhione/sbp/helper/safego/tasks	2.140s

BenchmarkGo-32
     100	  12601970 ns/op	  165947 B/op	   10012 allocs/op
PASS
ok  	github.com/wangzhione/sbp/helper/safego/tasks	1.473s

*/

func BenchmarkGo(b *testing.B) {
	var wg sync.WaitGroup
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		wg.Add(benchmarkTimes)
		for range benchmarkTimes {
			go func() {
				testFunc()
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func TestGoroutines(t *testing.T) {
	// 3 个函数分别打印 cat、dog、fish，要求每个函数都要起一个 goroutine，按照 cat、dog、fish 顺序打印在屏幕上 10 次。

	const count = 10

	catCh := make(chan struct{}, 1)
	dogCh := make(chan struct{}, 1)
	fishCh := make(chan struct{}, 1)

	var wait sync.WaitGroup
	wait.Add(3)

	fCat := func(context.Context) {
		n := 0
		for {
			n++
			t.Logf("%3d cat", n)

			dogCh <- struct{}{}

			<-catCh

			if n >= count {
				wait.Done()
				break
			}
		}
	}

	fDog := func(context.Context) {
		n := 0
		for {
			<-dogCh

			n++
			t.Logf("%3d dog", n)

			fishCh <- struct{}{}

			if n >= count {
				wait.Done()
				break
			}
		}
	}

	fFish := func(context.Context) {
		n := 0
		for {
			<-fishCh

			n++
			t.Logf("%3d fish", n)

			catCh <- struct{}{}

			if n >= count {
				wait.Done()
				break
			}
		}
	}

	p := NewPool(3)
	p.Go(ctx, fFish)
	p.Go(ctx, fDog)
	p.Go(ctx, fCat)

	wait.Wait()
}

func TestSucccessCompareInc(t *testing.T) {
	var capacity int32 = 2
	var worker int32

	for range 20000 {
		go func() {
			old := atomic.LoadInt32(&worker)
			if old < capacity {
				if atomic.CompareAndSwapInt32(&worker, old, old+1) {
					if worker > capacity {
						t.Logf("worker=%d, capacity=%d", worker, capacity)
					}

					go func() {
						defer atomic.AddInt32(&worker, -1)
					}()
				}
			}
		}()
	}
}

func TestErrorCompareInc(t *testing.T) {
	var capacity int32 = 2
	var worker int32

	for range 400 {
		go func() {
			if atomic.LoadInt32(&worker) < capacity {
				atomic.AddInt32(&worker, 1)

				if worker > capacity {
					t.Logf("worker=%d, capacity=%d", worker, capacity)
				}

				go func() {
					defer atomic.AddInt32(&worker, -1)
				}()
			}
		}()
	}
}
