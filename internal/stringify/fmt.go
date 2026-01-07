package stringify

import (
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

// ---------------------------------------------------------------------------
// types and interfaces
// ---------------------------------------------------------------------------

// duplicated here from secrets to avoid import cycles.
// Since this is an internal package, we don't let end users see it.
type Concealer interface {
	// Conceal produces an obfuscated representation of the value.
	Conceal() string
	// Concealers also need to comply with Format.
	// Complying with Conceal() alone doesn't guarantee that
	// the variable won't pass into fmt.Printf("%v") and skip
	// the whole conceal process.
	Format(fs fmt.State, verb rune)
	// PlainStringer is the opposite of conceal.
	// Useful for if you want to retrieve the raw value of a secret.
	PlainString() string
}

// ---------------------------------------------------------------------------
// funcs
// ---------------------------------------------------------------------------

// Fmt is an internal func that's currently exposed to get around some
// import dependencies.  It runs the marshal func for all values provided
// to it, which will stringify the values according to the clues marshaler
// and concealer rules.
func Fmt(vals ...any) []string {
	resp := make([]string, 0, len(vals))

	for _, v := range vals {
		resp = append(resp, Marshal(v, false))
	}

	return resp
}

// Marshal is the central marshalling handler for the entire package.  All
// stringification of values comes down to this function.  Priority for
// stringification follows this order:
// 1. nil -> ""
// 2. conceal all concealer interfaces
// 3. flat string values
// 4. Slog LogValuer interfaces
// 5. string all stringer interfaces
// 6. fmt.sprintf all formatter interfaces
// 7. spew.Sprintf the rest
func Marshal(a any, shouldConceal bool) string {
	if a == nil {
		return ""
	}

	// protect against nil pointer values with value-receiver funcs
	rvo := reflect.ValueOf(a)
	if rvo.Kind() == reflect.Ptr && rvo.IsNil() {
		return ""
	}

	if as, ok := a.(Concealer); shouldConceal && ok {
		return as.Conceal()
	}

	if as, ok := a.(string); ok {
		return as
	}

	if as, ok := a.(slog.LogValuer); ok {
		return marshalSlogValue(as.LogValue())
	}

	if as, ok := a.(fmt.Stringer); ok {
		return as.String()
	}

	// If value implements fmt.Formatter, then we do not need any additional formatting.
	if _, ok := a.(fmt.Formatter); ok {
		return fmt.Sprintf("%+v", a)
	}

	cfg := spew.NewDefaultConfig()
	cfg.SortKeys = true

	return cfg.Sprintf("%+v", a)
}

// Normalize ensures that the variadic of key-value pairs is even in length,
// and then transforms that slice of values into a map[string]any, where all
// keys are transformed to string using the marshal() func.
func Normalize(kvs ...any) map[string]any {
	norm := map[string]any{}

	for i := 0; i < len(kvs); i += 2 {
		key := Marshal(kvs[i], true)

		var value any
		if i+1 < len(kvs) {
			value = Marshal(kvs[i+1], true)
		}

		norm[key] = value
	}

	return norm
}

// Stringalize is a convenience function that does the exact same thing as
// Normalize, but it casts all values to string instead of any.
func Stringalize(kvs ...any) map[string]string {
	norm := Normalize(kvs...)
	stringy := make(map[string]string, len(norm))

	for k, v := range norm {
		sv, ok := v.(string)
		if !ok {
			sv = Marshal(v, false)
		}

		stringy[k] = sv
	}

	return stringy
}

func NormalizeMap[K comparable, V any](m map[K]V) map[string]any {
	kvs := make([]any, 0, len(m)*2)

	for k, v := range m {
		kvs = append(kvs, k, v)
	}

	return Normalize(kvs...)
}

// marshalSlogValue is a helper for iterating through slog.Value types.
//
// Output format should follow the pattern produced by spew Printf("%+v", v),
// including the alphabetical ordering of keys (we assert this option for spew
// prints as well).
// see: https://pkg.go.dev/github.com/davecgh/go-spew/spew
// That format isn't explicitly defined, but from observation the result is
// generally modeled as {key:value}.
//
// Ex: {str:some string int:42 struct:{nested:true}}
func marshalSlogValue(v slog.Value) string {
	// assume any non-group can be stringified directly.
	if v.Kind() != slog.KindGroup {
		return v.String()
	}

	buf := make([]byte, 0, 2)
	buf = append(buf, "{"...)

	attrs := v.Group()

	for i := range attrs {
		if len(attrs[i].Key) == 0 {
			attrs[i].Key = fmt.Sprintf("keyless-attr-%d", i)
		}
	}

	// order the attributes by key for consistency.
	slices.SortStableFunc(attrs, func(i, j slog.Attr) int {
		return strings.Compare(i.Key, j.Key)
	})

	for i, attr := range attrs {
		if i > 0 {
			buf = append(buf, " "...)
		}

		buf = appendVToSlogValueBuf(buf, attr)
	}

	buf = append(buf, "}"...)

	return string(buf)
}

// appendVToSlogValueBuf appends the slog.Attr key and value to the provided
// buffer, returning the appended buffer.  It is largely a convenience func
// to separate out the logic of appending slog.Value types.
func appendVToSlogValueBuf(buf []byte, attr slog.Attr) []byte {
	buf = append(buf, attr.Key...)
	buf = append(buf, ":"...)

	v := attr.Value

	// ensure we cover all kinds that are supported by slog.
	//exhaustive:enforce
	switch attr.Value.Kind() {
	case slog.KindGroup:
		return append(buf, marshalSlogValue(v)...)

	case slog.KindLogValuer:
		as, ok := v.Any().(slog.LogValuer)
		if ok {
			return append(buf, marshalSlogValue(as.LogValue())...)
		}

		fallthrough
	case slog.KindString, slog.KindInt64, slog.KindUint64, slog.KindFloat64,
		slog.KindBool, slog.KindDuration, slog.KindTime, slog.KindAny:
		return append(buf, v.String()...)

	default:
		// this is _extremely_ unlikely to happen, given that the slog.Value
		// kind is determined by the actual value itself, and not a metadata tag.
		// The result of String() is unknown in these cases.  This result gives
		// a graceful-but-informative fallback to work with.
		return append(buf, fmt.Sprintf("bad kind: %s; value: %s", v.Kind(), v.String())...)
	}
}
