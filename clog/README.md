# Clog

/ˈklȯg/

_verb_

1. To become filled with extraneous matter.
2. To unite in a mass.

---

## How to Clog your code

Clog is an opinionated logging wrapper backed by Zap and OTEL
under the hood. It provides an api for easy interaction with
Clues ctx and Cluerr errors, among a host of other supporting features.

Clog is built around three ideals:

1. Log messages should be small and easy to read.
2. Structured data in logs should provide expansive and exhaustive
   context for debugging.
3. Interoperable with other Clues packages without preventing
   interoperability with non-Clues packages.

Adding logs is simple:

```go
func foo(ctx context.Context) {
  ctx := clues.Add(ctx, "important_id", someID)
  bar(ctx, someID, 1)
}

func bar(ctx context.Context, someID string, count int) {
  // attributes in the ctx are automatically added to the log
  clog.Ctx(ctx).
    // or you can extend the log with local attributes
    With("count", count).
    Info("baring")
}
```

## Other Features

### Labels

Labeling has two benefits.

First, it provides a simple and concrete way to plan out logging
categorization. Why categorize logs? Because the first step in debugging
is often to reach the right set of logs. The faster and more reliably
you can do that, the faster you can debug.

```go
clog.CtxErr(ctx, err).
  Label(myconsts.UserReq, myconsts.Failures).
  Error("creating user")

// in your log viewer
*-logs*
| where attrs.labels contains "user_req"
```

Second, labels provide fine grained control over debug logging. No more
"inclusion by debug-level" or similar nonsense. Just say what debug logs
you want to include, and the rest get filtered out.

```go
set := clog.Settings{
  Format: clog.FormatForHumans,
  Level: clog.LevelDebug,
  OnlyLogDebugIfContainsLabel: []string{clog.APIReq},
}
ctx := clog.Init(ctx, set)

clog.Ctx(ctx).
  Label(clog.APIReq).
  Debug("this log will be included in debug output")

clog.Ctx(ctx).
  Label(clog.DBReq).
  Debug("this log will not")
```

## Comments

```go
clog.Ctx(ctx).
  Comment(`I could just add this in code... but now we can pull double duty!
  whatever I say here is readable to anyone who is looking at the logs (which is good
  if I'm trying to tell them what they need to know about due to this log occurring);
  it's also a regular comment, as if in code, so the code is now also commented!`).
  Info("information")
```

## Automatic OTEL Integration

If you've initialized OTEL through Clues, Clog will automatically
push all logs to the OTEL logger. No extra work required!
