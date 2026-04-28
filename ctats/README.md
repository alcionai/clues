# Ctats

/stæts/

_noun_

1. A random variable that takes on the possible values of a statistic.

---

## Easier OTEL metrics

OTEL metrics, despite being an invaluable addition to service telemetry,
require an obnoxiously verbose setup and implementation. Ctats isn't
here to provide any new features. Instead it wants to make the current
features more accessible and less painful.

### Step 1: Init OTEL with Clues

```go
func main() {
 ctx, err = clues.InitializeOTEL(
  context.Background(),
  serviceName,
  clues.OTELConfig{
   Resource:     r,
   GRPCEndpoint: otelEnvs.OTELGRPCEndpoint,
  },
 )
  // ...
}
```

### Step 2: Init Ctats

```go
func main() {
  // ...
  ctx, err = ctats.Initialize(ctx)
  // ...
}
```

### Step 3: Declare your metrics... or don't

```go
func main() {
  // We're not kidding, this step is purely optional.
  ctx, err := ctats.RegisterSum(
    ctx,
    "http.server.requests", // Name
    "1",                    // Unit
    "Incoming HTTP requests by status code.", // Description
  )
}
```

### Step 4: Use them

```go
func handler(ctx context.Context) {
  // ...
  ctats.Sum[int64]("http.server.requests").
    With("status_code", statusCode).
    Inc(ctx)
  // ...
}
```

## How it works

Ctats maintains an in-memory cache of all registered metrics.
Registration is a first-come, first-served process. If you don't
declare your metrics up front, that's no problem! They'll get
initialized the first time the metric gets recorded.

The variety of metric types available through the golang OTEL
package has been distilled down to a few basic types: Counters
(the up-down variety), Gauges, and Histograms. Why the reduction?
Because 99% of the time this is all your developers will need.

Better to have quick access to simple and effective tools- even
if you have fewer of them- than to lose out on service insights
because the available tools are frustrating to use.

## Corner Case: type contention

What happens when you try to register a metric twice with different
types?

```go
  ctats.Counter[int64]("foo").Inc()
  // oh no, type contention!
  ctats.Counter[float64]("foo").Inc()
```

Well... everything will work just fine. Why? Because all Ctats
values are `float64`s behind the scenes. Easier to avoid the problem
of potential conflicts altogether. What, would you prefer that we
panic?

## Histograms

Histograms are a bit more work than the other types because you have to think
about your data's distribution first. This is because the OTEL histograms
store observations in pre-defined buckets. The default boundaries top out at
**10,000** — anything above that disappears into an overflow bucket.

[This explainer](https://signoz.io/blog/opentelemetry-histogram/) is a good read
if you want a deeper understanding of OTEL Histograms.

So, how do you set yourself up for histogram success using ctats?
Just register your buckets on init!  Simple as that.

```go
func main() {
  ctx, err := ctats.RegisterHistogram(
    ctx,
    "op.latency",
    "ms",
    "End-to-end operation latency.",
    ctats.WithBoundaries(ctats.PresetLatencyBoundariesMs...),
  )
}

func handler(ctx context.Context) {
  ctats.Histogram[int64]("op.latency").Record(ctx, elapsed)
}
```

Registering is optional. You can also pass `WithBoundaries` directly to the
factory and the instrument is created on the first `Record` call. Just keep
in mind that the first creation wins — if the same id was already registered
or recorded against with different boundaries, the new ones are silently
ignored.

### Picking your boundaries

For latency in milliseconds, `PresetLatencyBoundariesMs` is a sensible
default: 15 logarithmically-spaced buckets from **1 ms to 60,000 ms**, with
finer resolution at the low end where most data clusters.

If your data has a different shape, use `ExponentialBoundaries` to generate
your own range. Note that `min` must be greater than zero — the boundaries
are log-spaced so zero has no meaningful place in the range:

```go
// background job duration in seconds: expected to time out at 1 hour
boundaries := ctats.ExponentialBoundaries(1, 3_600, 10)

ctats.Histogram[int64](
    "job.duration",
    ctats.WithBoundaries(boundaries...),
).Record(ctx, elapsed)
```

## Which metric type should I use?

Feeling overwhelmed?  Not sure which type to pick?  Just answer
these simple questions and you'll be a master in no time!

* Sum -> OTEL Counter
* Counter -> OTEL UpDownCounter
* Gauge -> OTEL Gauge (who knew?)
* Histogram -> OTEL Histogram (surprise!)

Do you need `Delta Temporality`? Use a Sum, it's your only option!

Do you need to decrement values? Use a Counter!

Do you need have a single threaded, single source of truth? Try
a Gauge!

Do you need statistics such as percentiles? Use a Histogram!

Sums are the most foolproof option all around.  Plug one in,
count away.  Counters are nearly as good, if it weren't for the
temporality constraint.  For monotonically increasing values,
you can't really go wrong either way.
