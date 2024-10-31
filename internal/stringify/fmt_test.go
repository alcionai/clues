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

func TestFmt(t *testing.T) {
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
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := Fmt(test.input...)
			assert.Equal(t, test.expect, result)
		})
	}
}