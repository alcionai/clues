# The Clues Logger

## Regular logging

Infow/Debugw/Errorw are also supported.  
So are the \*f variations. Because I'm trying to be nice.  
Warn is not included. Because I'm not _that_ nice.

```go
clog.Ctx(ctx).Info("information")
clog.Ctx(ctx).Label(clog.ExampleDebugLabel).Debug("debugging")
clog.Ctx(ctx).Err(err).Error("badness")
```

## Labeling your logs

Labeling is intended to make _categorical_ lookup of logs much easier.

Many times we build unintentional colloquialisms into our log vocabulary
and try to filter on those when looking for info. Ex: logs that say
"recoverable error" are "the important error logs". No, none of that.

If you have a set of logging that you always want to include or exclude, put
a label on it.
"How was this run configured?" -> filter clabel like /clabel_configuration/
"What caused the process to fail?" -> filter clabel like /clabel_error_origin/

```go
clog.CtxErr(ctx, err).
  Label(clog.LStartOfRun, clog.LFailureSource).
  Info("couldn't start up process")
```

## Commenting your logs

```go
clog.Ctx(ctx).
  Comment(`I could just add this in code... but now we can pull double duty!
  whatever I say here is readable to anyone who is looking at the logs (which is good
  if I'm trying to tell them what they need to know about due to this log occurring);
  it's also a regular comment, as if in code, so the code is now also commented!`).
  Info("information")
```

## Automatically adds structured data from clues

```go
ctx := clues.Add(ctx, "foo", "bar")
err := clues.New("a bad happened").With("fnords", "smarf")

clog.CtxErr(ctx, err).
  With("beaux", "regarde").
  Debug("all the info!")

// this produces a log containing:
// {
//  "msg": "all the info!",
//  "foo": "bar",
//  "fnords": "smarf",
//  "beaux": "regarde",
// }
```

## Setting up logs

```go
set := clog.Settings{
  Format: clog.FormatForHumans,
  Level: clog.LevelInfo,
}

ctx := clog.Init(ctx, set)
```

## Filtering Debug Logs (aka, improved debug levels)

You're using labels to categorize your logs, right? Right?
Well then you've already built out your debug logging levels!
Want to only include a certain set of your very noisy debug logs?
Just specify which label you want included in the debug level.

```go
set := clog.Settings{
  Format: clog.FormatForHumans,
  Level: clog.LevelDebug,
  OnlyLogDebugIfContainsLabel: []string{clog.APICall},
}

ctx := clog.Init(ctx, set)
```
