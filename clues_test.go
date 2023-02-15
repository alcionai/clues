package clues_test

import (
	"context"
	"encoding/json"
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

func toMSA[T any](m map[string]T) msa {
	to := make(msa, len(m))
	for k, v := range m {
		to[k] = v
	}

	return to
}

func (m msa) equals(t *testing.T, other map[string]any) {
	ma := maa(toMAA(m))
	a := toMAA(other)

	ma.equals(t, a)
}

type maa map[any]any

func (m maa) stringWith(other map[any]any) string {
	return fmt.Sprintf(
		"expected: %+v\nreceived: %+v\n\n",
		m, other,
	)
}

func (m maa) equals(t *testing.T, other map[any]any) {
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

func toMAA(m map[string]any) map[any]any {
	a := make(map[any]any, len(m))
	for k, v := range m {
		a[k] = v
	}

	return a
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
	vs := clues.In(ctx)
	nvs := clues.InNamespace(ctx, ns)
	eM.equals(t, vs)
	eMns.equals(t, nvs)
	eS.equals(t, vs.Slice())
	eSns.equals(t, nvs.Slice())
}

func assertMAA(
	t *testing.T,
	ctx context.Context,
	ns string,
	eM, eMns maa,
	eS, eSns sa,
) {
	vs := clues.In(ctx)
	nvs := clues.InNamespace(ctx, ns)
	eM.equals(t, toMAA(vs))
	eMns.equals(t, toMAA(nvs))
	eS.equals(t, vs.Slice())
	eSns.equals(t, nvs.Slice())
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
		{"duplicates", [][]string{{"a", "1"}, {"a", "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.In(ctx))

			for _, kv := range test.kvs {
				ctx = clues.Add(ctx, kv[0], kv[1])
				check[kv[0]] = kv[1]
				check.equals(t, clues.In(ctx))
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
			check.equals(t, clues.In(ctx))

			for _, m := range test.ms {
				ctx = clues.AddMap(ctx, m)
				for k, v := range m {
					check[k] = v
				}
				check.equals(t, clues.In(ctx))
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
		{"duplicates", [][]string{{"a", "1"}, {"a", "2"}}, msa{"a": "2"}, sa{"a", "2"}},
		{"none", [][]string{}, msa{}, sa{}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			check.equals(t, clues.InNamespace(ctx, "ns"))

			for _, kv := range test.kvs {
				ctx = clues.AddTo(ctx, "ns", kv[0], kv[1])
				check[kv[0]] = kv[1]
				check.equals(t, clues.InNamespace(ctx, "ns"))
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
			check.equals(t, clues.InNamespace(ctx, "ns"))

			for _, m := range test.ms {
				ctx = clues.AddMapTo(ctx, "ns", m)
				for k, v := range m {
					check[k] = v
				}
				check.equals(t, clues.InNamespace(ctx, "ns"))
			}

			assert(
				t, ctx, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}

func TestImmutableCtx(t *testing.T) {
	ctx := context.WithValue(context.Background(), testCtx{}, "instance")
	check := msa{}
	pre := clues.In(ctx)
	check.equals(t, pre)

	ctx2 := clues.Add(ctx, "k", "v")
	if _, ok := pre["k"]; ok {
		t.Errorf("previous map should not have been mutated by addition")
	}

	pre = clues.In(ctx)
	if _, ok := pre["k"]; ok {
		t.Errorf("previous map within ctx should not have been mutated by addition")
	}

	post := clues.In(ctx2)
	if post["k"] != "v" {
		t.Errorf("new map should contain the added value")
	}
}

type safe struct {
	v any
}

func (s safe) Conceal() string {
	bs, err := json.Marshal(s.v)
	if err != nil {
		return "ERR MARSHALLING"
	}

	return string(bs)
}

type custom struct {
	a, b string
}

func (c custom) Conceal() string {
	return c.a + " - " + clues.Conceal(clues.SHA256, c.b)
}

func concealed(a any) string {
	c, ok := a.(clues.Concealer)
	if !ok {
		return "NOT CONCEALER"
	}

	return c.Conceal()
}

func TestAdd_concealed(t *testing.T) {
	table := []struct {
		name       string
		concealers [][]any
		expectM    maa
		expectS    sa
	}{
		{
			name:       "all hidden",
			concealers: [][]any{{clues.Hide("k"), clues.Hide("v")}, {clues.Hide("not_k"), clues.Hide("not_v")}},
			expectM:    maa{"ec084d54826cf369": "072553c49de59ecf", "1d3298b660d45ba6": "6c33ba4c0581b0cc"},
			expectS:    sa{"ec084d54826cf369", "072553c49de59ecf", "1d3298b660d45ba6", "6c33ba4c0581b0cc"},
		},
		{
			name:       "partially hidden",
			concealers: [][]any{{clues.Hide("a"), safe{1}}, {clues.Hide(2), safe{"b"}}},
			expectM:    maa{"7804cbb0587c4711": "1", "6679863f298e5446": `"b"`},
			expectS:    sa{"7804cbb0587c4711", "1", "6679863f298e5446", `"b"`},
		},
		{
			name: "custom concealer",
			concealers: [][]any{
				{custom{"foo", "bar"}, custom{"baz", "qux"}},
				{custom{"fnords", "smarf"}, custom{"beau", "regard"}}},
			expectM: maa{"foo - fcde2b2edba56bf4": "baz - 21f58d27f827d295", "fnords - dd738d92a334bb85": "beau - fe099a0620ce9759"},
			expectS: sa{"foo - fcde2b2edba56bf4", "baz - 21f58d27f827d295", "fnords - dd738d92a334bb85", "beau - fe099a0620ce9759"},
		},
		{
			name:       "none",
			concealers: [][]any{},
			expectM:    maa{},
			expectS:    sa{},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := maa{}
			check.equals(t, toMAA(clues.In(ctx)))

			for _, cs := range test.concealers {
				ctx = clues.Add(ctx, cs...)
				check[concealed(cs[0])] = concealed(cs[1])
				check.equals(t, toMAA(clues.In(ctx)))
			}

			assertMAA(
				t, ctx, "",
				test.expectM, maa{},
				test.expectS, sa{})
		})
	}
}
