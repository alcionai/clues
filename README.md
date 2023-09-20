# CLUES

[![PkgGoDev](https://pkg.go.dev/badge/github.com/alcionai/clues)](https://pkg.go.dev/github.com/alcionai/clues) [![goreportcard](https://goreportcard.com/badge/github.com/alcionai/clues)](https://goreportcard.com/report/github.com/alcionai/clues)

A golang library for tracking runtime variables via ctx, passing them upstream within errors, and retrieving context- and error-bound variables for logging.

## Aggregate runtime state in ctx

Track runtime variables by adding them to the context.
```go
func foo(ctx context.Context, someID string) error {
    ctx = clues.Add(ctx, "importantID", someID)
    return bar(ctx, someID)
}
```

Keep error messages readable and augment your telemetry by packing errors with structured data.
```go
func bar(ctx context.Context, someID string) error {
    ctx = clues.Add(ctx, "importantID", someID)
    err := errors.New("a bad happened")
    if err != nil {
        return clues.Stack(err).WithClues(ctx)
    }
    return nil
}
```

Retrive structured data from your errors for logging and other telemetry.
```go
func main() {
    err := foo(context.Background(), "importantID")
    if err != nil {
        logger.
            Error("calling foo").
            WithError(err).
            WithAll(clues.InErr(err))
    }
}
```

## Track individual process flows

Each clues addition traces its additions with a tree of IDs, chaining those traces into the "clues_trace" value.  This lets you quickly and easily filter logs to a specific process tree.
```go
func iterateOver(ctx context.Context, users []string) {
    // automatically adds "clues_trace":"id_a"
    ctx = clues.Add(ctx, "status", good)
    for i, user := range users {
        // automatically appends another id to "clues_trace": "id_a,id_n"
        ictx := clues.Add(ctx, "currentUser", user, "iter", i)
        err := doSomething(ictx, user)
        if err != nil {
            ictx = clues.Add(ictx, "status", bad)
        }
    }
}
```

## Interoperable with pkg/errors

Clues errors can be wrapped by pkg/errors without slicing out
any stored data.
```go
func getIt(someID string) error {
    return clues.New("oh no!").With("importantID", someID)
}

func getItWrapper(someID string) error {
    if err := getIt(someID); err != nil {
        return errors.Wrap(err, "getting the thing")
    }

    return nil
}

func main() {
    err := getItWrapper("id")
    if err != nil {
        fmt.Println("error getting", err, "with vals", clues.InErr(err))
    }
}
```

## Stackable errors

Error stacking lets you embed error sentinels without slicing out the current error's data or relying on err.Error() strings.
```go
var ErrorCommonFailure = "a common failure condition"

func do() error {
    if err := dependency.Do(); err != nil {
        return clues.Stack(ErrorCommonFailure, err)
    }
    
    return nil
}

func main() {
    err := do()
    if errors.Is(err, ErrCommonFailure) {
        // true!
    }
}
```

## Labeling Errors

Rather than build an errors.As-compliant local error to annotate downstream errors, labels allow you to categorize errors with expected qualities.

Augment downstream errors with labels
```go
func foo(ctx context.Context, someID string) error {
    err := externalPkg.DoThing(ctx, someID)
    if err != nil {
        return clues.Wrap(err).Label("retryable")
    }
    return nil
}
```

Check your labels upstream.
```go
func main() {
    err := foo(context.Background(), "importantID")
    if err != nil {
        if clues.HasLabel(err, "retryable")) {
            err := foo(context.Background(), "importantID")
        }
    }
}
```

## Design

Clues is not the first of its kind: ctx-err-combo packages already exist.  Most other packages tend to couple the two notions, packing both into a single handler.  This is, in my opinion, an anti-pattern.  Errors are not context, and context are not errors.  Unifying the two can couple layers together, and your maintenance woes from handling that coupling are not worth the tradeoff in syntactical sugar.

In turn, Clues maintains a clear separation between accumulating data into a context and passing data back in an error.  Both handlers operate independent of the other, so you can choose to only use the ctx (accumulate data into the context, but maybe log it instead of returning data in the err) or the err (only pack immedaite details into the error).

### References
* [https://github.com/mvndaai/ctxerr](https://github.com/mvndaai/ctxerr)
* [https://github.com/suzuki-shunsuke/go-errctx](https://github.com/suzuki-shunsuke/go-errctx)

## Similar Art

Fault is most similar in design to this package, and also attempts to maintain separation between errors and contexts.  The differences are largely syntactical: Fault prefers a composable interface with decorator packages.  I like to keep error production as terse as possible, thus preferring a more populated interface of methods over the decorator design.

### References
* [https://github.com/Southclaws/fault](https://github.com/Southclaws/fault)