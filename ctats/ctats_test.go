package ctats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatID(t *testing.T) {
	table := []struct {
		name   string
		in     string
		expect string
	}{
		{
			name:   "empty",
			in:     "",
			expect: "",
		},
		{
			name:   "simple",
			in:     "foobarbaz",
			expect: "foobarbaz",
		},
		{
			name:   "already correct",
			in:     "foo.bar.baz",
			expect: "foo.bar.baz",
		},
		{
			name:   "only underscore delimited",
			in:     "foo_bar_baz",
			expect: "foo_bar_baz",
		},
		{
			name:   "spaces to underscores",
			in:     "foo bar baz",
			expect: "foo_bar_baz",
		},
		{
			name:   "camel case",
			in:     "FooBarBaz",
			expect: "foo.bar.baz",
		},
		{
			name:   "all caps",
			in:     "FOOBARBAZ",
			expect: "foobarbaz",
		},
		{
			name:   "kebab case",
			in:     "foo-bar-baz",
			expect: "foo.bar.baz",
		},
		{
			name:   "mixed",
			in:     "fooBar baz-fnords",
			expect: "foo.bar_baz.fnords",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := formatID(test.in)
			assert.Equal(t, test.expect, result, "input: %s", test.in)
		})
	}
}
