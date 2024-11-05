package tester

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/alcionai/clues/internal/node"
)

func MapEquals(
	t *testing.T,
	ctx context.Context,
	expect MSA,
	expectCluesTrace bool,
) {
	MustEquals(
		t,
		expect,
		node.FromCtx(ctx).Map(),
		expectCluesTrace)
}

func MustEquals[K comparable, V any](
	t *testing.T,
	expect, got map[K]V,
	hasCluesTrace bool,
) {
	e, g := ToMSS(expect), ToMSS(got)

	if len(g) > 0 {
		if _, ok := g["clues_trace"]; hasCluesTrace && !ok {
			t.Errorf(
				"expected map to contain key [clues_trace]\ngot: %+v",
				g)
		}
		delete(g, "clues_trace")
	}

	if len(g) > 0 {
		if _, ok := g["clues_trace"]; hasCluesTrace && !ok {
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

type MSS map[string]string

func ToMSS[K comparable, V any](m map[K]V) MSS {
	r := MSS{}

	for k, v := range m {
		ks := fmt.Sprintf("%v", k)
		vs := fmt.Sprintf("%v", v)
		r[ks] = vs
	}

	return r
}

type MSA map[string]any

func ToMSA[T any](m map[string]T) MSA {
	to := make(MSA, len(m))
	for k, v := range m {
		to[k] = v
	}

	return to
}

type SA []any

func (s SA) stringWith(other []any) string {
	return fmt.Sprintf(
		"\nexpected: %+v\nreceived: %+v\n",
		s, other,
	)
}

func (s SA) equals(t *testing.T, other []any) {
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

func AssertEq(
	t *testing.T,
	ctx context.Context,
	ns string,
	eM, eMns MSA,
	eS, eSns SA,
) {
	vs := node.FromCtx(ctx)
	MustEquals(t, eM, vs.Map(), false)
	eS.equals(t, vs.Slice())
}

func AssertMSA(
	t *testing.T,
	ctx context.Context,
	ns string,
	eM, eMns MSA,
	eS, eSns SA,
) {
	vs := node.FromCtx(ctx)
	MustEquals(t, eM, ToMSA(vs.Map()), false)
	eS.equals(t, vs.Slice())
}

type StubCtx struct{}
