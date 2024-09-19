package goroutines

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

const benchmarkTimes = 10000

func DoCopyStack(a, b int) int {
	if b < 100 {
		return DoCopyStack(0, b+1)
	}
	return 0
}

func testFunc() {
	DoCopyStack(0, 0)
}

func testPanicFunc() {
	panic("test")
}

const bechmarkCount = 10000000

func TestPool(t *testing.T) {
	p := NewPool(8)
	var n int32

	var wg sync.WaitGroup
	for i := 0; i < bechmarkCount; i++ {
		wg.Add(1)
		p.Go(context.Background(), func() {
			defer wg.Done()
			atomic.AddInt32(&n, 1)
		})
	}
	wg.Wait()

	if n != bechmarkCount {
		t.Error(n)
	}
}

func TestGo(t *testing.T) {
	var n int32

	var wg sync.WaitGroup
	for i := 0; i < bechmarkCount; i++ {
		wg.Add(1)
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
	p := NewPool(128)
	p.Go(context.Background(), testPanicFunc)
}

func BenchmarkPool(b *testing.B) {
	p := NewPool(int32(runtime.GOMAXPROCS(0)))

	var wg sync.WaitGroup
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchmarkTimes)
		for j := 0; j < benchmarkTimes; j++ {
			p.Go(context.Background(), func() {
				testFunc()
				wg.Done()
			})
		}
		wg.Wait()
	}
}

func BenchmarkGo(b *testing.B) {
	var wg sync.WaitGroup
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(benchmarkTimes)
		for j := 0; j < benchmarkTimes; j++ {
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

	var catCh = make(chan struct{}, 1)
	var dogCh = make(chan struct{}, 1)
	var fishCh = make(chan struct{}, 1)

	var wait sync.WaitGroup
	wait.Add(3)

	fCat := func() {
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

	fDog := func() {
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

	fFish := func() {
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

	ctx := context.Background()

	p := NewPool(3)
	p.Go(ctx, fFish)
	p.Go(ctx, fDog)
	p.Go(ctx, fCat)

	wait.Wait()
}

func TestCompareInc(t *testing.T) {
	var capacity int32 = 2
	var worker int32

	for range 2000 {
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

func TestCompareInc2(t *testing.T) {
	var capacity int32 = 2
	var worker int32

	for range 2000 {
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
