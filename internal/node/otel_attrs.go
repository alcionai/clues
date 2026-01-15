package node

import (
	"reflect"

	"go.opentelemetry.io/otel/attribute"

	"github.com/alcionai/clues/internal/stringify"
)

// MapToOTELAttribute converts a single key/value to an otel attribute.
// Values are type-preserved where possible; everything else stringifies.
func MapToOTELAttribute(k string, v any) attribute.KeyValue {
	switch tv := v.(type) {
	case string:
		return attribute.String(k, tv)
	case bool:
		return attribute.Bool(k, tv)
	case int:
		return attribute.Int(k, tv)
	case int64:
		return attribute.Int64(k, tv)
	case int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		return attribute.Int64(k, toInt64(tv))
	case float32:
		return attribute.Float64(k, float64(tv))
	case float64:
		return attribute.Float64(k, tv)
	case []string:
		return attribute.StringSlice(k, tv)
	case []int64:
		return attribute.Int64Slice(k, tv)
	case []int, []int8, []int16, []int32, []uint, []uint8, []uint16, []uint32, []uint64:
		return attribute.Int64Slice(k, toInt64Slice(tv))
	case []float32, []float64:
		return attribute.Float64Slice(k, toFloat64Slice(tv))
	case []bool:
		return attribute.BoolSlice(k, tv)
	default:
		return attribute.String(k, stringify.Marshal(v, false))
	}
}

// MapToOTELAttributes converts a map to otel attributes using shared stringification.
func MapToOTELAttributes(m map[string]any) []attribute.KeyValue {
	if len(m) == 0 {
		return nil
	}

	attrs := make([]attribute.KeyValue, 0, len(m))

	for k, v := range m {
		attrs = append(attrs, MapToOTELAttribute(k, v))
	}

	return attrs
}

func toInt64(v any) int64 {
	switch tv := v.(type) {
	case int64:
		return tv
	case int, int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		return toInt64Numeric(tv)
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch tv := v.(type) {
	case int64:
		return uint64(tv)
	case int, int8, int16, int32, uint, uint8, uint16, uint32, uint64:
		return toUint64Numeric(tv)
	default:
		return 0
	}
}

func toInt64Slice(v any) []int64 {
	switch tv := v.(type) {
	case []int64:
		return tv
	default:
		return toInt64SliceNumeric(tv)
	}
}

func toFloat64Slice(v any) []float64 {
	switch tv := v.(type) {
	case []float64:
		return tv
	case []float32:
		out := make([]float64, len(tv))
		for i, val := range tv {
			out[i] = float64(val)
		}
		return out
	default:
		return nil
	}
}

func toInt64Numeric(v any) int64 {
	rv := reflect.ValueOf(v)

	if !rv.IsValid() {
		return 0
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Convert(reflect.TypeOf(int64(0))).Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(rv.Convert(reflect.TypeOf(uint64(0))).Uint())
	default:
		return 0
	}
}

func toUint64Numeric(v any) uint64 {
	rv := reflect.ValueOf(v)

	if !rv.IsValid() {
		return 0
	}

	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(rv.Convert(reflect.TypeOf(int64(0))).Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Convert(reflect.TypeOf(uint64(0))).Uint()
	default:
		return 0
	}
}

func toInt64SliceNumeric(v any) []int64 {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Slice {
		return nil
	}

	lenSlice := rv.Len()
	out := make([]int64, lenSlice)

	for i := 0; i < lenSlice; i++ {
		out[i] = toInt64Numeric(rv.Index(i).Interface())
	}

	return out
}
