# Cecrets

/ˈsiː.krətz/

_noun_

1. 7 method, formula, or process used in an art or operation
   and divulged only to those of one's own company or craft.
2. something kept from the knowledge of others or shared
   only confidentially with a few.

_adjective_

1. Marked by the habit of discretion.

---

## Concealing sensitive data and pii

Cecrets is a pii obfuscation tool for Clues attributes, designed
to provide ad-hoc or automated hashing of sensitive data within
telemetry.

## Ad-hoc hashing

Cecrets can be used to wrap any Clues attribute.

```go
func foo(ctx context.Context) {
  clog.Ctx(ctx).Infow(
    "user identifiers",
    "user.id", cecrets.Hide(user.ID),
    "user.phone", cecrets.Hide(user.phone),
  )
}
```

## Interfaced hashing

Alternatively, you can have any struct comply with the `Concealer`
interface. Clues attribute processing will prefer this stringifier
in all telemetry serialization.

```go
type User struct {
  ID    string
  Phone string
}

func (u User) Conceal() string {
  return fmt.Sprintf("user: %s", cecrets.Hide(u.ID))
}

func (u User) PlainString() string {
  return fmt.Sprintf("%+v", u)
}

func (u User) String() string {
  return u.PlainString(u)
}

func foo(ctx context.Context) {
  clog.Ctx(ctx).Infow(
    "user identifiers",
    // automatically calls user.Conceal()
    "user", user,
  )
}
```
