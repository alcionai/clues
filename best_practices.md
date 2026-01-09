# CLUES BEST PRACTICES

A guide for getting the most out of Clues.

## OTEL Spans

OTEL spans can be generated with `clues.AddSpan()`. clues.Add()
automatically adds attributes to the current span.

```go
func main() {
  ctx := clues.Add(context.Background(), "k", "v")
  foo(ctx)
}

func foo(ctx context.Context) {
  ctx = clues.AddSpan(ctx, "foo")
  defer clues.CloseSpan(ctx)
  bar(ctx)
}
```

## Iteration within a func

A local context should always be used when adding attributes
within a loop. This prevents accidendal cross-contamination
with the parent ctx.

```go
func getUsers(ctx context.Context, users []User) {
  ctx = clues.Add(ctx, "total_users", len(users))
  for _, user := range users {
    ictx := clues.Add(ctx, "user_id", user.ID)
    handleUser(ictx, user)
  }
}
```

## Always stack unwrapped errors

A single-error Stack ensures stacktrace references are added for
every return statement.

```go
func foo (ctx context.Context) error {
  err := downstreamCall(ctx)
  if err != nil {
    // don't `return err`, do:
    return clues.Stack(err)
  }
}
```

### Attach Context to Errors

Attach the context to the error at the lowest available point.

```go
func foo(ctx context.Context) error {
  req, err := prepRequest()
  if err != nil {
    // the called function didn't accept a ctx param,
    // therefore the wrap here should attach the ctx.
    return clues.WrapWC(ctx)
  }

  err = req.Do(ctx)
  if errors.Is(err, ErrBadRequest)
    // lets assume req.Do calls external code; even though
    // we passed it a ctx, the called func does not add
    // the ctx to the error.  We still need to do that.
    return clues.StackWC(ctx, ErrFailedReq, err)
  }

  err = cleanup(ctx, req)
  if err != nil {
    // finally, we have an internal func that accepts a ctx
    // and returns an error.  Assume that the downstream code
    // already added the ctx you provided it, which means the
    // error can be returned as-is.
    return clues.Stack(err)
  }
}
```

### Cluerr labels are not sentinels

Clues errors already comply with Is and As, so you can use them
for errors.Is and errors.As calls. This means you shouldn't use
labels for the same purpose.

```go
// don't do this
cluerr.Stack(err).Label(const.BadStatus)
// when you can do this
cluerr.Stack(errBadStatus, err)
```

The best usage of labels is when you want to add _categorization_
to an error. Useful for conditions where multiple different
errors can be flagged to produce specific handling without changing
the identity of the error itself.

```go
// don't do this
cluerr.Stack(errSetOTELSpanStatus, err)
// when you can do this
cluerr.Stack(err).Label(const.SetOTELSpanStatus)
