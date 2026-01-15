package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func TestMapToOTELAttribute_Primitives(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  any
		want attribute.KeyValue
	}{
		{"string", "k", "v", attribute.String("k", "v")},
		{"bool", "k", true, attribute.Bool("k", true)},
		{"int", "k", int(5), attribute.Int("k", 5)},
		{"int64", "k", int64(6), attribute.Int64("k", 6)},
		{"uint", "k", uint(7), attribute.Int64("k", 7)},
		{"float32", "k", float32(1.5), attribute.Float64("k", 1.5)},
		{"float64", "k", float64(2.5), attribute.Float64("k", 2.5)},
		{"fallback", "k", struct{ X string }{"x"}, attribute.String("k", "{X:x}")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapToOTELAttribute(tc.key, tc.val)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMapToOTELAttribute_Slices(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  any
		want attribute.KeyValue
	}{
		{"strings", "k", []string{"a", "b"}, attribute.StringSlice("k", []string{"a", "b"})},
		{"ints", "k", []int{1, 2}, attribute.Int64Slice("k", []int64{1, 2})},
		{"uints", "k", []uint{3, 4}, attribute.Int64Slice("k", []int64{3, 4})},
		{"floats", "k", []float32{1.0, 2.0}, attribute.Float64Slice("k", []float64{1.0, 2.0})},
		{"bools", "k", []bool{true, false}, attribute.BoolSlice("k", []bool{true, false})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MapToOTELAttribute(tc.key, tc.val)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMapToOTELAttributes_Map(t *testing.T) {
	m := map[string]any{
		"s":  "v",
		"i":  int64(1),
		"u":  uint(2),
		"f":  1.5,
		"bs": []bool{true, false},
	}

	got := MapToOTELAttributes(m)
	require.Len(t, got, len(m))

	// Convert slice to map for easy lookup
	result := map[string]attribute.Value{}
	for _, kv := range got {
		result[string(kv.Key)] = kv.Value
	}

	assert.Equal(t, attribute.StringValue("v"), result["s"])
	assert.Equal(t, attribute.Int64Value(1), result["i"])
	assert.Equal(t, attribute.Int64Value(2), result["u"])
	assert.Equal(t, attribute.Float64Value(1.5), result["f"])
	assert.Equal(t, attribute.BoolSliceValue([]bool{true, false}), result["bs"])
}

func TestNumericHelpers(t *testing.T) {
	assert.Equal(t, int64(5), toInt64(int(5)))
	assert.Equal(t, int64(6), toInt64(int64(6)))
	assert.Equal(t, uint64(7), toUint64(uint(7)))
	assert.Equal(t, uint64(8), toUint64(int64(8)))

	assert.Equal(t, []int64{1, 2}, toInt64Slice([]int{1, 2}))
	assert.Equal(t, []int64{3, 4}, toInt64Slice([]uint{3, 4}))
	assert.Equal(t, []float64{1.0, 2.0}, toFloat64Slice([]float32{1.0, 2.0}))
}
