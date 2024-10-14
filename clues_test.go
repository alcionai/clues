package clues_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/alcionai/clues"
	"golang.org/x/exp/slices"
)

func mapEquals(
	t *testing.T,
	ctx context.Context,
	expect msa,
	expectCluesTrace bool,
) {
	mustEquals(
		t,
		expect,
		clues.In(ctx).Map(),
		expectCluesTrace)
}

func mustEquals[K comparable, V any](
	t *testing.T,
	expect, got map[K]V,
	hasCluesTrace bool,
) {
	e, g := toMSS(expect), toMSS(got)

	if len(g) > 0 {
		if _, ok := g["clues_trace"]; !ok && hasCluesTrace {
			t.Errorf(
				"expected map to contain key [clues_trace]\ngot: %+v",
				g)
		}
		delete(g, "clues_trace")
	}

	if len(e) != len(g) {
		t.Errorf(
			"expected map of len [%d], received len [%d]\n%s",
			len(e), len(g), expectedReceived(expect, got),
		)
	}

	for k, v := range e {
		if g[k] != v {
			t.Errorf(
				"expected map to contain key:value [%s: %s]\n%s",
				k, v, expectedReceived(expect, got),
			)
		}
	}

	for k, v := range g {
		if e[k] != v {
			t.Errorf(
				"map contains unexpected key:value [%s: %s]\n%s",
				k, v, expectedReceived(expect, got),
			)
		}
	}
}

func expectedReceived[K comparable, V any](e, r map[K]V) string {
	return fmt.Sprintf(
		"expected: %#v\nreceived: %#v\n\n",
		e, r)
}

type mss map[string]string

func toMSS[K comparable, V any](m map[K]V) mss {
	r := mss{}

	for k, v := range m {
		ks := fmt.Sprintf("%v", k)
		vs := fmt.Sprintf("%v", v)
		r[ks] = vs
	}

	return r
}

type msa map[string]any

func toMSA[T any](m map[string]T) msa {
	to := make(msa, len(m))
	for k, v := range m {
		to[k] = v
	}

	return to
}

type sa []any

func (s sa) stringWith(other []any) string {
	return fmt.Sprintf(
		"\nexpected: %+v\nreceived: %+v\n",
		s, other,
	)
}

func (s sa) equals(t *testing.T, other []any) {
	idx := slices.Index(other, "clues_trace")
	if idx >= 0 {
		other = append(other[:idx], other[idx+2:]...)
	}

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
	mustEquals(t, eM, vs.Map(), false)
	mustEquals(t, eMns, nvs.Map(), false)
	eS.equals(t, vs.Slice())
	eSns.equals(t, nvs.Slice())
}

func assertMSA(
	t *testing.T,
	ctx context.Context,
	ns string,
	eM, eMns msa,
	eS, eSns sa,
) {
	vs := clues.In(ctx)
	nvs := clues.InNamespace(ctx, ns)
	mustEquals(t, eM, toMSA(vs.Map()), false)
	mustEquals(t, eMns, toMSA(nvs.Map()), false)
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
			mustEquals(t, check, clues.In(ctx).Map(), false)

			for _, kv := range test.kvs {
				ctx = clues.Add(ctx, kv[0], kv[1])
				check[kv[0]] = kv[1]
				mustEquals(t, check, clues.In(ctx).Map(), false)
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
			mustEquals(t, check, clues.In(ctx).Map(), false)

			for _, m := range test.ms {
				ctx = clues.AddMap(ctx, m)
				for k, v := range m {
					check[k] = v
				}
				mustEquals(t, check, clues.In(ctx).Map(), false)
			}

			assert(
				t, ctx, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

func TestAddSpan(t *testing.T) {
	table := []struct {
		name        string
		names       []string
		expectTrace string
		kvs         sa
		expectM     msa
		expectS     sa
	}{
		{"single", []string{"single"}, "single", nil, msa{}, sa{}},
		{"multiple", []string{"single", "multiple"}, "single,multiple", nil, msa{}, sa{}},
		{"duplicates", []string{"single", "multiple", "multiple"}, "single,multiple,multiple", nil, msa{}, sa{}},
		{"single with kvs", []string{"single"}, "single", sa{"k", "v"}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple with kvs", []string{"single", "multiple"}, "single,multiple", sa{"k", "v"}, msa{"k": "v"}, sa{"k", "v"}},
		{"duplicates with kvs", []string{"single", "multiple", "multiple"}, "single,multiple,multiple", sa{"k", "v"}, msa{"k": "v"}, sa{"k", "v"}},
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			t.Run(test.name, func(t *testing.T) {
				ctx := context.Background()

				if init {
					ictx, err := clues.Initialize(ctx, test.name, clues.OTELConfig{
						GRPCEndpoint: "localhost:4317",
					})
					if err != nil {
						t.Error("initializing clues", err)
						return
					}

					defer func() {
						err := clues.Close(ictx)
						if err != nil {
							t.Error("closing clues:", err)
							return
						}
					}()

					ctx = ictx
				}

				ctx = context.WithValue(ctx, testCtx{}, "instance")
				mustEquals(t, msa{}, clues.In(ctx).Map(), false)

				for _, name := range test.names {
					ctx = clues.AddSpan(ctx, name, test.kvs...)
					defer clues.CloseSpan(ctx)
				}

				assert(
					t, ctx, "",
					test.expectM, msa{},
					test.expectS, sa{})

				c := clues.In(ctx).Map()
				if c["clues_trace"] != test.expectTrace {
					t.Errorf("expected clues_trace to equal %q, got %q", test.expectTrace, c["clues_trace"])
				}
			})
		}
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
			mustEquals(t, check, clues.InNamespace(ctx, "ns").Map(), false)

			for _, kv := range test.kvs {
				ctx = clues.AddTo(ctx, "ns", kv[0], kv[1])
				check[kv[0]] = kv[1]
				mustEquals(t, check, clues.InNamespace(ctx, "ns").Map(), false)
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
			mustEquals(t, check, clues.InNamespace(ctx, "ns").Map(), false)

			for _, m := range test.ms {
				ctx = clues.AddMapTo(ctx, "ns", m)
				for k, v := range m {
					check[k] = v
				}
				mustEquals(t, check, clues.InNamespace(ctx, "ns").Map(), false)
			}

			assert(
				t, ctx, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)
		})
	}
}

func TestAddTraceNameTo(t *testing.T) {
	table := []struct {
		name        string
		names       []string
		expectTrace string
		kvs         sa
		expectM     msa
		expectS     sa
	}{
		{"single", []string{"single"}, "single", nil, msa{}, sa{}},
		{"multiple", []string{"single", "multiple"}, "single,multiple", nil, msa{}, sa{}},
		{"duplicates", []string{"single", "multiple", "multiple"}, "single,multiple,multiple", nil, msa{}, sa{}},
		{"single with kvs", []string{"single"}, "single", sa{"k", "v"}, msa{"k": "v"}, sa{"k", "v"}},
		{"multiple with kvs", []string{"single", "multiple"}, "single,multiple", sa{"k", "v"}, msa{"k": "v"}, sa{"k", "v"}},
		{"duplicates with kvs", []string{"single", "multiple", "multiple"}, "single,multiple,multiple", sa{"k", "v"}, msa{"k": "v"}, sa{"k", "v"}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			mustEquals(t, msa{}, clues.InNamespace(ctx, "ns").Map(), false)

			for _, name := range test.names {
				ctx = clues.AddSpanTo(ctx, name, "ns", test.kvs...)
			}

			assert(
				t, ctx, "ns",
				msa{}, test.expectM,
				sa{}, test.expectS)

			c := clues.InNamespace(ctx, "ns").Map()
			if c["clues_trace"] != test.expectTrace {
				t.Errorf("expected clues_trace to equal %q, got %q", test.expectTrace, c["clues_trace"])
			}
		})
	}
}

func TestImmutableCtx(t *testing.T) {
	var (
		ctx     = context.Background()
		testCtx = context.WithValue(ctx, testCtx{}, "instance")
		check   = msa{}
		pre     = clues.In(testCtx)
		preMap  = pre.Map()
	)
	mustEquals(t, check, preMap, false)

	ctx2 := clues.Add(testCtx, "k", "v")
	if _, ok := preMap["k"]; ok {
		t.Errorf("previous map should not have been mutated by addition")
	}

	pre = clues.In(testCtx)
	if _, ok := preMap["k"]; ok {
		t.Errorf("previous map within ctx should not have been mutated by addition")
	}

	post := clues.In(ctx2).Map()
	if post["k"] != "v" {
		t.Errorf("new map should contain the added value")
	}

	var (
		l  = clues.Add(ctx, "foo", "bar")
		r  = clues.Add(ctx, "baz", "qux")
		ll = clues.Add(l, "fnords", "smarf")
		lr = clues.Add(l, "beaux", "regard")
	)

	mustEquals(t, msa{}, clues.In(ctx).Map(), false)
	mustEquals(t, msa{"foo": "bar"}, clues.In(l).Map(), false)
	mustEquals(t, msa{"baz": "qux"}, clues.In(r).Map(), false)
	mustEquals(t, msa{"foo": "bar", "fnords": "smarf"}, clues.In(ll).Map(), false)
	mustEquals(t, msa{"foo": "bar", "beaux": "regard"}, clues.In(lr).Map(), false)
}

var _ clues.Concealer = &safe{}

type safe struct {
	v any
}

func (s safe) PlainString() string            { return fmt.Sprintf("%v", s.v) }
func (s safe) Format(fs fmt.State, verb rune) { io.WriteString(fs, fmt.Sprintf("%"+string(verb), s.v)) }

func (s safe) Conceal() string {
	bs, err := json.Marshal(s.v)
	if err != nil {
		return "ERR MARSHALLING"
	}

	return string(bs)
}

var _ clues.Concealer = &custom{}

type custom struct {
	a, b string
}

func (c custom) PlainString() string            { return c.a + " - " + c.b }
func (c custom) Format(fs fmt.State, verb rune) { io.WriteString(fs, c.Conceal()) }

func (c custom) Conceal() string {
	return c.a + " - " + clues.ConcealWith(clues.SHA256, c.b)
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
		expectM    msa
		expectS    sa
	}{
		{
			name:       "all hidden",
			concealers: [][]any{{clues.Hide("k"), clues.Hide("v")}, {clues.Hide("not_k"), clues.Hide("not_v")}},
			expectM:    msa{"cc69e8e6a3b991d5": "f669b3b5927161b2", "ba3acd7f61e405ca": "509bf4fb69f55ca3"},
			expectS:    sa{"cc69e8e6a3b991d5", "f669b3b5927161b2", "ba3acd7f61e405ca", "509bf4fb69f55ca3"},
		},
		{
			name:       "partially hidden",
			concealers: [][]any{{clues.Hide("a"), safe{1}}, {clues.Hide(2), safe{"b"}}},
			expectM:    msa{"7d2ded59f6a549d7": "1", "cbdd96fab83ece85": `"b"`},
			expectS:    sa{"7d2ded59f6a549d7", "1", "cbdd96fab83ece85", `"b"`},
		},
		{
			name: "custom concealer",
			concealers: [][]any{
				{custom{"foo", "bar"}, custom{"baz", "qux"}},
				{custom{"fnords", "smarf"}, custom{"beau", "regard"}}},
			expectM: msa{"foo - fcde2b2edba56bf4": "baz - 21f58d27f827d295", "fnords - dd738d92a334bb85": "beau - fe099a0620ce9759"},
			expectS: sa{"foo - fcde2b2edba56bf4", "baz - 21f58d27f827d295", "fnords - dd738d92a334bb85", "beau - fe099a0620ce9759"},
		},
		{
			name:       "none",
			concealers: [][]any{},
			expectM:    msa{},
			expectS:    sa{},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), testCtx{}, "instance")
			check := msa{}
			mustEquals(t, check, toMSA(clues.In(ctx).Map()), false)

			for _, cs := range test.concealers {
				ctx = clues.Add(ctx, cs...)
				check[concealed(cs[0])] = concealed(cs[1])
				mustEquals(t, check, toMSA(clues.In(ctx).Map()), false)
			}

			assertMSA(
				t, ctx, "",
				test.expectM, msa{},
				test.expectS, sa{})
		})
	}
}

type pointable struct{}

func (p pointable) String() string {
	return "pointable"
}

func TestPointerDereferenceMarshal(t *testing.T) {
	var (
		p   *pointable
		ctx = context.Background()
	)

	// should not panic
	clues.Add(ctx, "pointable", p)
}

func TestAddComment(t *testing.T) {
	ctx := context.Background()
	dn := clues.In(ctx)

	comments := dn.Comments()
	if len(comments) > 0 {
		t.Errorf("no comments should have been added\n\tgot: %v", comments)
	}

	ctx = clues.AddComment(ctx, "first comment!")
	dn2 := clues.In(ctx)

	comments = dn2.Comments()
	if len(comments) != 1 {
		t.Errorf("should have found exactly one comment\n\tgot: %v", comments)
	}

	if comments[0].Message != "first comment!" {
		t.Errorf("unexpected comment:\n\twant: %s\n\tgot: %s", "first comment!", comments[0])
	}

	comments = dn.Comments()
	if len(comments) > 0 {
		t.Errorf("no comments should have been added to the original ctx\n\tgot: %v", comments)
	}

	ctx = clues.AddComment(ctx, "comment %d!", 2)
	dn3 := clues.In(ctx)

	comments = dn3.Comments()
	if len(comments) != 2 {
		t.Errorf("should have found exactly two comments\n\tgot: %v", comments)
	}

	if comments[0].Message != "first comment!" {
		t.Errorf("unexpected comment:\n\twant: %s\n\tgot: %s", "first comment!", comments[0])
	}

	if comments[1].Message != "comment 2!" {
		t.Errorf("unexpected comment:\n\twant: %s\n\tgot: %s", "comment 2!", comments[1])
	}

	comments = dn2.Comments()
	if len(comments) != 1 {
		t.Errorf("parent should have found exactly one comment\n\tgot: %v", comments)
	}

	if comments[0].Message != "first comment!" {
		t.Errorf("parent had unexpected comment:\n\twant: %s\n\tgot: %s", "first comment!", comments[0])
	}

	comments = dn.Comments()
	if len(comments) > 0 {
		t.Errorf("no comments should have been added to the original ctx\n\tgot: %v", comments)
	}
}

func addCommentToCtx(ctx context.Context, msg string) context.Context {
	return clues.AddComment(ctx, msg)
}

// requires sets of 3 strings
func commentRE(ss ...string) string {
	result := ""

	for i := 0; i < len(ss); i += 3 {
		result += ss[i] + " - "
		result += ss[i+1] + `:\d+\n`
		result += `\t` + ss[i+2]

		if len(ss) > i+3 {
			result += `\n`
		}
	}

	return result
}

func commentMatches(
	t *testing.T,
	expect, result string,
) {
	re := regexp.MustCompile(expect)

	if !re.MatchString(result) {
		t.Errorf(
			"unexpected comments stack"+
				"\n\nexpected (raw)\n\"%s\""+
				"\n\ngot (raw)\n%#v"+
				"\n\ngot (fmt)\n\"%s\"",
			re, result, result)
	}
}

func TestAddComment_trace(t *testing.T) {
	ctx := context.Background()
	ctx = clues.AddComment(ctx, "one")
	ctx = addCommentToCtx(ctx, "two")
	ctx = clues.AddComment(ctx, "three")

	dn := clues.In(ctx)
	comments := dn.Comments()
	stack := comments.String()
	expected := commentRE(
		"TestAddComment_trace", "clues/clues_test.go", "one",
		"addCommentToCtx", "clues/clues_test.go", "two",
		"TestAddComment_trace", "clues/clues_test.go", `three$`)

	commentMatches(t, expected, stack)
}

func TestAddAgent(t *testing.T) {
	ctx := context.Background()
	ctx = clues.Add(ctx, "one", 1)

	mapEquals(t, ctx, msa{
		"one": 1,
	}, false)

	ctxWithWit := clues.AddAgent(ctx, "wit")
	clues.Relay(ctx, "wit", "zero", 0)
	clues.Relay(ctxWithWit, "wit", "two", 2)

	mapEquals(t, ctx, msa{
		"one": 1,
	}, false)

	mapEquals(t, ctxWithWit, msa{
		"one": 1,
		"agents": map[string]map[string]any{
			"wit": {
				"two": 2,
			},
		},
	}, false)

	ctxWithTim := clues.AddAgent(ctxWithWit, "tim")
	clues.Relay(ctxWithTim, "tim", "three", 3)

	mapEquals(t, ctx, msa{
		"one": 1,
	}, false)

	mapEquals(t, ctxWithTim, msa{
		"one": 1,
		"agents": map[string]map[string]any{
			"wit": {
				"two": 2,
			},
			"tim": {
				"three": 3,
			},
		},
	}, false)

	ctxWithBob := clues.AddAgent(ctx, "bob")
	clues.Relay(ctxWithBob, "bob", "four", 4)

	mapEquals(t, ctx, msa{
		"one": 1,
	}, false)

	// should not have changed since its first usage
	mapEquals(t, ctxWithWit, msa{
		"one": 1,
		"agents": map[string]map[string]any{
			"wit": {
				"two": 2,
			},
		},
	}, false)

	// should not have changed since its first usage
	mapEquals(t, ctxWithTim, msa{
		"one": 1,
		"agents": map[string]map[string]any{
			"wit": {
				"two": 2,
			},
			"tim": {
				"three": 3,
			},
		},
	}, false)

	mapEquals(t, ctxWithBob, msa{
		"one": 1,
		"agents": map[string]map[string]any{
			"bob": {
				"four": 4,
			},
		},
	}, false)
}
