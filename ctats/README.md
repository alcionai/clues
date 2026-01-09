# Ctats

/stæts/

_noun_

1. A random variable that takes on the possible values of a statistic.

---

## Easier OTEL metrics

OTEL metrics, despite being an invaluable addition to service telemetry,
require an obnoxiously verbose setup and implementation. Ctats isn't
here to provide any new features. Instead in wants to make the current
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
  ctx, err := ctats.RegisterHistogram(
    ctx,
    "http.server.latency", // Name
    "ms", // Unit
    "New user additions.", // Description
  )
}
```

### Step 4: Use them

```go
func handler(ctx context.Context) {
  //...
  ctats.Histogram[int64]("http.server.latency").Record(latency)
  //...

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

## Corner Case: histogram bucket definitions

You can't define your histogram buckets with Ctats. Why? Because
[OTEL doesn't let you define them at runtime either](https://github.com/open-telemetry/opentelemetry-go/issues/3826).
You'll have to take it up with the package authors, not us.

## Sum vs Counter vs Gauge

Feeling overwhelmed?  Not sure which type to pick?  Just answer
these simple questions and you'll be a master in no time!

* Sum -> OTEL Counter
* Counter -> OTEL UpDownCounter
* Gauge -> OTEL Gauge (who knew?)

Do you need `Delta Temporality`? Use a Sum, it's your only option!

Do you need to decrement values? Use a Counter!

Do you need have a single threaded, single source of truth? Try
a Gauge!

Sums are the most foolproof option all around.  Plug one in,
count away.  Counters are nearly as good, if it weren't for the
temporality constraint.  For monotonically increasing values,
you can't really go wrong either way.
