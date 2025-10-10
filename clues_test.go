package clues_test

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/cecrets"
	"github.com/alcionai/clues/internal/tester"
)

func init() {
	cecrets.SetHasher(cecrets.HashCfg{
		HashAlg: cecrets.HMAC_SHA256,
		HMACKey: []byte("gobbledeygook-believe-it-or-not-this-is-randomly-generated"),
	})
}

func TestAdd(t *testing.T) {
	table := []struct {
		name    string
		kvs     [][]string
		expectM tester.MSA
		expectS tester.SA
	}{
		{
			"single",
			[][]string{{"k", "v"}},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"multiple",
			[][]string{{"a", "1"}, {"b", "2"}},
			tester.MSA{"a": "1", "b": "2"},
			tester.SA{"a", "1", "b", "2"},
		},
		{
			"duplicates",
			[][]string{{"a", "1"}, {"a", "2"}},
			tester.MSA{"a": "2"},
			tester.SA{"a", "2"},
		},
		{
			"none",
			[][]string{},
			tester.MSA{},
			tester.SA{},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), tester.StubCtx{}, "instance")
			check := tester.MSA{}
			tester.MustEquals(t, check, clues.In(ctx).Map(), false)

			for _, kv := range test.kvs {
				ctx = clues.Add(ctx, kv[0], kv[1])
				check[kv[0]] = kv[1]
				tester.MustEquals(t, check, clues.In(ctx).Map(), false)
			}

			tester.AssertEq(
				ctx, t, "",
				test.expectM, tester.MSA{},
				test.expectS, tester.SA{})
		})
	}
}

func TestAddMap(t *testing.T) {
	table := []struct {
		name    string
		ms      []tester.MSA
		expectM tester.MSA
		expectS tester.SA
	}{
		{
			"single",
			[]tester.MSA{{"k": "v"}},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"multiple",
			[]tester.MSA{{"a": "1"}, {"b": "2"}},
			tester.MSA{"a": "1", "b": "2"},
			tester.SA{"a", "1", "b", "2"},
		},
		{
			"duplicate",
			[]tester.MSA{{"a": "1"}, {"a": "2"}},
			tester.MSA{"a": "2"},
			tester.SA{"a", "2"},
		},
		{
			"none",
			[]tester.MSA{},
			tester.MSA{},
			tester.SA{},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), tester.StubCtx{}, "instance")
			check := tester.MSA{}
			tester.MustEquals(t, check, clues.In(ctx).Map(), false)

			for _, m := range test.ms {
				ctx = clues.AddMap(ctx, m)

				for k, v := range m {
					check[k] = v
				}

				tester.MustEquals(t, check, clues.In(ctx).Map(), false)
			}

			tester.AssertEq(
				ctx, t, "",
				test.expectM, tester.MSA{},
				test.expectS, tester.SA{},
			)
		})
	}
}

// TestAddSpan_Uninitialized ensures nothing panics if AddSpan is called and
// neither clues nor OTEL is initialized.
func TestAddSpan_Uninitialized(t *testing.T) {
	assert.NotPanics(
		t,
		func() {
			clues.AddSpan(t.Context(), "test span")
		},
	)
}

// TestAddSpan_Uninitialized_Concurrent ensures that even if OTEL isn't
// initialized there's no race condition when attempting to add spans to a
// parent context concurrently.
func TestAddSpan_Uninitialized_Concurrent(t *testing.T) {
	table := []struct {
		name  string
		attrs []any
	}{
		{
			name: "NoAttributes",
		},
		{
			name:  "Attributes",
			attrs: []any{"key", "value"},
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			var (
				wg sync.WaitGroup
				c  = make(chan struct{})
			)

			ctx := clues.AddSpan(t.Context(), "parent span", "some", "value")

			for range 5 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					<-c

					ctx := clues.AddSpan(ctx, "worker span", test.attrs...)
					defer clues.CloseSpan(ctx)
				}()
			}

			time.Sleep(500 * time.Millisecond)

			close(c)

			wg.Wait()
		})
	}
}

func TestAddSpan(t *testing.T) {
	table := []struct {
		name        string
		names       []string
		expectTrace string
		kvs         tester.SA
		expectM     tester.MSA
		expectS     tester.SA
	}{
		{
			"single",
			[]string{"single"},
			"single",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"multiple",
			[]string{"single", "multiple"},
			"single,multiple",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"duplicates",
			[]string{"single", "multiple", "multiple"},
			"single,multiple,multiple",
			nil,
			tester.MSA{},
			tester.SA{},
		},
		{
			"single with kvs",
			[]string{"single"},
			"single",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"multiple with kvs",
			[]string{"single", "multiple"},
			"single,multiple",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
		{
			"duplicates with kvs",
			[]string{"single", "multiple", "multiple"},
			"single,multiple,multiple",
			tester.SA{"k", "v"},
			tester.MSA{"k": "v"},
			tester.SA{"k", "v"},
		},
	}
	for _, test := range table {
		for _, init := range []bool{true, false} {
			tname := fmt.Sprintf("%s-%v", test.name, init)

			t.Run(tname, func(t *testing.T) {
				ctx := context.Background()

				if init {
					ocfg := clues.OTELConfig{GRPCEndpoint: "localhost:4317"}

					ictx, err := clues.InitializeOTEL(ctx, test.name, ocfg)
					require.NoError(t, err, "initializing otel")

					if err != nil {
						return
					}

					//nolint:lll
					// FIXME: this is causing failures at the moment which are non-trivial to
					// hack around.  Will need to return to it for more complete otel/grpc testing.
					// suggestion: https://github.com/pellared/opentelemetry-go-contrib/blob/8f8e9b60693177b91af45d0495289fc52aa5c50e/instrumentation/google.golang.org/grpc/otelgrpc/test/grpc_test.go#L88
					// defer func() {
					// 	err := clues.Close(ictx)
					// 	require.NoError(t, err, "closing clues")
					// 	if err != nil {
					// 		return
					// 	}
					// }()

					ctx = ictx
				}

				ctx = context.WithValue(ctx, tester.StubCtx{}, "instance")
				tester.MustEquals(t, tester.MSA{}, clues.In(ctx).Map(), false)

				for _, name := range test.names {
					ctx = clues.AddSpan(ctx, name, test.kvs...)
					defer clues.CloseSpan(ctx)
				}

				tester.AssertEq(
					ctx, t, "",
					test.expectM, tester.MSA{},
					test.expectS, tester.SA{},
				)

				c := clues.In(ctx).Map()
				if c["clues_trace"] != test.expectTrace {
					t.Errorf(
						"expected clues_trace to equal %q, got %q",
						test.expectTrace,
						c["clues_trace"],
					)
				}
			})
		}
	}
}

func TestImmutableCtx(t *testing.T) {
	var (
		ctx     = context.Background()
		testCtx = context.WithValue(ctx, tester.StubCtx{}, "instance")
		check   = tester.MSA{}
		pre     = clues.In(testCtx)
		preMap  = pre.Map()
	)

	tester.MustEquals(t, check, preMap, false)

	ctx2 := clues.Add(testCtx, "k", "v")

	if _, ok := preMap["k"]; ok {
		t.Errorf("previous map should not have been mutated by addition")
	}

	pre = clues.In(testCtx)
	if _, ok := pre.Map()["k"]; ok {
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

	tester.MustEquals(
		t,
		tester.MSA{},
		clues.In(ctx).Map(),
		false,
	)
	tester.MustEquals(
		t,
		tester.MSA{"foo": "bar"},
		clues.In(l).Map(),
		false,
	)
	tester.MustEquals(
		t,
		tester.MSA{"baz": "qux"},
		clues.In(r).Map(),
		false,
	)
	tester.MustEquals(
		t,
		tester.MSA{"foo": "bar", "fnords": "smarf"},
		clues.In(ll).Map(),
		false,
	)
	tester.MustEquals(
		t,
		tester.MSA{"foo": "bar", "beaux": "regard"},
		clues.In(lr).Map(),
		false,
	)
}

var _ cecrets.Concealer = &safe{}

type safe struct {
	v any
}

func (s safe) PlainString() string            { return fmt.Sprintf("%v", s.v) }
func (s safe) Format(fs fmt.State, verb rune) { fmt.Fprintf(fs, "%"+string(verb), s.v) }

func (s safe) Conceal() string {
	bs, err := json.Marshal(s.v)
	if err != nil {
		return "ERR MARSHALLING"
	}

	return string(bs)
}

var _ cecrets.Concealer = &custom{}

type custom struct {
	a, b string
}

func (c custom) PlainString() string            { return c.a + " - " + c.b }
func (c custom) Format(fs fmt.State, verb rune) { fmt.Fprint(fs, c.Conceal()) }

func (c custom) Conceal() string {
	return c.a + " - " + cecrets.ConcealWith(cecrets.SHA256, c.b)
}

func concealed(a any) string {
	c, ok := a.(cecrets.Concealer)
	if !ok {
		return "NOT CONCEALER"
	}

	return c.Conceal()
}

func TestAdd_concealed(t *testing.T) {
	// if not set here, test reordering in golang can cause
	// cross contamination when other tests ste the global
	// cecrets handler.
	cecrets.SetHasher(cecrets.HashCfg{
		HashAlg: cecrets.HMAC_SHA256,
		HMACKey: []byte("gobbledeygook-believe-it-or-not-this-is-randomly-generated"),
	})

	table := []struct {
		name       string
		concealers [][]any
		expectM    tester.MSA
		expectS    tester.SA
	}{
		{
			name: "all hidden",
			concealers: [][]any{
				{cecrets.Hide("k"), cecrets.Hide("v")},
				{cecrets.Hide("not_k"), cecrets.Hide("not_v")},
			},
			expectM: tester.MSA{
				"ba3acd7f61e405ca": "509bf4fb69f55ca3",
				"cc69e8e6a3b991d5": "f669b3b5927161b2",
			},
			expectS: tester.SA{
				"ba3acd7f61e405ca", "509bf4fb69f55ca3",
				"cc69e8e6a3b991d5", "f669b3b5927161b2",
			},
		},
		{
			name:       "partially hidden",
			concealers: [][]any{{cecrets.Hide("a"), safe{1}}, {cecrets.Hide(2), safe{"b"}}},
			expectM: tester.MSA{
				"7d2ded59f6a549d7": "1",
				"cbdd96fab83ece85": `"b"`,
			},
			expectS: tester.SA{
				"7d2ded59f6a549d7", "1",
				"cbdd96fab83ece85", `"b"`,
			},
		},
		{
			name: "custom concealer",
			concealers: [][]any{
				{custom{"foo", "bar"}, custom{"baz", "qux"}},
				{custom{"fnords", "smarf"}, custom{"beau", "regard"}},
			},
			expectM: tester.MSA{
				"foo - fcde2b2edba56bf4":    "baz - 21f58d27f827d295",
				"fnords - dd738d92a334bb85": "beau - fe099a0620ce9759",
			},
			expectS: tester.SA{
				"foo - fcde2b2edba56bf4", "baz - 21f58d27f827d295",
				"fnords - dd738d92a334bb85", "beau - fe099a0620ce9759",
			},
		},
		{
			name:       "none",
			concealers: [][]any{},
			expectM:    tester.MSA{},
			expectS:    tester.SA{},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.WithValue(t.Context(), tester.StubCtx{}, "instance")

			check := tester.MSA{}
			tester.MustEquals(t, check, tester.ToMSA(clues.In(ctx).Map()), false)

			for _, cs := range test.concealers {
				ctx = clues.Add(ctx, cs...)
				check[concealed(cs[0])] = concealed(cs[1])
				tester.MustEquals(t, check, tester.ToMSA(clues.In(ctx).Map()), false)
			}

			tester.AssertMSA(
				ctx, t, "",
				test.expectM, tester.MSA{},
				test.expectS, tester.SA{})
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
		t.Errorf(
			"parent had unexpected comment:\n\twant: %s\n\tgot: %s",
			"first comment!",
			comments[0],
		)
	}

	comments = dn.Comments()
	if len(comments) > 0 {
		t.Errorf("no comments should have been added to the original ctx\n\tgot: %v", comments)
	}
}

func addCommentToCtx(
	ctx context.Context,
	msg string,
	args ...any,
) context.Context {
	return clues.AddComment(ctx, msg, args...)
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

	tester.MapEquals(ctx, t, tester.MSA{
		"one": 1,
	}, false)

	ctxWithWit := clues.AddAgent(ctx, "wit")
	clues.Relay(ctx, "wit", "zero", 0)
	clues.Relay(ctxWithWit, "wit", "two", 2)

	tester.MapEquals(ctx, t, tester.MSA{
		"one": 1,
	}, false)

	tester.MapEquals(ctxWithWit, t, tester.MSA{
		"one": 1,
		"agents": map[string]map[string]any{
			"wit": {
				"two": 2,
			},
		},
	}, false)

	ctxWithTim := clues.AddAgent(ctxWithWit, "tim")
	clues.Relay(ctxWithTim, "tim", "three", 3)

	tester.MapEquals(ctx, t, tester.MSA{
		"one": 1,
	}, false)

	tester.MapEquals(ctxWithTim, t, tester.MSA{
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

	tester.MapEquals(ctx, t, tester.MSA{
		"one": 1,
	}, false)

	// should not have changed since its first usage
	tester.MapEquals(ctxWithWit, t, tester.MSA{
		"one": 1,
		"agents": map[string]map[string]any{
			"wit": {
				"two": 2,
			},
		},
	}, false)

	// should not have changed since its first usage
	tester.MapEquals(ctxWithTim, t, tester.MSA{
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

	tester.MapEquals(ctxWithBob, t, tester.MSA{
		"one": 1,
		"agents": map[string]map[string]any{
			"bob": {
				"four": 4,
			},
		},
	}, false)
}
