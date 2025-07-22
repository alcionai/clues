# Cluerr

/ˈklü-ər/

_noun_

1. The difference between an observed or calculated value and a true value.

---

## Structure your errors

Ctx allows runtime annotations to trickle downward. Errors
allow them to flow back upstream. By attaching clues attributes
to your errors, you can centralize and minimize your logging
surface area without losing any low-level details.

```go
func main() {
  ctx := clues.Add(
    context.Background()
    "env.name", os.getenv("NAME"),
  )

  err := foo(ctx)
  if err != nil {
    // Only top-level logging is needed.
    clog.CtxErr(ctx, err).Error("calling foo")
  }
}

func foo(ctx context.Context) error {
  ctx = clues.AddSpan(ctx, "foo", "bar")
  err := bar()
  if err != nil {
    return clues.WrapWC(ctx, err, "calling bar")
  }
  return nil
}
```

## Add local attributes

Attributes can be added to any error, even without a context.

```go
func foo() error {
  err := bar()
  if err != nil {
    return cluerr.Wrap(err, "calling bar").
      With("local_attr", "value")
  }
  return nil
}
```

## Skip callers in helper funcs

Helper funcs can back out their stack trace to report
the calling line.

```go
func enrichErr(err error) error {
  return clues.Stack(err).
    With("enriched", "value").
    SkipCaller(1)
}

func foo(ctx context.Context) error {
  err := bar(ctx)
  if err != nil {
    return enrichErr(err)
  }
  return nil
}
```

## Label errors

Labeling errors not only provides additional insights,
it lets you handle errors in different ways without changing
the error's identity.

```go
func getUsers(ctx context.Context, users []string) error {
  for _, user := range users {
    ictx = clues.Add(ctx, "user", user)
    err := processUser(ictx, user)
    if err != nil {
      if clues.HasLabel(err, consts.Unrecoverable) {
        // Handle user processing errors differently
        return cluerr.Wrap(ictx, err, "unrecoverable crash processing user")
      }

      clog.CtxErr(ictx, err).Error("processing user")
    }
  }
  return nil
}

func processUser(ctx context.Context, user string) error {
  err := externalPkg.DoSomething(ctx, user)
  if errors.Is(err, externalPkg.ErrSuperBadJustStopTrying) {
    return clues.Stack(err).Label(consts.Unrecoverable)
  } else {
    return clues.Wrap(err, "processing user").
  }
}
```

## Print your error details in testing

ToCore() serializes the error details for easy printing.

```go
func TestFoo(t *testing.T) {
  err := foo(ctx)
  assert.NoError(t, err, cluerr.ToCore(err))
}
```
