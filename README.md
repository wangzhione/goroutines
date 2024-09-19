# goroutines

goroutines is a simple goroutine pool which aims to reuse goroutines and limit the number of goroutines.

> goroutines 是一个简单的 goroutine 池，旨在复用 goroutine，并**限制 goroutine 的数量**。

## example

**[optional] Step 0 : main.init add goroutines.InitPanicHandler** 

```Go
// register global panic handler
goroutines.InitPanicHandler(func (ctx context.Context, cover any) {
    // ctx is goroutines.Go func context, cover = recover()
}) 
```

**Step 1 : Let's Go**

```Go
o := goroutines.NewPool(8)

o.Go(ctx, func(){
    // Your business
})
```
