更好的选型和补充请异步到 see 👁 [safego](https://github.com/wangzhione/sbp/tree/master/helper/safego) 特别是其中 [chango](https://github.com/wangzhione/sbp/blob/master/helper/safego/chango/chango.go#L17-L23) chan go 模型实战更加有效率

# goroutines

goroutines is a simple goroutine pool which aims to reuse goroutines and limit the number of goroutines.

> goroutines 是一个简单的 goroutine 池，旨在复用 goroutine，并**限制 goroutine 的数量**。

## example

**[optional] Step 0 : main.init add goroutines.InitPanicHandler** 

```Go
// first register global panic handler
goroutines.PanicHandler = func (ctx context.Context, cover any) {
    // ctx is goroutines.Go func context, cover = recover()
}
```

**Step 1 : Let's Go**

```Go
o := goroutines.NewPool(8)

// ctx 参照 chain.CopyTrace https://github.com/wangzhione/sbp/blob/master/chain/trace.go#L30-L44
o.Go(ctx, func(c context.Context) {
    // Your business
})
```
