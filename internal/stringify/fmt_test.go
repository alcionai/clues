package stringify

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type aStringer struct {
	v any
}

func (a aStringer) String() string {
	return fmt.Sprintf("%v", a.v)
}

type notAStringer struct {
	v any
}

var _ Concealer = &aConcealer{}

type aConcealer struct {
	v any
}

func (a aConcealer) Conceal() string                { return "***" }
func (a aConcealer) Format(fs fmt.State, verb rune) { io.WriteString(fs, "***") }
func (a aConcealer) PlainString() string            { return fmt.Sprintf("%v", a.v) }

type aPtrStruct struct {
	val   *string
	inner *aPtrInnerStruct
}

type aPtrInnerStruct struct {
	val *string
}

type aFormatter struct {
	v *string
}

func (a aFormatter) Format(fs fmt.State, verb rune) { io.WriteString(fs, "formatted") }

func TestFmt(t *testing.T) {
	ptrTestString := "ptrString"
	ptrString := &ptrTestString

	structWithPointerFields := &aPtrStruct{
		val: ptrString,
		inner: &aPtrInnerStruct{
			val: ptrString,
		},
	}

	table := []struct {
		name   string
		input  []any
		expect []string
	}{
		{
			name:   "nil",
			input:  nil,
			expect: []string{},
		},
		{
			name:   "any is nil",
			input:  []any{nil},
			expect: []string{""},
		},
		{
			name:   "string",
			input:  []any{"fisher flannigan fitzbog"},
			expect: []string{"fisher flannigan fitzbog"},
		},
		{
			name:   "number",
			input:  []any{-1.2345},
			expect: []string{"-1.2345"},
		},
		{
			name:   "slice",
			input:  []any{[]int{1, 2, 3, 4, 5}},
			expect: []string{"[1 2 3 4 5]"},
		},
		{
			name: "map",
			input: []any{map[string]struct{}{
				"fisher flannigan fitzbog": struct{}{},
			}},
			expect: []string{`map[fisher flannigan fitzbog:{}]`},
		},
		{
			name:   "map sort keys",
			input:  []any{map[string]string{"a": "a", "c": "c", "b": "b"}},
			expect: []string{"map[a:a b:b c:c]"},
		},
		{
			name:   "concealer",
			input:  []any{aConcealer{"fisher flannigan fitzbog"}},
			expect: []string{"***"},
		},
		{
			name:   "stringer",
			input:  []any{aStringer{"I have seen the fnords."}},
			expect: []string{"I have seen the fnords."},
		},
		{
			name:   "not a stringer",
			input:  []any{notAStringer{"I have seen the fnords."}},
			expect: []string{"{v:I have seen the fnords.}"},
		},
		{
			name:   "many values",
			input:  []any{1, "a", true, aStringer{"smarf"}},
			expect: []string{"1", "a", "true", "smarf"},
		},
		{
			name:   "ptr value",
			input:  []any{ptrString},
			expect: []string{fmt.Sprintf("<*>(%p)%s", ptrString, ptrTestString)},
		},
		{
			name:  "ptr struct",
			input: []any{structWithPointerFields},
			expect: []string{fmt.Sprintf("<*>(%p){val:<*>(%p)%s inner:<*>(%p){val:<*>(%p)%s}}",
				structWithPointerFields,
				ptrString,
				ptrTestString,
				structWithPointerFields.inner,
				ptrString,
				ptrTestString,
			)},
		},
		{
			name:   "formatter",
			input:  []any{aFormatter{ptrString}},
			expect: []string{"formatted"},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := Fmt(test.input...)
			assert.Equal(t, test.expect, result)
		})
	}
}

func TestNormalize(t *testing.T) {
	table := []struct {
		name   string
		input  []any
		expect map[string]any
	}{
		{
			name:   "nil",
			input:  nil,
			expect: map[string]any{},
		},
		{
			name:   "any is nil",
			input:  []any{nil},
			expect: map[string]any{"": nil},
		},
		{
			name:   "string",
			input:  []any{"fisher flannigan", "fitzbog"},
			expect: map[string]any{"fisher flannigan": "fitzbog"},
		},
		{
			name:   "number",
			input:  []any{-1.2345, 54321},
			expect: map[string]any{"-1.2345": "54321"},
		},
		{
			name:   "slice",
			input:  []any{[]int{1, 2, 3, 4, 5}, []string{"a", "b"}},
			expect: map[string]any{"[1 2 3 4 5]": "[a b]"},
		},
		{
			name: "map",
			input: []any{
				map[string]struct{}{
					"fisher flannigan fitzbog": struct{}{},
				},
				map[int]int{},
			},
			expect: map[string]any{`map[fisher flannigan fitzbog:{}]`: "map[]"},
		},
		{
			name:   "concealer",
			input:  []any{aConcealer{"fisher flannigan fitzbog"}},
			expect: map[string]any{"***": nil},
		},
		{
			name:   "stringer",
			input:  []any{aStringer{"I have seen the fnords."}},
			expect: map[string]any{"I have seen the fnords.": nil},
		},
		{
			name:   "not a stringer",
			input:  []any{notAStringer{"I have seen the fnords."}},
			expect: map[string]any{"{v:I have seen the fnords.}": nil},
		},
		{
			name:  "many values",
			input: []any{1, "a", true, aStringer{"smarf"}},
			expect: map[string]any{
				"1":    "a",
				"true": "smarf",
			},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := Normalize(test.input...)
			assert.Equal(t, test.expect, result)
		})
	}
}
