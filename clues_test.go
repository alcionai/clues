package clues_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ryanfkeepers/clues"
)

type msa map[string]any

func (m msa) stringWith(other map[string]any) string {
	return fmt.Sprintf(
		"\nexpected: %+v\nreceived: %+v\n",
		m, other,
	)
}

func (m msa) equals(t *testing.T, other map[string]any) {
	if len(m) != len(other) {
		t.Errorf(
			"expected map of len [%d], received len [%d]\n%s",
			len(m), len(other), m.stringWith(other),
		)
	}

	for k, v := range m {
		if other[k] != v {
			t.Errorf(
				"expected map to contain key:value [%s: %v]\n%s",
				k, v, m.stringWith(other),
			)
		}
	}

	for k, v := range other {
		if m[k] != v {
			t.Errorf(
				"map contains unexpected key:value [%s: %v]\n%s",
				k, v, m.stringWith(other),
			)
		}
	}
}

type sa []any

func (s sa) stringWith(other []any) string {
	return fmt.Sprintf(
		"\nexpected: %+v\nreceived: %+v\n",
		s, other,
	)
}

func (s sa) equals(t *testing.T, other []any) {
	if len(s) != len(other) {
		t.Errorf(
			"expected slice of len [%d], received len [%d]\n%s",
			len(s), len(other), s.stringWith(other),
		)
	}

	for i, v := range s {
		if other[i] != v {
			t.Errorf(
				"expected slice to contain [%v] at index [%d]\n%s",
				v, i, s.stringWith(other),
			)
		}
	}

	for i, v := range other {
		if s[i] != v {
			t.Errorf(
				"did not expect slice to contain [%v] at index [%d]\n%s",
				v, i, s.stringWith(other),
			)
		}
	}
}

func assert(
	t *testing.T,
	ctx context.Context,
	ns string,
	eM, eMns msa,
	eS, eSns sa,
) {
	m := clues.Namespace(ctx, ns)
	eMns.equals(t, m)
	m = clues.Values(ctx)
	eM.equals(t, m)
	s := clues.NameSlice(ctx, ns)
	eSns.equals(t, s)
	s = clues.Slice(ctx)
	eS.equals(t, s)
}

func TestAdd(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name    string
		kvs     [][]string
		expectM msa
		expectS sa
	}{
		{"single", [][]string{{"k", "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", [][]string{{"a", "1"}, {"b", "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			c := ctx
			for _, kv := range test.kvs {
				c = clues.Add(ctx, kv[0], kv[1])
			}
			assert(
				t, c, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

func TestAddAll(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name    string
		kvs     [][]string
		expectM msa
		expectS sa
	}{
		{"single", [][]string{{"k", "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", [][]string{{"a", "1"}, {"b", "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"duplicates", [][]string{{"a", "1"}, {"a", "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"imbalanced", [][]string{{"a"}}, msa{"a": nil}, sa{"a", ""}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			c := ctx
			for _, kv := range test.kvs {
				c = clues.Add(ctx, kv[0], kv[1])
			}
			assert(
				t, c, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

func TestAddMap(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name    string
		ms      []msa
		expectM msa
		expectS sa
	}{
		{"single", []msa{{"k": "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", []msa{{"a": "1"}, {"b": "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"duplicate", []msa{{"a": "1"}, {"a": "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"none", []msa{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			c := ctx
			for _, m := range test.ms {
				c = clues.AddMap(ctx, m)
			}
			assert(
				t, c, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

func TestAddTo(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name    string
		kvs     [][]string
		expectM msa
		expectS sa
	}{
		{"single", [][]string{{"k", "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", [][]string{{"a", "1"}, {"b", "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			c := ctx
			for _, kv := range test.kvs {
				c = clues.AddTo(ctx, "ns", kv[0], kv[1])
			}
			assert(
				t, c, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}

func TestAddAllTo(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name    string
		kvs     [][]string
		expectM msa
		expectS sa
	}{
		{"single", [][]string{{"k", "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", [][]string{{"a", "1"}, {"b", "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"duplicates", [][]string{{"a", "1"}, {"a", "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"imbalanced", [][]string{{"a"}}, msa{"a": nil}, sa{"a", ""}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			c := ctx
			for _, kv := range test.kvs {
				c = clues.AddTo(ctx, "ns", kv[0], kv[1])
			}
			assert(
				t, c, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}

func TestAddMapTo(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name    string
		ms      []msa
		expectM msa
		expectS sa
	}{
		{"single", []msa{{"k": "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", []msa{{"a": "1"}, {"b": "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"duplicate", []msa{{"a": "1"}, {"a": "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"none", []msa{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			c := ctx
			for _, m := range test.ms {
				c = clues.AddMapTo(ctx, "ns", m)
			}
			assert(
				t, c, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}
