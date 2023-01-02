# CLUES

A golang library for tracking and presenting clues about process runtime state.

Rather than have you standardize around emitting telemetry within every error handler or fmting contextual details into the error message (or worse, both), Clues provides an interface to accumulate stateful data throughout the process tree, and pack that data into an error as an upstream-readable map.

## Usage Examples

```go
// aggregate state during processing
func foo(ctx context.Context, someID string) error {
    ctx = clues.Add(ctx, "importantID", someID)
    return bar(ctx, someID)
}
```

```go
// wrap an error with the recording data
func bar(ctx context.Context, someID string) error {
    err := doThing(ctx, someID)
    if err != nil {
        return clues.WithMap(err, clues.Values(ctx))
    }
    return nil
}
```

```go
// back upstream, retrive values from the error
func main() {
    err := foo(context.Background(), "importantID")
    if err != nil {
        logger.
            Error("calling foo").
            WithError(err).
            WithAll(clues.ErrValues(err))
    }
}
```

## Design

Clues is not the first of its kind: ctx-err-combo packages already exist.  However, other packages tend to couple the two notions, packing both into a single handler.  This is, in my opinion, an anti-pattern.  Errors are not context, and context are not errors.  Unifying the two introduces problematic coupling, maintenance woes from coupling are not worth the syntactical sugar.

In turn, this package maintains a clear separation between accumulating data into a context and passing data back in an error.  Both handlers operate independent of the other, so you can choose to only use the ctx (accumulate data into the context, but maybe log it instead of returning data in the err) or the err (only pack immedaite details into the error).

## Similar Packages
* [https://github.com/mvndaai/ctxerr](ctxerr)
* [https://github.com/suzuki-shunsuke/go-errctx](go-errctx)