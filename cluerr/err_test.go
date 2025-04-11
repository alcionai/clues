package cluerr_test

import (
	"context"
	"fmt"
	"maps"
	"testing"

	"github.com/pkg/errors"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/tester"
)

type msa map[string]any

func toMSA[T any](m map[string]T) msa {
	to := make(msa, len(m))
	for k, v := range m {
		to[k] = v
	}

	return to
}

type testingError struct{}

func (e testingError) Error() string {
	return "an error"
}

type testingErrorIface interface {
	error
}

func TestStack(t *testing.T) {
	table := []struct {
		name      string
		getErr    func() error
		expectNil bool
	}{
		{
			name: "SingleNil",
			getErr: func() error {
				return cluerr.Stack(nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "DoubleNil",
			getErr: func() error {
				return cluerr.Stack(nil, nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "TripleNil",
			getErr: func() error {
				return cluerr.Stack(nil, nil, nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "StackNilNil",
			getErr: func() error {
				return cluerr.Stack(cluerr.Stack(nil), nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "NilStackNilNil",
			getErr: func() error {
				return cluerr.Stack(nil, cluerr.Stack(nil), nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "NilInterfaceError",
			getErr: func() error {
				var e testingErrorIface

				return cluerr.Stack(nil, e, cluerr.Stack(nil)).OrNil()
			},
			expectNil: true,
		},
		{
			name: "NonNilNonPointerInterfaceError",
			getErr: func() error {
				var e testingErrorIface = testingError{}

				return cluerr.Stack(nil, e, cluerr.Stack(nil)).OrNil()
			},
			expectNil: false,
		},
		{
			name: "NonNilNonPointerInterfaceError",
			getErr: func() error {
				var e testingErrorIface = &testingError{}

				return cluerr.Stack(nil, e, cluerr.Stack(nil)).OrNil()
			},
			expectNil: false,
		},
		{
			name: "NonPointerError",
			getErr: func() error {
				return cluerr.Stack(nil, testingError{}, cluerr.Stack(nil)).OrNil()
			},
			expectNil: false,
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := test.getErr()

			if test.expectNil && err != nil {
				t.Errorf("expected nil error but got: %+v\n", err)
			} else if !test.expectNil && err == nil {
				t.Error("expected non-nil error but got nil")
			}
		})
	}
}

func TestHasLabel(t *testing.T) {
	const label = "some-label"

	table := []struct {
		name    string
		initial error
	}{
		{
			name: "multiple stacked clues errors with label on first",
			initial: cluerr.Stack(
				cluerr.New("Labeled").Label(label),
				cluerr.New("NotLabeled")),
		},
		{
			name: "multiple stacked clues errors with label on second",
			initial: cluerr.Stack(
				cluerr.New("NotLabeled"),
				cluerr.New("Labeled").Label(label)),
		},
		{
			name:    "single stacked clues error with label",
			initial: cluerr.Stack(cluerr.New("Labeled").Label(label)),
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			if !cluerr.HasLabel(test.initial, label) {
				t.Errorf(
					"expected error to have label [%s] but got %v",
					label,
					maps.Keys(cluerr.Labels(test.initial)))
			}
		})
	}
}

func TestLabel(t *testing.T) {
	table := []struct {
		name    string
		initial error
		expect  func(*testing.T, *cluerr.Err)
	}{
		{"nil", nil, nil},
		{"standard error", errors.New("an error"), nil},
		{"clues error", cluerr.New("clues error"), nil},
		{"clues error wrapped", fmt.Errorf("%w", cluerr.New("clues error")), nil},
		{
			"clues error with label",
			cluerr.New("clues error").Label("fnords"),
			func(t *testing.T, err *cluerr.Err) {
				if !err.HasLabel("fnords") {
					t.Error("expected error to have label [fnords]")
				}
			},
		},
		{
			"clues error with label wrapped",
			fmt.Errorf("%w", cluerr.New("clues error").Label("fnords")),
			func(t *testing.T, err *cluerr.Err) {
				if !err.HasLabel("fnords") {
					t.Error("expected error to have label [fnords]")
				}
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			if cluerr.HasLabel(test.initial, "foo") {
				t.Error("new error should have no label")
			}

			err := cluerr.Label(test.initial, "foo")
			if !cluerr.HasLabel(err, "foo") && test.initial != nil {
				t.Error("expected error to have label [foo]")
			}

			if err == nil {
				if test.initial != nil {
					t.Error("error should not be nil after labeling")
				}

				return
			}

			ref := err.Label("bar")

			if !cluerr.HasLabel(err, "bar") {
				t.Error("expected error to have label [bar]")
			}

			if !cluerr.HasLabel(ref, "bar") {
				t.Error("expected error to have label [bar]")
			}

			if test.expect != nil {
				test.expect(t, err)
			}
		})
	}
}

func TestLabels(t *testing.T) {
	var (
		ma    = msa{"a": struct{}{}}
		mab   = msa{"a": struct{}{}, "b": struct{}{}}
		a     = cluerr.New("a").Label("a")
		acopy = cluerr.New("acopy").Label("a")
		b     = cluerr.New("b").Label("b")
		wrap  = cluerr.Wrap(
			cluerr.Stack(
				fmt.Errorf("%w", a),
				fmt.Errorf("%w", b),
				fmt.Errorf("%w", acopy),
			), "wa")
	)

	table := []struct {
		name    string
		initial error
		expect  msa
	}{
		{"nil", nil, msa{}},
		{"standard error", errors.New("an error"), msa{}},
		{"unlabeled error", cluerr.New("clues error"), msa{}},
		{"pkg/errs wrap around labeled error", errors.Wrap(a, "wa"), ma},
		{"clues wrapped", cluerr.Wrap(a, "wrap"), ma},
		{"clues stacked", cluerr.Stack(a, b), mab},
		{"clues stacked with copy", cluerr.Stack(a, b, acopy), mab},
		{"error chain", cluerr.Stack(b, fmt.Errorf("%w", a), fmt.Errorf("%w", acopy)), mab},
		{"error wrap chain", wrap, mab},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := cluerr.Labels(test.initial)
			tester.MustEquals(t, test.expect, toMSA(result), false)
		})
	}
}

var (
	errBase                   = errors.New("an error")
	errCluesStackedBase       = func() error { return cluerr.Stack(errBase) }
	errFmtWrappedCluesWrapped = func() error {
		return fmt.Errorf("%w", cluerr.Wrap(errBase, "wrapped error with vals").With("z", 0))
	}
	errCluesStackedCluesNew = func() error {
		return cluerr.Stack(cluerr.New("primary").With("z", 0), errors.New("secondary"))
	}
)

func TestWith(t *testing.T) {
	table := []struct {
		name    string
		initial error
		k, v    string
		with    [][]any
		expect  msa
	}{
		{
			"nil error",
			nil,
			"k",
			"v",
			[][]any{{"k2", "v2"}},
			msa{},
		},
		{
			"only base error vals",
			errBase,
			"k",
			"v",
			nil,
			msa{"k": "v"},
		},
		{
			"empty base error vals",
			errBase,
			"",
			"",
			nil,
			msa{"": ""},
		},
		{
			"standard",
			errBase,
			"k",
			"v",
			[][]any{{"k2", "v2"}},
			msa{"k": "v", "k2": "v2"},
		},
		{
			"duplicates",
			errBase,
			"k",
			"v",
			[][]any{{"k", "v2"}},
			msa{"k": "v2"},
		},
		{
			"multi",
			errBase,
			"a",
			"1",
			[][]any{{"b", "2"}, {"c", "3"}},
			msa{"a": "1", "b": "2", "c": "3"},
		},
		{
			"only clue error vals",
			errCluesStackedBase(),
			"k",
			"v",
			nil,
			msa{"k": "v"},
		},
		{
			"empty clue error vals",
			errCluesStackedBase(),
			"",
			"",
			nil,
			msa{"": ""},
		},
		{
			"standard cerr",
			errCluesStackedBase(),
			"k",
			"v",
			[][]any{{"k2", "v2"}},
			msa{"k": "v", "k2": "v2"},
		},
		{
			"duplicates cerr",
			errCluesStackedBase(),
			"k",
			"v",
			[][]any{{"k", "v2"}},
			msa{"k": "v2"},
		},
		{
			"multi cerr",
			errCluesStackedBase(),
			"a",
			"1",
			[][]any{{"b", "2"}, {"c", "3"}},
			msa{"a": "1", "b": "2", "c": "3"},
		},
		{
			"only wrapped error vals",
			errFmtWrappedCluesWrapped(),
			"k",
			"v",
			nil,
			msa{"k": "v", "z": 0},
		},
		{
			"empty wrapped error vals",
			errFmtWrappedCluesWrapped(),
			"",
			"",
			nil,
			msa{"": "", "z": 0},
		},
		{
			"standard wrapped",
			errFmtWrappedCluesWrapped(),
			"k",
			"v",
			[][]any{{"k2", "v2"}},
			msa{"k": "v", "k2": "v2", "z": 0},
		},
		{
			"duplicates wrapped",
			errFmtWrappedCluesWrapped(),
			"k",
			"v",
			[][]any{{"k", "v2"}},
			msa{"k": "v2", "z": 0},
		},
		{
			"multi wrapped",
			errFmtWrappedCluesWrapped(),
			"a",
			"1",
			[][]any{{"b", "2"}, {"c", "3"}},
			msa{"a": "1", "b": "2", "c": "3", "z": 0},
		},
		{
			"only stacked error vals",
			errCluesStackedCluesNew(),
			"k",
			"v",
			nil,
			msa{"k": "v", "z": 0},
		},
		{
			"empty stacked error vals",
			errCluesStackedCluesNew(),
			"",
			"",
			nil,
			msa{"": "", "z": 0},
		},
		{
			"standard stacked",
			errCluesStackedCluesNew(),
			"k",
			"v",
			[][]any{{"k2", "v2"}},
			msa{"k": "v", "k2": "v2", "z": 0},
		},
		{
			"duplicates stacked",
			errCluesStackedCluesNew(),
			"k",
			"v",
			[][]any{{"k", "v2"}},
			msa{"k": "v2", "z": 0},
		},
		{
			"multi stacked",
			errCluesStackedCluesNew(),
			"a",
			"1",
			[][]any{{"b", "2"}, {"c", "3"}},
			msa{"a": "1", "b": "2", "c": "3", "z": 0},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := cluerr.Stack(test.initial).With(test.k, test.v)

			for _, kv := range test.with {
				err = err.With(kv...)
			}

			tester.MustEquals(t, test.expect, cluerr.CluesIn(err).Map(), false)
			tester.MustEquals(t, test.expect, err.Values().Map(), false)
		})
	}
}

func TestWithMap(t *testing.T) {
	table := []struct {
		name    string
		initial error
		kv      msa
		with    msa
		expect  msa
	}{
		{
			"nil error",
			nil,
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{},
		},
		{
			"only base error vals",
			errBase,
			msa{"k": "v"},
			nil,
			msa{"k": "v"},
		},
		{
			"empty base error vals",
			errBase,
			msa{"": ""},
			nil,
			msa{"": ""},
		},
		{
			"standard",
			errBase,
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2"},
		},
		{
			"duplicates",
			errBase,
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2"},
		},
		{
			"multi",
			errBase,
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3"},
		},
		{
			"only clue error vals",
			errCluesStackedBase(),
			msa{"k": "v"},
			nil,
			msa{"k": "v"},
		},
		{
			"empty clue error vals",
			errCluesStackedBase(),
			msa{"": ""},
			nil,
			msa{"": ""},
		},
		{
			"standard cerr",
			errCluesStackedBase(),
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2"},
		},
		{
			"duplicates cerr",
			errCluesStackedBase(),
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2"},
		},
		{
			"multi cerr",
			errCluesStackedBase(),
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3"},
		},
		{
			"only wrapped error vals",
			errFmtWrappedCluesWrapped(),
			msa{"k": "v"},
			nil,
			msa{"k": "v", "z": 0},
		},
		{
			"empty wrapped error vals",
			errFmtWrappedCluesWrapped(),
			msa{"": ""},
			nil,
			msa{"": "", "z": 0},
		},
		{
			"standard wrapped",
			errFmtWrappedCluesWrapped(),
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2", "z": 0},
		},
		{
			"duplicates wrapped",
			errFmtWrappedCluesWrapped(),
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2", "z": 0},
		},
		{
			"multi wrapped",
			errFmtWrappedCluesWrapped(),
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3", "z": 0},
		},
		{
			"only stacked error vals",
			errCluesStackedCluesNew(),
			msa{"k": "v"},
			nil,
			msa{"k": "v", "z": 0},
		},
		{
			"empty stacked error vals",
			errCluesStackedCluesNew(),
			msa{"": ""},
			nil,
			msa{"": "", "z": 0},
		},
		{
			"standard stacked",
			errCluesStackedCluesNew(),
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2", "z": 0},
		},
		{
			"duplicates stacked",
			errCluesStackedCluesNew(),
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2", "z": 0},
		},
		{
			"multi stacked",
			errCluesStackedCluesNew(),
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3", "z": 0},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := cluerr.Stack(test.initial).WithMap(test.kv)
			err = err.WithMap(test.with)
			tester.MustEquals(t, test.expect, cluerr.CluesIn(err).Map(), false)
			tester.MustEquals(t, test.expect, err.Values().Map(), false)
		})
	}
}

func TestWithClues(t *testing.T) {
	ctx := context.Background()

	table := []struct {
		name    string
		initial error
		kv      msa
		with    msa
		expect  msa
	}{
		{
			"nil error",
			nil,
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{},
		},
		{
			"only base error vals",
			errBase,
			msa{"k": "v"},
			nil,
			msa{"k": "v"},
		},
		{
			"empty base error vals",
			errBase,
			msa{"": ""},
			nil,
			msa{"": ""},
		},
		{
			"standard",
			errBase,
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2"},
		},
		{
			"duplicates",
			errBase,
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2"},
		},
		{
			"multi",
			errBase,
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3"},
		},
		{
			"only clue error vals",
			errCluesStackedBase(),
			msa{"k": "v"},
			nil,
			msa{"k": "v"},
		},
		{
			"empty clue error vals",
			errCluesStackedBase(),
			msa{"": ""},
			nil,
			msa{"": ""},
		},
		{
			"standard cerr",
			errCluesStackedBase(),
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2"},
		},
		{
			"duplicates cerr",
			errCluesStackedBase(),
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2"},
		},
		{
			"multi cerr",
			errCluesStackedBase(),
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3"},
		},
		{
			"only wrapped error vals",
			errFmtWrappedCluesWrapped(),
			msa{"k": "v"},
			nil,
			msa{"k": "v", "z": 0},
		},
		{
			"empty wrapped error vals",
			errFmtWrappedCluesWrapped(),
			msa{"": ""},
			nil,
			msa{"": "", "z": 0},
		},
		{
			"standard wrapped",
			errFmtWrappedCluesWrapped(),
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2", "z": 0},
		},
		{
			"duplicates wrapped",
			errFmtWrappedCluesWrapped(),
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2", "z": 0},
		},
		{
			"multi wrapped",
			errFmtWrappedCluesWrapped(),
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3", "z": 0},
		},
		{
			"only stacked error vals",
			errCluesStackedCluesNew(),
			msa{"k": "v"},
			nil,
			msa{"k": "v", "z": 0},
		},
		{
			"empty stacked error vals",
			errCluesStackedCluesNew(),
			msa{"": ""},
			nil,
			msa{"": "", "z": 0},
		},
		{
			"standard stacked",
			errCluesStackedCluesNew(),
			msa{"k": "v"},
			msa{"k2": "v2"},
			msa{"k": "v", "k2": "v2", "z": 0},
		},
		{
			"duplicates stacked",
			errCluesStackedCluesNew(),
			msa{"k": "v"},
			msa{"k": "v2"},
			msa{"k": "v2", "z": 0},
		},
		{
			"multi stacked",
			errCluesStackedCluesNew(),
			msa{"a": "1"},
			msa{"b": "2", "c": "3"},
			msa{"a": "1", "b": "2", "c": "3", "z": 0},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			tctx := clues.AddMap(ctx, test.kv)
			err := cluerr.Stack(test.initial).WithClues(tctx)
			err = err.WithMap(test.with)
			tester.MustEquals(t, test.expect, cluerr.CluesIn(err).Map(), false)
			tester.MustEquals(t, test.expect, err.Values().Map(), false)
		})
	}
}

func TestValuePriority(t *testing.T) {
	table := []struct {
		name   string
		err    error
		expect msa
	}{
		{
			name: "lowest data wins",
			err: func() error {
				ctx := clues.Add(context.Background(), "in-ctx", 1)
				// the last addition to a ctx should take priority
				ctx = clues.Add(ctx, "in-ctx", 2)

				err := cluerr.NewWC(ctx, "err").With("in-err", 1)
				// the first addition to an error should take priority
				err = cluerr.StackWC(ctx, err).With("in-err", 2)

				return err
			}(),
			expect: msa{"in-ctx": 2, "in-err": 1},
		},
		{
			name: "last stack wins",
			err: func() error {
				ctx := clues.Add(context.Background(), "in-ctx", 1)
				err := cluerr.NewWC(ctx, "last in stack").With("in-err", 1)
				err = cluerr.Stack(
					cluerr.New("first in stack").With("in-err", 2),
					err)
				return err
			}(),
			expect: msa{"in-ctx": 1, "in-err": 1},
		},
		{
			name: ".With wins over ctx",
			err: func() error {
				ctx := clues.Add(context.Background(), "k", 1)
				err := cluerr.NewWC(ctx, "last in stack").With("k", 2)
				return err
			}(),
			expect: msa{"k": 2},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			tester.MustEquals(t, test.expect, cluerr.CluesIn(test.err).Map(), false)
		})
	}
}

func TestUnwrap(t *testing.T) {
	e := errors.New("cause")
	we := cluerr.Wrap(e, "outer")

	ce := we.Unwrap()
	if ce != e {
		t.Errorf("expected result error [%v] to be base error [%v]\n", ce, e)
	}

	se := cluerr.Stack(e)

	ce = se.Unwrap()
	if ce != e {
		t.Errorf("expected result error [%v] to be base error [%v]\n", ce, e)
	}
}

func TestWrapNilStackSlice(t *testing.T) {
	// an empty slice of errors
	sl := make([]error, 10)
	// when stacked
	st := cluerr.Stack(sl...)
	// then wrapped
	e := cluerr.Wrap(st, "wrapped")
	// should contain nil
	if e.OrNil() != nil {
		t.Errorf("e.OrNil() <%+v> should be nil", e.OrNil())
	}
}

func TestErr_Error(t *testing.T) {
	sentinel := errors.New("sentinel")

	table := []struct {
		name   string
		err    error
		expect string
	}{
		{
			name:   "new error",
			err:    cluerr.New("new"),
			expect: "new",
		},
		{
			name:   "stacked error",
			err:    cluerr.Stack(sentinel),
			expect: sentinel.Error(),
		},
		{
			name:   "wrapped new error",
			err:    cluerr.Wrap(cluerr.New("new"), "wrap"),
			expect: "wrap: new",
		},
		{
			name:   "wrapped non-clues error",
			err:    cluerr.Wrap(sentinel, "wrap"),
			expect: "wrap: " + sentinel.Error(),
		},
		{
			name:   "wrapped stacked error",
			err:    cluerr.Wrap(cluerr.Stack(sentinel), "wrap"),
			expect: "wrap: " + sentinel.Error(),
		},
		{
			name:   "multiple wraps",
			err:    cluerr.Wrap(cluerr.Wrap(cluerr.New("new"), "wrap"), "wrap2"),
			expect: "wrap2: wrap: new",
		},
		{
			name:   "wrap-stack-wrap-new",
			err:    cluerr.Wrap(cluerr.Stack(cluerr.Wrap(cluerr.New("new"), "wrap")), "wrap2"),
			expect: "wrap2: wrap: new",
		},
		{
			name:   "many stacked errors",
			err:    cluerr.Stack(sentinel, errors.New("middle"), errors.New("base")),
			expect: sentinel.Error() + ": middle: base",
		},
		{
			name: "stacked stacks",
			err: cluerr.Stack(
				cluerr.Stack(sentinel, errors.New("left")),
				cluerr.Stack(errors.New("right"), errors.New("base")),
			),
			expect: sentinel.Error() + ": left: right: base",
		},
		{
			name: "wrapped stacks",
			err: cluerr.Stack(
				cluerr.Wrap(cluerr.Stack(errors.New("top"), errors.New("left")), "left-stack"),
				cluerr.Wrap(cluerr.Stack(errors.New("right"), errors.New("base")), "right-stack"),
			),
			expect: "left-stack: top: left: right-stack: right: base",
		},
		{
			name: "wrapped stacks, all cluerr.New",
			err: cluerr.Stack(
				cluerr.Wrap(cluerr.Stack(cluerr.New("top"), cluerr.New("left")), "left-stack"),
				cluerr.Wrap(cluerr.Stack(cluerr.New("right"), cluerr.New("base")), "right-stack"),
			),
			expect: "left-stack: top: left: right-stack: right: base",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := test.err.Error()
			if result != test.expect {
				t.Errorf("expected error message [%s], got [%s]", test.expect, result)
			}
		})
	}
}

func TestErrValues_stacks(t *testing.T) {
	table := []struct {
		name   string
		err    error
		expect msa
	}{
		{
			name:   "single err",
			err:    cluerr.Stack(cluerr.New("an err").With("k", "v")),
			expect: msa{"k": "v"},
		},
		{
			name: "two stack",
			err: cluerr.Stack(
				cluerr.New("an err").With("k", "v"),
				cluerr.New("other").With("k2", "v2"),
			),
			expect: msa{"k": "v", "k2": "v2"},
		},
		{
			name: "sandvitch",
			err: cluerr.Stack(
				cluerr.New("top").With("k", "v"),
				errors.New("mid"),
				cluerr.New("base").With("k2", "v2"),
			),
			expect: msa{"k": "v", "k2": "v2"},
		},
		{
			name: "value collision",
			err: cluerr.Stack(
				cluerr.New("top").With("k", "v"),
				cluerr.New("mid").With("k2", "v2"),
				cluerr.New("base").With("k", "v3"),
			),
			expect: msa{"k": "v3", "k2": "v2"},
		},
		{
			name: "double double",
			err: cluerr.Stack(
				cluerr.Stack(
					cluerr.New("top").With("k", "v"),
					cluerr.New("left").With("k2", "v2"),
				),
				cluerr.Stack(
					cluerr.New("right").With("k3", "v3"),
					cluerr.New("base").With("k4", "v4"),
				),
			),
			expect: msa{
				"k":  "v",
				"k2": "v2",
				"k3": "v3",
				"k4": "v4",
			},
		},
		{
			name: "double double collision",
			err: cluerr.Stack(
				cluerr.Stack(
					cluerr.New("top").With("k", "v"),
					cluerr.New("left").With("k2", "v2"),
				),
				cluerr.Stack(
					cluerr.New("right").With("k3", "v3"),
					cluerr.New("base").With("k", "v4"),
				),
			),
			expect: msa{
				"k":  "v4",
				"k2": "v2",
				"k3": "v3",
			},
		},
		{
			name: "double double animal wrap",
			err: cluerr.Stack(
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("top").With("k", "v"),
						cluerr.New("left").With("k2", "v2"),
					),
					"left-stack"),
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("right").With("k3", "v3"),
						cluerr.New("base").With("k4", "v4"),
					),
					"right-stack"),
			),
			expect: msa{
				"k":  "v",
				"k2": "v2",
				"k3": "v3",
				"k4": "v4",
			},
		},
		{
			name: "double double animal wrap collision",
			err: cluerr.Stack(
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("top").With("k", "v"),
						cluerr.New("left").With("k2", "v2"),
					),
					"left-stack"),
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("right").With("k3", "v3"),
						cluerr.New("base").With("k", "v4"),
					),
					"right-stack"),
			),
			expect: msa{
				"k":  "v4",
				"k2": "v2",
				"k3": "v3",
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			vs := cluerr.CluesIn(test.err)
			tester.MustEquals(t, test.expect, vs.Map(), false)
		})
	}
}

func TestImmutableErrors(t *testing.T) {
	err := cluerr.New("an error").With("k", "v")
	check := msa{"k": "v"}
	pre := cluerr.CluesIn(err)

	tester.MustEquals(t, check, pre.Map(), false)

	err2 := err.With("k2", "v2")

	if _, ok := pre.Map()["k2"]; ok {
		t.Errorf("previous map should not have been mutated by addition")
	}

	post := cluerr.CluesIn(err2)
	if post.Map()["k2"] != "v2" {
		t.Errorf("new map should contain the added value")
	}
}

type mockTarget struct {
	err error
}

func (mt mockTarget) Error() string {
	return mt.err.Error()
}

func (mt mockTarget) Cause() error {
	return mt.err
}

func (mt mockTarget) Unwrap() error {
	return mt.err
}

const (
	lt   = "left-top"
	lb   = "left-base"
	rt   = "right-top"
	rb   = "right-base"
	stnl = "sentinel"
	tgt  = "target"
)

//nolint:revive
var (
	target    = mockTarget{errors.New(tgt)}
	sentinel  = errors.New(stnl)
	other     = errors.New("other")
	leftTop   = cluerr.New(lt).With(lt, "v"+lt).Label(lt)
	leftBase  = cluerr.New(lb).With(lb, "v"+lb).Label(lb)
	rightTop  = cluerr.New(rt).With(rt, "v"+rt).Label(rt)
	rightBase = cluerr.New(rb).With(rb, "v"+rb).Label(rb)
)

var testTable = []struct {
	name         string
	err          error
	expectMsg    string
	expectLabels msa
	expectValues msa
}{
	{
		name:         "plain stack",
		err:          cluerr.Stack(target, sentinel),
		expectMsg:    "target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "plain wrap",
		err:          cluerr.Wrap(cluerr.Stack(target, sentinel), "wrap"),
		expectLabels: msa{},
		expectMsg:    "wrap: target: sentinel",
		expectValues: msa{},
	},
	{
		name:         "two stack; top",
		err:          cluerr.Stack(cluerr.Stack(target, sentinel), other),
		expectMsg:    "target: sentinel: other",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "two stack; base",
		err:          cluerr.Stack(other, cluerr.Stack(target, sentinel)),
		expectMsg:    "other: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name: "two wrap",
		err: cluerr.Wrap(
			cluerr.Wrap(
				cluerr.Stack(target, sentinel),
				"inner",
			),
			"outer",
		),
		expectMsg:    "outer: inner: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "wrap stack",
		err:          cluerr.Wrap(cluerr.Stack(target, sentinel), "wrap"),
		expectMsg:    "wrap: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "stackwrap",
		err:          cluerr.StackWrap(target, sentinel, "wrap"),
		expectMsg:    "target: wrap: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "stackwrapWC",
		err:          cluerr.StackWrapWC(context.Background(), target, sentinel, "wrap"),
		expectMsg:    "target: wrap: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "wrap two stack: top",
		err:          cluerr.Wrap(cluerr.Stack(target, sentinel, other), "wrap"),
		expectMsg:    "wrap: target: sentinel: other",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "wrap two stack: base",
		err:          cluerr.Wrap(cluerr.Stack(other, target, sentinel), "wrap"),
		expectMsg:    "wrap: other: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name: "double double stack; left top",
		err: cluerr.Stack(
			cluerr.Stack(target, sentinel, leftBase),
			cluerr.Stack(rightTop, rightBase),
		),
		expectMsg: "target: sentinel: left-base: right-top: right-base",
		expectLabels: msa{
			lb: struct{}{},
			rt: struct{}{},
			rb: struct{}{},
		},
		expectValues: msa{
			lb: "v" + lb,
			rt: "v" + rt,
			rb: "v" + rb,
		},
	},
	{
		name: "double double stack; left base",
		err: cluerr.Stack(
			cluerr.Stack(leftTop, target, sentinel),
			cluerr.Stack(rightTop, rightBase),
		),
		expectMsg: "left-top: target: sentinel: right-top: right-base",
		expectLabels: msa{
			lt: struct{}{},
			rt: struct{}{},
			rb: struct{}{},
		},
		expectValues: msa{
			lt: "v" + lt,
			rt: "v" + rt,
			rb: "v" + rb,
		},
	},
	{
		name: "double double stack; right top",
		err: cluerr.Stack(
			cluerr.Stack(leftTop, leftBase),
			cluerr.Stack(target, sentinel, rightBase),
		),
		expectMsg: "left-top: left-base: target: sentinel: right-base",
		expectLabels: msa{
			lt: struct{}{},
			lb: struct{}{},
			rb: struct{}{},
		},
		expectValues: msa{
			lt: "v" + lt,
			lb: "v" + lb,
			rb: "v" + rb,
		},
	},
	{
		name: "double double animal wrap; right base",
		err: cluerr.Stack(
			cluerr.Wrap(cluerr.Stack(leftTop, leftBase), "left-stack"),
			cluerr.Wrap(cluerr.Stack(rightTop, target, sentinel), "right-stack"),
		),
		expectMsg: "left-stack: left-top: left-base: right-stack: right-top: target: sentinel",
		expectLabels: msa{
			lt: struct{}{},
			lb: struct{}{},
			rt: struct{}{},
		},
		expectValues: msa{
			lt: "v" + lt,
			lb: "v" + lb,
			rt: "v" + rt,
		},
	},
	{
		name: "double double animal wrap; left top",
		err: cluerr.Stack(
			cluerr.Wrap(cluerr.Stack(target, sentinel, leftBase), "left-stack"),
			cluerr.Wrap(cluerr.Stack(rightTop, rightBase), "right-stack"),
		),
		//nolint:lll
		expectMsg: "left-stack: target: sentinel: left-base: right-stack: right-top: right-base",
		expectLabels: msa{
			lb: struct{}{},
			rt: struct{}{},
			rb: struct{}{},
		},
		expectValues: msa{
			lb: "v" + lb,
			rt: "v" + rt,
			rb: "v" + rb,
		},
	},
	{
		name: "double double animal wrap; left base",
		err: cluerr.Stack(
			cluerr.Wrap(cluerr.Stack(leftTop, target, sentinel), "left-stack"),
			cluerr.Wrap(cluerr.Stack(rightTop, rightBase), "right-stack"),
		),
		expectMsg: "left-stack: left-top: target: sentinel: right-stack: right-top: right-base",
		expectLabels: msa{
			lt: struct{}{},
			rt: struct{}{},
			rb: struct{}{},
		},
		expectValues: msa{
			lt: "v" + lt,
			rt: "v" + rt,
			rb: "v" + rb,
		},
	},
	{
		name: "double double animal wrap; right top",
		err: cluerr.Stack(
			cluerr.Wrap(cluerr.Stack(leftTop, leftBase), "left-stack"),
			cluerr.Wrap(cluerr.Stack(target, sentinel, rightBase), "right-stack"),
		),
		expectMsg: "left-stack: left-top: left-base: right-stack: target: sentinel: right-base",
		expectLabels: msa{
			lt: struct{}{},
			lb: struct{}{},
			rb: struct{}{},
		},
		expectValues: msa{
			lt: "v" + lt,
			lb: "v" + lb,
			rb: "v" + rb,
		},
	},
	{
		name: "double double animal wrap; right base",
		err: cluerr.Stack(
			cluerr.Wrap(cluerr.Stack(leftTop, leftBase), "left-stack"),
			cluerr.Wrap(cluerr.Stack(rightTop, target, sentinel), "right-stack"),
		),
		expectMsg: "left-stack: left-top: left-base: right-stack: right-top: target: sentinel",
		expectLabels: msa{
			lt: struct{}{},
			lb: struct{}{},
			rt: struct{}{},
		},
		expectValues: msa{
			lt: "v" + lt,
			lb: "v" + lb,
			rt: "v" + rt,
		},
	},
}

func TestIs(t *testing.T) {
	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			if !errors.Is(test.err, sentinel) {
				t.Errorf("expected err [%v] to be true for errors.Is with [%s]", test.err, sentinel)
			}
		})
	}

	notSentinel := cluerr.New("sentinel")

	// NOT Is checks
	table := []struct {
		name string
		err  error
	}{
		{
			name: "plain stack",
			err:  cluerr.Stack(notSentinel),
		},
		{
			name: "plain wrap",
			err:  cluerr.Wrap(notSentinel, "wrap"),
		},
		{
			name: "double double animal wrap",
			err: cluerr.Stack(
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("left-top"),
						cluerr.New("left-base"),
					),
					"left-stack"),
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("right-top"),
						notSentinel,
					),
					"right-stack"),
			),
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			if errors.Is(test.err, sentinel) {
				t.Errorf("expected err [%v] to be FALSE for errors.Is with [%s]", test.err, sentinel)
			}
		})
	}
}

func TestAs(t *testing.T) {
	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			mt := mockTarget{}
			if !errors.As(test.err, &mt) {
				t.Errorf("expected err [%v] to be true for errors.As with [%s]", test.err, target)
			}
		})
	}

	notTarget := errors.New("target")

	// NOT As checks
	table := []struct {
		name string
		err  error
	}{
		{
			name: "plain stack",
			err:  cluerr.Stack(notTarget),
		},
		{
			name: "plain wrap",
			err:  cluerr.Wrap(notTarget, "wrap"),
		},
		{
			name: "double double animal wrap",
			err: cluerr.Stack(
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("left-top"),
						cluerr.New("left-base"),
					),
					"left-stack"),
				cluerr.Wrap(
					cluerr.Stack(
						cluerr.New("right-top"),
						notTarget,
					),
					"right-stack"),
			),
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			mt := mockTarget{}
			if errors.As(test.err, &mt) {
				t.Errorf("expected err [%v] to be FALSE for errors.As with [%s]", test.err, target)
			}
		})
	}
}

func TestToCore(t *testing.T) {
	for _, test := range testTable {
		t.Run(test.name, func(t *testing.T) {
			c := cluerr.ToCore(test.err)

			if test.expectMsg != c.Msg {
				t.Errorf("expected Msg [%v], got [%v]", test.expectMsg, c.Msg)
			}

			tester.MustEquals(t, test.expectLabels, toMSA(c.Labels), false)
			tester.MustEquals(t, test.expectValues, toMSA(c.Values), false)
		})
	}
}

func TestStackNils(t *testing.T) {
	result := cluerr.Stack(nil)
	if result != nil {
		t.Errorf("expected nil, got [%v]", result)
	}

	e := cluerr.New("err")

	result = cluerr.Stack(e, nil)
	if result.Error() != e.Error() {
		t.Errorf("expected [%v], got [%v]", e, result)
	}

	result = cluerr.Stack(nil, e)
	if result.Error() != e.Error() {
		t.Errorf("expected [%v], got [%v]", e, result)
	}
}

func TestOrNil(t *testing.T) {
	table := []struct {
		name      string
		err       *cluerr.Err
		expectNil bool
	}{
		{
			name:      "nil",
			err:       nil,
			expectNil: true,
		},
		{
			name:      "nil stack",
			err:       cluerr.Stack(nil).With("foo", "bar"),
			expectNil: true,
		},
		{
			name:      "nil wrap",
			err:       cluerr.Wrap(nil, "msg").With("foo", "bar"),
			expectNil: true,
		},
		{
			name:      "nil",
			err:       cluErr,
			expectNil: false,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			if test.expectNil != (test.err.OrNil() == nil) {
				t.Errorf("nil state doesn't match expectations: got %v", test.err.OrNil())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func withSkipCaller(err error, depth int) error {
	return cluerr.SkipCaller(err, depth)
}

func cluesWithSkipCaller(err *cluerr.Err, depth int) error {
	return err.SkipCaller(depth)
}

func wrapWithFuncWithGeneric[E error](err E) *cluerr.Err {
	return cluerr.Wrap(err, "with-generic")
}

func withNoTrace(err error) *cluerr.Err {
	return cluerr.Wrap(err, "no-trace").NoTrace()
}

func withCommentWrapper(
	err error,
	msg string,
	vs ...any,
) error {
	// always add two comments to test that both are saved
	return cluerr.
		Stack(err).
		Comment(msg, vs...).
		Comment(msg+" - repeat", vs...)
}

func cluerrWithCommentWrapper(
	err *cluerr.Err,
	msg string,
	vs ...any,
) error {
	// always add two comments to test that both are saved
	return err.
		Comment(msg, vs...).
		Comment(msg+" - repeat", vs...)
}
