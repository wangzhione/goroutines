# goroutines

goroutines is a simple high-performance goroutine pool which aims to reuse goroutines and limit the number of goroutines.

> goroutines 是一个简单的高性能的 goroutine 池，旨在复用 goroutine，并**限制 goroutine 的数量**。

## example

```Go
G := NewPool(8)

G.Go(ctx, func(){})
```