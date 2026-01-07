# CLUES

/kluːz/

_noun_

1. Something that guides through an intricate procedure or maze of difficulties.
   Specifically: a piece of evidence that leads one toward the solution of a problem.

_verb_

1. To give reliable information to.

---

[![PkgGoDev](https://pkg.go.dev/badge/github.com/alcionai/clues)](https://pkg.go.dev/github.com/alcionai/clues) [![goreportcard](https://goreportcard.com/badge/github.com/alcionai/clues)](https://goreportcard.com/report/github.com/alcionai/clues)

Clues is a golang telemetry aid designed to simplify debugging down to an O(1) operation.
What is O(1) debugging? That's when a single event provides all the runtime
context you need to grok what was happening in your code at that moment.

Clues works in tandem with the Cluerr and Clog subpackages to produce a simple
api that achieves this goal. By populating a context-bound cache of runtime
variables, Cluerr can bind a cache snapshot within an error, and Clog
extracts those variables for logging.

Need more support? Clues comes with OTEL configuration out of the box. Attributes
added to the context are automatically added to the span, while Clog implicitly
handles OTEL logging alongside the default stdout logger. Additional support
packages like Ctats and Clutel help minimize the effort of engaging with those
systems and provide a consistent interface for your telemetry production.

---

## How To Get A Clue

```go
func foo(ctx context.Context, someID string) error {
    // Annotate the ctx with your important runtime attributes.
    ctx = clues.Add(ctx, "important_id", someID)
    return bar(ctx, someID)
}
```

```go
func bar(ctx context.Context, someID string) error {
    err := externalPkg.DoThing(ctx, someID)
    if err != nil {
        // Wrap the error with a snapshot of the current attributes
        return clues.WrapWC(ctx, err, "doing something")
    }
    return nil
}
```

```go
func main() {
    err := foo(context.Background(), "importantID")
    if err != nil {
        // Centralize your logging at the top
        // without losing any low-level details.
        clog.
            CtxErr(ctx, err).
            Error("calling foo")
    }
}
```

---

## Quickstart

The most basic setup only needs to flush the logger on exit.

```go
package main

import (
    "context"

    "github.com/alcionai/clues/clog"
)
func main() {
  ctx := clog.Init(context.Background(), clog.Settings{
   Format: clog.FormatForHumans,
  })
  defer clog.Flush()
  // And away you go!
}
```

OTEL support requires its own initialization and flush.

```go
package main
import (
    "context"

    "github.com/alcionai/clues"
    "github.com/alcionai/clues/clog"
    "github.com/alcionai/clues/clutel"
)

func main() {
  ctx, err := clues.InitializeOTEL(
    context.Background(),
    myServiceName,
    clutel.OTELConfig{
      GRPCEndpoint: os.GetEnv(myconsts.OTELGRPCEndpoint),
    },
  )
  if err != nil {
    panic(err)
  }

  ctx = clog.Init(ctx, clog.Settings{
    Format: clog.FormatForHumans,
  })

  defer func() {
    clog.Flush(ctx)

    err := clues.Close(ctx)
    if err != nil {
      // use the standard log here since clog was already flushed
      log.Printf("closing clues: %v\n", err)
    }
  }
}
```

---

## Resources

- [Best Practices](https://github.com/alcionai/clues/blob/main/BEST_PRACTICES.md)
- [Cluerr](https://github.com/alcionai/clues/blob/main/cluerr/README.md) - error api.
- [Clog](https://github.com/alcionai/clues/blob/main/clog/README.md) - logging api.
- [Ctats](https://github.com/alcionai/clues/blob/main/ctats/README.md) - OTEL metrics wraper.
- [Cecrets](https://github.com/alcionai/clues/blob/main/cecrets/README.md) - pii-in-telemetry obfuscation.
- [Clutel]((https://github.com/alcionai/clues/blob/main/clutel/README.md)) - OTEL trace/span wrapper.

---

## Why Not {my favorite logging package}?

Many logging packages let you build attributes within the context, handing
in down the layers of your runtime stack. But that's often as far as they go.
In order to utilize those built-up attributes, you have to push all of your
logging to the leaves of your process tree. This adds considerable boilerplate,
not to mention cognitive burden, to your telemetry surface area.

Providing an interface that hooks into both downward (ctx) and upward (error)
data transmission, clues helps you minimize the amount of logging you do to
only those necessary occurrences.

As for your favorite logger: look forward for more robust logger support in
clog in the future.

## Why Not Just OTEL?

OTEL is awesome. We love OTEL 'round here. We do not, however, love the effort
it takes to set it up. And we think the apis are awful clunky.

If your codebase is already OTEL-enabled and decked out, then there isn't much
clues can offer except for nicer, cleaner code and happier devs.

But if you're on a new project, prototyping a new feature, or switching to
OTEL from some other paradigm, then Clues can get your telemetry bootstrapped
and running in, well, less time that it took you to read this far.
