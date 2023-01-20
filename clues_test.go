package clues_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/alcionai/clues"
)

type msa map[string]any

func (m msa) stringWith(other map[string]any) string {
	return fmt.Sprintf(
		"expected: %+v\nreceived: %+v\n\n",
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

	for _, v := range s {
		var found bool
		for _, o := range other {
			if v == o {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected slice to contain [%v]\n%s", v, s.stringWith(other))
		}
	}

	for _, o := range other {
		var found bool
		for _, v := range s {
			if v == o {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("did not expect slice to contain [%v]\n%s", o, s.stringWith(other))
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
	eM.equals(t, clues.Values(ctx))
	eMns.equals(t, clues.Namespace(ctx, ns))
	eS.equals(t, clues.Slice(ctx))
	eSns.equals(t, clues.NameSlice(ctx, ns))
}

type testCtx struct{}

func TestAdd(t *testing.T) {
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
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.Values(ctx))

			for _, kv := range test.kvs {
				ctx = clues.Add(ctx, kv[0], kv[1])
				check[kv[0]] = kv[1]
				check.equals(t, clues.Values(ctx))
			}

			assert(
				t, ctx, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

func TestAddAll(t *testing.T) {
	table := []struct {
		name    string
		kvs     [][]string
		expectM msa
		expectS sa
	}{
		{"single", [][]string{{"k", "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", [][]string{{"a", "1"}, {"b", "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"duplicates", [][]string{{"a", "1"}, {"a", "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.Values(ctx))

			for _, kv := range test.kvs {
				ctx = clues.AddAll(ctx, kv[0], kv[1])
				check[kv[0]] = kv[1]
				check.equals(t, clues.Values(ctx))
			}

			assert(
				t, ctx, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

func TestAddMap(t *testing.T) {
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
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.Values(ctx))

			for _, m := range test.ms {
				ctx = clues.AddMap(ctx, m)
				for k, v := range m {
					check[k] = v
				}
				check.equals(t, clues.Values(ctx))
			}

			assert(
				t, ctx, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

func TestAddTo(t *testing.T) {
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
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.Namespace(ctx, "ns"))

			for _, kv := range test.kvs {
				ctx = clues.AddTo(ctx, "ns", kv[0], kv[1])
				check[kv[0]] = kv[1]
				check.equals(t, clues.Namespace(ctx, "ns"))
			}

			assert(
				t, ctx, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}

func TestAddAllTo(t *testing.T) {
	table := []struct {
		name    string
		kvs     [][]string
		expectM msa
		expectS sa
	}{
		{"single", [][]string{{"k", "v"}}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple", [][]string{{"a", "1"}, {"b", "2"}}, msa{"a": "1", "b": "2"}, sa{"a", "1", "b", "2"}},
		{"duplicates", [][]string{{"a", "1"}, {"a", "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.Namespace(ctx, "ns"))

			for _, kv := range test.kvs {
				ctx = clues.AddAllTo(ctx, "ns", kv[0], kv[1])
				check[kv[0]] = kv[1]
				check.equals(t, clues.Namespace(ctx, "ns"))
			}

			assert(
				t, ctx, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}

func TestAddMapTo(t *testing.T) {
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
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.Namespace(ctx, "ns"))

			for _, m := range test.ms {
				ctx = clues.AddMapTo(ctx, "ns", m)
				for k, v := range m {
					check[k] = v
				}
				check.equals(t, clues.Namespace(ctx, "ns"))
			}

			assert(
				t, ctx, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}
