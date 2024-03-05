# CLUES BEST PRACTICES

A guide for getting the most out of Clues.

## CTX

The clues package can leverage golang's `context.Context` to pack in
metadata that you can later retrieve for logging or observation.  These
additions form a tree, so you're always safe to extend or replace existing
state.

### Tracing

Calling `AddTrace` or `AddTraceName` will append to an internal property
with the key `clues_trace`.  Clues traces form a slice of comma delimited
hashes (or names, when using TraceName).  These can be useful for filtering
logs to certain process branches.

clues.Add() always automatically appends to the trace, so you don't need
to go out of your way to get the benefits unless you specifically want to
add or name traces.

```go
func main() {
    ctx := clues.Add(context.Background(), "k", "v")
    // clues_trace: abc123
    foo(ctx)
}

func foo(ctx context.Context) {
    ctx = clues.AddTrace(ctx)
    // clues_trace: abc123,xyz987
    bar(ctx)
}

func bar(ctx context.Context) {
    ctx = clues.AddTraceName(ctx, "bar", "k2", "v2")
    // clues_trace: abc123,xyz987,bar
    // ...
}
```

### Iterators

Create a separate ctx variable when recording clues within a loop.

```go
ctx = clues.Add(ctx, "k", "v")

for _, user := range users {
    ictx := clues.Add(ctx, "user_id", user.ID)
    handleUser(ictx, user)
}
```

## ERRORS

Errors are the bottom-to-top metadata tracing counterpart to contexts.
At minimum, they replicate the creation, wrapping, and stacking of
errors.  You can also label clues errors for broad categorization and
add key:value metadata sets, including the full set of clues embedded
in a ctx value.


### Always Stack

A single-error Stack ensures clues will append a stacktrace reference
for that point of return.  New(), Wrap() and Stack()ing multiple errors
all do this same process.  This tip is specific for cases where you'd
normally `return err`.

```go
err := downstreamCall(ctx)
if err != nil {
    return clues.Stack(err)
}
```

### With Ctx As Needed

Clues doesn't automatically push the clues from a ctx into an error.
When you want to do that, you can either append to a clues error using
WithClues(ctx), or call a constructor that includes the ctx.

Although it's relatively benign to add the ctx at many different layers
throughout the error return, it isn't idiomatic, and you'll get the best
results if you do it only once: at the beginning of the error chain.

```go
req, err := prepRequest()
if err != nil {
    // the called function didn't accept a ctx param,
    // therefore the wrap here should attach the ctx.
    return clues.WrapWC(ctx)
}

if !req.Ready() {
    // similar case here; since we're creating a new error
    // it's best to attach the values embedded in the ctx.
    return clues.NewWC(ctx)
}

err = req.Do(ctx)
if errors.Is(err, ErrBadRequest)
    // lets assume req.Do calls external code; even though
    // we passed it a ctx, the called func won't have added
    // the ctx to the error.  We still need to do that here.
    return clues.StackWC(ctx, ErrFailedReq, err)
}

err = cleanup(ctx, req)
if err != nil {
    // finally, we have an internal func that accepts a ctx
    // and returns an error.  It's a best practice to assume
    // that the downstream code already added the ctx you
    // provided it, which means the error can be returned
    // as-is.
    return clues.Stack(err)
}
```

### Log the Core

The clues within errors aren't going to show up in a log message
or test output naturally.  The formatted value will only include
the message.  You can easily extract the full details using ToCore().

```go
// in logs:
logger.With("err", clues.ToCore(err)).Error("trying to foo")
// or in tests
assert.NoError(t, err, clues.ToCore(err))
```

### Labels Are Not Sentinels

Clues errors already support Is and As, which means you can use them
for errors.Is and errors.As calls.  This means you shouldn't use
labels for the same purpose.

```go
// don't do this
clues.Stack(err).Label(resp.Status)
// when you can do this
clues.Stack(errByRespCode(resp.StatusCode), err)
```

The best usage of labels is when you want to add _identifiable_ metadata
to an error.  This is great for conditions where multiple different
errors can be flagged to get some specific handling.  This allows you
to identify the condition without altering the error itself.

```go
// example 1: 
// doesn't matter what error took place, we want to end in a
// certain process state as a categorical result
for err := range processCh {
    if clues.HasLabel(err, mustFailBackup) {
       // set the backup state to 'failed' 
    }
}

// example 2:
// we can categorically ignore errors based on configuration. 
for _, err := range processFailures {
    for _, cat := range config.ignoreErrorCategories {
        if clues.HasLabel(err, cat) {
           processFailures.IgnoreError(err) 
        }
    }
}
```
