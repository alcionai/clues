package clues

import (
	"fmt"
	"reflect"
)

// TODO: move this and secrets.go to its own package

// Fmt is an internal func that's currently exposed to get around some
// import dependencies.  It runs the marshal func for all values provided
// to it, which will stringify the values according to the clues marshaler
// and concealer rules.
func Fmt(vals ...any) []string {
	resp := make([]string, len(vals))

	for _, v := range vals {
		resp = append(resp, marshal(v, false))
	}

	return resp
}

// normalize ensures that the variadic of key-value pairs is even in length,
// and then transforms that slice of values into a map[string]any, where all
// keys are transformed to string using the marshal() func.
func normalize(kvs ...any) map[string]any {
	norm := map[string]any{}

	for i := 0; i < len(kvs); i += 2 {
		key := marshal(kvs[i], true)

		var value any
		if i+1 < len(kvs) {
			value = marshal(kvs[i+1], true)
		}

		norm[key] = value
	}

	return norm
}

// marshal is the central marshalling handler for the entire package.  All
// stringification of values comes down to this function.  Priority for
// stringification follows this order:
// 1. nil -> ""
// 2. conceal all concealer interfaces
// 3. flat string values
// 4. string all stringer interfaces
// 5. fmt.sprintf the rest
func marshal(a any, conceal bool) string {
	if a == nil {
		return ""
	}

	// protect against nil pointer values with value-receiver funcs
	rvo := reflect.ValueOf(a)
	if rvo.Kind() == reflect.Ptr && rvo.IsNil() {
		return ""
	}

	if as, ok := a.(Concealer); conceal && ok {
		return as.Conceal()
	}

	if as, ok := a.(string); ok {
		return as
	}

	if as, ok := a.(fmt.Stringer); ok {
		return as.String()
	}

	return fmt.Sprintf("%+v", a)
}
