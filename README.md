# goroutines

goroutines is a simple goroutine pool which aims to reuse goroutines and limit the number of goroutines.

> goroutines 是一个简单的 goroutine 池，旨在复用 goroutine，并**限制 goroutine 的数量**。

## example

```Go
G := NewPool(8)

G.Go(ctx, func(){})
```