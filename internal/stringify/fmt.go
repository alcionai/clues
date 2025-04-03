package stringify

import (
	"fmt"
	"reflect"

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
// 4. string all stringer interfaces
// 5. fmt.sprintf all formatter interfaces
// 6. spew.Sprintf the rest
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

	if as, ok := a.(fmt.Stringer); ok {
		return as.String()
	}

	// If value implements fmt.Formatter, then we do not need any additional formatting.
	if _, ok := a.(fmt.Formatter); ok {
		return fmt.Sprintf("%+v", a)
	}

	return spew.Sprintf("%+v", a)
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
