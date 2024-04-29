package clues_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"golang.org/x/exp/maps"

	"github.com/alcionai/clues"
)

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
				return clues.Stack(nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "DoubleNil",
			getErr: func() error {
				return clues.Stack(nil, nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "TripleNil",
			getErr: func() error {
				return clues.Stack(nil, nil, nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "StackNilNil",
			getErr: func() error {
				return clues.Stack(clues.Stack(nil), nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "NilStackNilNil",
			getErr: func() error {
				return clues.Stack(nil, clues.Stack(nil), nil).OrNil()
			},
			expectNil: true,
		},
		{
			name: "NilInterfaceError",
			getErr: func() error {
				var e testingErrorIface

				return clues.Stack(nil, e, clues.Stack(nil)).OrNil()
			},
			expectNil: true,
		},
		{
			name: "NonNilNonPointerInterfaceError",
			getErr: func() error {
				var e testingErrorIface = testingError{}

				return clues.Stack(nil, e, clues.Stack(nil)).OrNil()
			},
			expectNil: false,
		},
		{
			name: "NonNilNonPointerInterfaceError",
			getErr: func() error {
				var e testingErrorIface = &testingError{}

				return clues.Stack(nil, e, clues.Stack(nil)).OrNil()
			},
			expectNil: false,
		},
		{
			name: "NonPointerError",
			getErr: func() error {
				return clues.Stack(nil, testingError{}, clues.Stack(nil)).OrNil()
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
			initial: clues.Stack(
				clues.New("Labeled").Label(label),
				clues.New("NotLabeled")),
		},
		{
			name: "multiple stacked clues errors with label on second",
			initial: clues.Stack(
				clues.New("NotLabeled"),
				clues.New("Labeled").Label(label)),
		},
		{
			name:    "single stacked clues error with label",
			initial: clues.Stack(clues.New("Labeled").Label(label)),
		},
	}

	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			if !clues.HasLabel(test.initial, label) {
				t.Errorf(
					"expected error to have label [%s] but got %v",
					label,
					maps.Keys(clues.Labels(test.initial)))
			}
		})
	}
}

func TestLabel(t *testing.T) {
	table := []struct {
		name    string
		initial error
		expect  func(*testing.T, *clues.Err)
	}{
		{"nil", nil, nil},
		{"standard error", errors.New("an error"), nil},
		{"clues error", clues.New("clues error"), nil},
		{"clues error wrapped", fmt.Errorf("%w", clues.New("clues error")), nil},
		{
			"clues error with label",
			clues.New("clues error").Label("fnords"),
			func(t *testing.T, err *clues.Err) {
				if !err.HasLabel("fnords") {
					t.Error("expected error to have label [fnords]")
				}
			},
		},
		{
			"clues error with label wrapped",
			fmt.Errorf("%w", clues.New("clues error").Label("fnords")),
			func(t *testing.T, err *clues.Err) {
				if !err.HasLabel("fnords") {
					t.Error("expected error to have label [fnords]")
				}
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			if clues.HasLabel(test.initial, "foo") {
				t.Error("new error should have no label")
			}

			err := clues.Label(test.initial, "foo")
			if !clues.HasLabel(err, "foo") && test.initial != nil {
				t.Error("expected error to have label [foo]")
			}

			if err == nil {
				if test.initial != nil {
					t.Error("error should not be nil after labeling")
				}
				return
			}

			err.Label("bar")
			if !clues.HasLabel(err, "bar") {
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
		a     = clues.New("a").Label("a")
		acopy = clues.New("acopy").Label("a")
		b     = clues.New("b").Label("b")
		wrap  = clues.Wrap(
			clues.Stack(
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
		{"unlabeled error", clues.New("clues error"), msa{}},
		{"pkg/errs wrap around labeled error", errors.Wrap(a, "wa"), ma},
		{"clues wrapped", clues.Wrap(a, "wrap"), ma},
		{"clues stacked", clues.Stack(a, b), mab},
		{"clues stacked with copy", clues.Stack(a, b, acopy), mab},
		{"error chain", clues.Stack(b, fmt.Errorf("%w", a), fmt.Errorf("%w", acopy)), mab},
		{"error wrap chain", wrap, mab},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			result := clues.Labels(test.initial)
			mustEquals(t, test.expect, toMSA(result), false)
		})
	}
}

var (
	base = errors.New("an error")
	cerr = func() error { return clues.Stack(base) }
	werr = func() error {
		return fmt.Errorf("%w", clues.Wrap(base, "wrapped error with vals").With("z", 0))
	}
	serr = func() error {
		return clues.Stack(clues.New("primary").With("z", 0), errors.New("secondary"))
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
		{"nil error", nil, "k", "v", [][]any{{"k2", "v2"}}, msa{}},
		{"only base error vals", base, "k", "v", nil, msa{"k": "v"}},
		{"empty base error vals", base, "", "", nil, msa{"": ""}},
		{"standard", base, "k", "v", [][]any{{"k2", "v2"}}, msa{"k": "v", "k2": "v2"}},
		{"duplicates", base, "k", "v", [][]any{{"k", "v2"}}, msa{"k": "v2"}},
		{"multi", base, "a", "1", [][]any{{"b", "2"}, {"c", "3"}}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only clue error vals", cerr(), "k", "v", nil, msa{"k": "v"}},
		{"empty clue error vals", cerr(), "", "", nil, msa{"": ""}},
		{"standard cerr", cerr(), "k", "v", [][]any{{"k2", "v2"}}, msa{"k": "v", "k2": "v2"}},
		{"duplicates cerr", cerr(), "k", "v", [][]any{{"k", "v2"}}, msa{"k": "v2"}},
		{"multi cerr", cerr(), "a", "1", [][]any{{"b", "2"}, {"c", "3"}}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only wrapped error vals", werr(), "k", "v", nil, msa{"k": "v", "z": 0}},
		{"empty wrapped error vals", werr(), "", "", nil, msa{"": "", "z": 0}},
		{"standard wrapped", werr(), "k", "v", [][]any{{"k2", "v2"}}, msa{"k": "v", "k2": "v2", "z": 0}},
		{"duplicates wrapped", werr(), "k", "v", [][]any{{"k", "v2"}}, msa{"k": "v2", "z": 0}},
		{"multi wrapped", werr(), "a", "1", [][]any{{"b", "2"}, {"c", "3"}}, msa{"a": "1", "b": "2", "c": "3", "z": 0}},
		{"only stacked error vals", serr(), "k", "v", nil, msa{"k": "v", "z": 0}},
		{"empty stacked error vals", serr(), "", "", nil, msa{"": "", "z": 0}},
		{"standard stacked", serr(), "k", "v", [][]any{{"k2", "v2"}}, msa{"k": "v", "k2": "v2", "z": 0}},
		{"duplicates stacked", serr(), "k", "v", [][]any{{"k", "v2"}}, msa{"k": "v2", "z": 0}},
		{"multi stacked", serr(), "a", "1", [][]any{{"b", "2"}, {"c", "3"}}, msa{"a": "1", "b": "2", "c": "3", "z": 0}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := clues.With(test.initial, test.k, test.v)
			for _, kv := range test.with {
				err = err.With(kv...)
			}
			mustEquals(t, test.expect, clues.InErr(err).Map(), true)
			mustEquals(t, test.expect, err.Values().Map(), true)
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
		{"nil error", nil, msa{"k": "v"}, msa{"k2": "v2"}, msa{}},
		{"only base error vals", base, msa{"k": "v"}, nil, msa{"k": "v"}},
		{"empty base error vals", base, msa{"": ""}, nil, msa{"": ""}},
		{"standard", base, msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2"}},
		{"duplicates", base, msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2"}},
		{"multi", base, msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only clue error vals", cerr(), msa{"k": "v"}, nil, msa{"k": "v"}},
		{"empty clue error vals", cerr(), msa{"": ""}, nil, msa{"": ""}},
		{"standard cerr", cerr(), msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2"}},
		{"duplicates cerr", cerr(), msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2"}},
		{"multi cerr", cerr(), msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only wrapped error vals", werr(), msa{"k": "v"}, nil, msa{"k": "v", "z": 0}},
		{"empty wrapped error vals", werr(), msa{"": ""}, nil, msa{"": "", "z": 0}},
		{"standard wrapped", werr(), msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2", "z": 0}},
		{"duplicates wrapped", werr(), msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2", "z": 0}},
		{"multi wrapped", werr(), msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3", "z": 0}},
		{"only stacked error vals", serr(), msa{"k": "v"}, nil, msa{"k": "v", "z": 0}},
		{"empty stacked error vals", serr(), msa{"": ""}, nil, msa{"": "", "z": 0}},
		{"standard stacked", serr(), msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2", "z": 0}},
		{"duplicates stacked", serr(), msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2", "z": 0}},
		{"multi stacked", serr(), msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3", "z": 0}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := clues.WithMap(test.initial, test.kv)
			err = err.WithMap(test.with)
			mustEquals(t, test.expect, clues.InErr(err).Map(), true)
			mustEquals(t, test.expect, err.Values().Map(), true)
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
		{"nil error", nil, msa{"k": "v"}, msa{"k2": "v2"}, msa{}},
		{"only base error vals", base, msa{"k": "v"}, nil, msa{"k": "v"}},
		{"empty base error vals", base, msa{"": ""}, nil, msa{"": ""}},
		{"standard", base, msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2"}},
		{"duplicates", base, msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2"}},
		{"multi", base, msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only clue error vals", cerr(), msa{"k": "v"}, nil, msa{"k": "v"}},
		{"empty clue error vals", cerr(), msa{"": ""}, nil, msa{"": ""}},
		{"standard cerr", cerr(), msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2"}},
		{"duplicates cerr", cerr(), msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2"}},
		{"multi cerr", cerr(), msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only wrapped error vals", werr(), msa{"k": "v"}, nil, msa{"k": "v", "z": 0}},
		{"empty wrapped error vals", werr(), msa{"": ""}, nil, msa{"": "", "z": 0}},
		{"standard wrapped", werr(), msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2", "z": 0}},
		{"duplicates wrapped", werr(), msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2", "z": 0}},
		{"multi wrapped", werr(), msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3", "z": 0}},
		{"only stacked error vals", serr(), msa{"k": "v"}, nil, msa{"k": "v", "z": 0}},
		{"empty stacked error vals", serr(), msa{"": ""}, nil, msa{"": "", "z": 0}},
		{"standard stacked", serr(), msa{"k": "v"}, msa{"k2": "v2"}, msa{"k": "v", "k2": "v2", "z": 0}},
		{"duplicates stacked", serr(), msa{"k": "v"}, msa{"k": "v2"}, msa{"k": "v2", "z": 0}},
		{"multi stacked", serr(), msa{"a": "1"}, msa{"b": "2", "c": "3"}, msa{"a": "1", "b": "2", "c": "3", "z": 0}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			tctx := clues.AddMap(ctx, test.kv)
			err := clues.WithClues(test.initial, tctx)
			err = err.WithMap(test.with)
			mustEquals(t, test.expect, clues.InErr(err).Map(), true)
			mustEquals(t, test.expect, err.Values().Map(), true)
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

				err := clues.NewWC(ctx, "err").With("in-err", 1)
				// the first addition to an error should take priority
				err = clues.StackWC(ctx, err).With("in-err", 2)

				return err
			}(),
			expect: msa{"in-ctx": 2, "in-err": 1},
		},
		{
			name: "last stack wins",
			err: func() error {
				ctx := clues.Add(context.Background(), "in-ctx", 1)
				err := clues.NewWC(ctx, "last in stack").With("in-err", 1)
				err = clues.Stack(
					clues.New("first in stack").With("in-err", 2),
					err)
				return err
			}(),
			expect: msa{"in-ctx": 1, "in-err": 1},
		},
		{
			name: ".With wins over ctx",
			err: func() error {
				ctx := clues.Add(context.Background(), "k", 1)
				err := clues.NewWC(ctx, "last in stack").With("k", 2)
				return err
			}(),
			expect: msa{"k": 2},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			mustEquals(t, test.expect, clues.InErr(test.err).Map(), true)
		})
	}
}

func TestUnwrap(t *testing.T) {
	e := errors.New("cause")
	we := clues.Wrap(e, "outer")

	ce := we.Unwrap()
	if ce != e {
		t.Errorf("expected result error [%v] to be base error [%v]\n", ce, e)
	}

	ce = clues.Unwrap(we)
	if ce != e {
		t.Errorf("expected result error [%v] to be base error [%v]\n", ce, e)
	}

	se := clues.Stack(e)

	ce = se.Unwrap()
	if ce != e {
		t.Errorf("expected result error [%v] to be base error [%v]\n", ce, e)
	}

	ce = clues.Unwrap(se)
	if ce != e {
		t.Errorf("expected result error [%v] to be base error [%v]\n", ce, e)
	}

	if clues.Unwrap(nil) != nil {
		t.Errorf("expected nil unwrap input to return nil")
	}
}

func TestWrapNilStackSlice(t *testing.T) {
	// an empty slice of errors
	sl := make([]error, 10)
	// when stacked
	st := clues.Stack(sl...)
	// then wrapped
	e := clues.Wrap(st, "wrapped")
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
			err:    clues.New("new"),
			expect: "new",
		},
		{
			name:   "stacked error",
			err:    clues.Stack(sentinel),
			expect: sentinel.Error(),
		},
		{
			name:   "wrapped new error",
			err:    clues.Wrap(clues.New("new"), "wrap"),
			expect: "wrap: new",
		},
		{
			name:   "wrapped non-clues error",
			err:    clues.Wrap(sentinel, "wrap"),
			expect: "wrap: " + sentinel.Error(),
		},
		{
			name:   "wrapped stacked error",
			err:    clues.Wrap(clues.Stack(sentinel), "wrap"),
			expect: "wrap: " + sentinel.Error(),
		},
		{
			name:   "multiple wraps",
			err:    clues.Wrap(clues.Wrap(clues.New("new"), "wrap"), "wrap2"),
			expect: "wrap2: wrap: new",
		},
		{
			name:   "wrap-stack-wrap-new",
			err:    clues.Wrap(clues.Stack(clues.Wrap(clues.New("new"), "wrap")), "wrap2"),
			expect: "wrap2: wrap: new",
		},
		{
			name:   "many stacked errors",
			err:    clues.Stack(sentinel, errors.New("middle"), errors.New("base")),
			expect: sentinel.Error() + ": middle: base",
		},
		{
			name: "stacked stacks",
			err: clues.Stack(
				clues.Stack(sentinel, errors.New("left")),
				clues.Stack(errors.New("right"), errors.New("base")),
			),
			expect: sentinel.Error() + ": left: right: base",
		},
		{
			name: "wrapped stacks",
			err: clues.Stack(
				clues.Wrap(clues.Stack(errors.New("top"), errors.New("left")), "left-stack"),
				clues.Wrap(clues.Stack(errors.New("right"), errors.New("base")), "right-stack"),
			),
			expect: "left-stack: top: left: right-stack: right: base",
		},
		{
			name: "wrapped stacks, all clues.New",
			err: clues.Stack(
				clues.Wrap(clues.Stack(clues.New("top"), clues.New("left")), "left-stack"),
				clues.Wrap(clues.Stack(clues.New("right"), clues.New("base")), "right-stack"),
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
			err:    clues.Stack(clues.New("an err").With("k", "v")),
			expect: msa{"k": "v"},
		},
		{
			name: "two stack",
			err: clues.Stack(
				clues.New("an err").With("k", "v"),
				clues.New("other").With("k2", "v2"),
			),
			expect: msa{"k": "v", "k2": "v2"},
		},
		{
			name: "sandvitch",
			err: clues.Stack(
				clues.New("top").With("k", "v"),
				errors.New("mid"),
				clues.New("base").With("k2", "v2"),
			),
			expect: msa{"k": "v", "k2": "v2"},
		},
		{
			name: "value collision",
			err: clues.Stack(
				clues.New("top").With("k", "v"),
				clues.New("mid").With("k2", "v2"),
				clues.New("base").With("k", "v3"),
			),
			expect: msa{"k": "v3", "k2": "v2"},
		},
		{
			name: "double double",
			err: clues.Stack(
				clues.Stack(
					clues.New("top").With("k", "v"),
					clues.New("left").With("k2", "v2"),
				),
				clues.Stack(
					clues.New("right").With("k3", "v3"),
					clues.New("base").With("k4", "v4"),
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
			err: clues.Stack(
				clues.Stack(
					clues.New("top").With("k", "v"),
					clues.New("left").With("k2", "v2"),
				),
				clues.Stack(
					clues.New("right").With("k3", "v3"),
					clues.New("base").With("k", "v4"),
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
			err: clues.Stack(
				clues.Wrap(
					clues.Stack(
						clues.New("top").With("k", "v"),
						clues.New("left").With("k2", "v2"),
					),
					"left-stack"),
				clues.Wrap(
					clues.Stack(
						clues.New("right").With("k3", "v3"),
						clues.New("base").With("k4", "v4"),
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
			err: clues.Stack(
				clues.Wrap(
					clues.Stack(
						clues.New("top").With("k", "v"),
						clues.New("left").With("k2", "v2"),
					),
					"left-stack"),
				clues.Wrap(
					clues.Stack(
						clues.New("right").With("k3", "v3"),
						clues.New("base").With("k", "v4"),
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
			vs := clues.InErr(test.err)
			mustEquals(t, test.expect, vs.Map(), true)
		})
	}
}

func TestImmutableErrors(t *testing.T) {
	err := clues.New("an error").With("k", "v")
	check := msa{"k": "v"}
	pre := clues.InErr(err)
	mustEquals(t, check, pre.Map(), true)

	err2 := err.With("k2", "v2")
	if _, ok := pre.Map()["k2"]; ok {
		t.Errorf("previous map should not have been mutated by addition")
	}

	post := clues.InErr(err2)
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

var (
	target    = mockTarget{errors.New(tgt)}
	sentinel  = errors.New(stnl)
	other     = errors.New("other")
	leftTop   = clues.New(lt).With(lt, "v"+lt).Label(lt)
	leftBase  = clues.New(lb).With(lb, "v"+lb).Label(lb)
	rightTop  = clues.New(rt).With(rt, "v"+rt).Label(rt)
	rightBase = clues.New(rb).With(rb, "v"+rb).Label(rb)
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
		err:          clues.Stack(target, sentinel),
		expectMsg:    "target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "plain wrap",
		err:          clues.Wrap(clues.Stack(target, sentinel), "wrap"),
		expectLabels: msa{},
		expectMsg:    "wrap: target: sentinel",
		expectValues: msa{},
	},
	{
		name:         "two stack; top",
		err:          clues.Stack(clues.Stack(target, sentinel), other),
		expectMsg:    "target: sentinel: other",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "two stack; base",
		err:          clues.Stack(other, clues.Stack(target, sentinel)),
		expectMsg:    "other: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "two wrap",
		err:          clues.Wrap(clues.Wrap(clues.Stack(target, sentinel), "inner"), "outer"),
		expectMsg:    "outer: inner: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "wrap stack",
		err:          clues.Wrap(clues.Stack(target, sentinel), "wrap"),
		expectMsg:    "wrap: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "wrap two stack: top",
		err:          clues.Wrap(clues.Stack(target, sentinel, other), "wrap"),
		expectMsg:    "wrap: target: sentinel: other",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name:         "wrap two stack: base",
		err:          clues.Wrap(clues.Stack(other, target, sentinel), "wrap"),
		expectMsg:    "wrap: other: target: sentinel",
		expectLabels: msa{},
		expectValues: msa{},
	},
	{
		name: "double double stack; left top",
		err: clues.Stack(
			clues.Stack(target, sentinel, leftBase),
			clues.Stack(rightTop, rightBase),
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
		err: clues.Stack(
			clues.Stack(leftTop, target, sentinel),
			clues.Stack(rightTop, rightBase),
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
		err: clues.Stack(
			clues.Stack(leftTop, leftBase),
			clues.Stack(target, sentinel, rightBase),
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
		err: clues.Stack(
			clues.Wrap(clues.Stack(leftTop, leftBase), "left-stack"),
			clues.Wrap(clues.Stack(rightTop, target, sentinel), "right-stack"),
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
		err: clues.Stack(
			clues.Wrap(clues.Stack(target, sentinel, leftBase), "left-stack"),
			clues.Wrap(clues.Stack(rightTop, rightBase), "right-stack"),
		),
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
		err: clues.Stack(
			clues.Wrap(clues.Stack(leftTop, target, sentinel), "left-stack"),
			clues.Wrap(clues.Stack(rightTop, rightBase), "right-stack"),
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
		err: clues.Stack(
			clues.Wrap(clues.Stack(leftTop, leftBase), "left-stack"),
			clues.Wrap(clues.Stack(target, sentinel, rightBase), "right-stack"),
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
		err: clues.Stack(
			clues.Wrap(clues.Stack(leftTop, leftBase), "left-stack"),
			clues.Wrap(clues.Stack(rightTop, target, sentinel), "right-stack"),
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

	notSentinel := clues.New("sentinel")

	// NOT Is checks
	table := []struct {
		name string
		err  error
	}{
		{
			name: "plain stack",
			err:  clues.Stack(notSentinel),
		},
		{
			name: "plain wrap",
			err:  clues.Wrap(notSentinel, "wrap"),
		},
		{
			name: "double double animal wrap",
			err: clues.Stack(
				clues.Wrap(
					clues.Stack(
						clues.New("left-top"),
						clues.New("left-base"),
					),
					"left-stack"),
				clues.Wrap(
					clues.Stack(
						clues.New("right-top"),
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
			err:  clues.Stack(notTarget),
		},
		{
			name: "plain wrap",
			err:  clues.Wrap(notTarget, "wrap"),
		},
		{
			name: "double double animal wrap",
			err: clues.Stack(
				clues.Wrap(
					clues.Stack(
						clues.New("left-top"),
						clues.New("left-base"),
					),
					"left-stack"),
				clues.Wrap(
					clues.Stack(
						clues.New("right-top"),
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
			c := clues.ToCore(test.err)
			if test.expectMsg != c.Msg {
				t.Errorf("expected Msg [%v], got [%v]", test.expectMsg, c.Msg)
			}
			mustEquals(t, test.expectLabels, toMSA(c.Labels), false)
			mustEquals(t, test.expectValues, toMSA(c.Values), true)
		})
	}
}

func TestStackNils(t *testing.T) {
	result := clues.Stack(nil)
	if result != nil {
		t.Errorf("expected nil, got [%v]", result)
	}

	e := clues.New("err")
	result = clues.Stack(e, nil)
	if result.Error() != e.Error() {
		t.Errorf("expected [%v], got [%v]", e, result)
	}

	result = clues.Stack(nil, e)
	if result.Error() != e.Error() {
		t.Errorf("expected [%v], got [%v]", e, result)
	}
}

func TestOrNil(t *testing.T) {
	table := []struct {
		name      string
		err       *clues.Err
		expectNil bool
	}{
		{
			name:      "nil",
			err:       nil,
			expectNil: true,
		},
		{
			name:      "nil stack",
			err:       clues.Stack(nil).With("foo", "bar"),
			expectNil: true,
		},
		{
			name:      "nil wrap",
			err:       clues.Wrap(nil, "msg").With("foo", "bar"),
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

type labelCounter map[string]int64

func (tla labelCounter) Add(l string, i int64) {
	tla[l] = tla[l] + i
}

var labelTable = []struct {
	name   string
	labels []string
	expect map[string]int64
}{
	{
		name:   "no labels",
		labels: []string{},
		expect: map[string]int64{},
	},
	{
		name:   "single label",
		labels: []string{"un"},
		expect: map[string]int64{
			"un": 1,
		},
	},
	{
		name:   "multiple labels",
		labels: []string{"un", "deux"},
		expect: map[string]int64{
			"un":   1,
			"deux": 1,
		},
	},
	{
		name:   "duplicated label",
		labels: []string{"un", "un"},
		expect: map[string]int64{
			"un": 1,
		},
	},
	{
		name:   "multiple duplicated labels",
		labels: []string{"un", "un", "deux", "deux"},
		expect: map[string]int64{
			"un":   1,
			"deux": 1,
		},
	},
	{
		name:   "empty string labels",
		labels: []string{"", "", "un", "deux"},
		expect: map[string]int64{
			"":     1,
			"un":   1,
			"deux": 1,
		},
	},
}

func TestLabelCounter_iterative(t *testing.T) {
	for _, test := range labelTable {
		t.Run(test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.NewWC(ctx, "an err")
			)

			for _, l := range test.labels {
				err.Label(l)
			}

			mustEquals(t, toMSA(test.expect), toMSA(lc), false)
		})
	}
}

func TestLabelCounter_variadic(t *testing.T) {
	for _, test := range labelTable {
		t.Run(test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.NewWC(ctx, "an err")
			)

			err.Label(test.labels...)

			mustEquals(t, toMSA(test.expect), toMSA(lc), false)
		})
	}
}

func TestLabelCounter_iterative_stacked(t *testing.T) {
	for _, test := range labelTable {
		t.Run(test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.NewWC(ctx, "an err")
			)

			for _, l := range test.labels {
				err.Label(l)
			}

			err = clues.Stack(err)

			// duplicates on the wrapped error should not get counted
			for _, l := range test.labels {
				err.Label(l)
			}

			mustEquals(t, toMSA(test.expect), toMSA(lc), false)
		})
	}
}

func TestLabelCounter_variadic_stacked(t *testing.T) {
	for _, test := range labelTable {
		t.Run(test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.NewWC(ctx, "an err")
			)

			err.Label(test.labels...)

			// duplicates on the wrapped error should not get counted
			err = clues.Stack(err).Label(test.labels...)

			mustEquals(t, toMSA(test.expect), toMSA(lc), false)
		})
	}
}

func TestLabelCounter_iterative_wrapped(t *testing.T) {
	for _, test := range labelTable {
		t.Run(test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.NewWC(ctx, "an err")
			)

			for _, l := range test.labels {
				err.Label(l)
			}

			err = clues.Wrap(err, "wrap")

			// duplicates on the wrapped error should not get counted
			for _, l := range test.labels {
				err.Label(l)
			}

			mustEquals(t, toMSA(test.expect), toMSA(lc), false)
		})
	}
}

func TestLabelCounter_variadic_wrapped(t *testing.T) {
	for _, test := range labelTable {
		t.Run(test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.NewWC(ctx, "an err")
			)

			err.Label(test.labels...)

			// duplicates on the wrapped error should not get counted
			err = clues.Wrap(err, "wrap").Label(test.labels...)

			mustEquals(t, toMSA(test.expect), toMSA(lc), false)
		})
	}
}

func TestLabelCounter_iterative_noCluesInErr(t *testing.T) {
	for _, test := range labelTable {
		t.Run(test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.New("an err")
			)

			for _, l := range test.labels {
				err.Label(l)
			}

			err = err.WithClues(ctx)

			// no labeling before WithClues is called on the error
			mustEquals(t, toMSA(lc), msa{}, false)
		})
	}
}

func TestLabelCounter_variadic_noCluesInErr(t *testing.T) {
	for _, test := range labelTable {
		t.Run("variadic_"+test.name, func(t *testing.T) {
			var (
				lc  = labelCounter{}
				ctx = clues.AddLabelCounter(context.Background(), lc)
				err = clues.New("an err")
			)

			err.Label(test.labels...).WithClues(ctx)

			// no labeling before WithClues is called on the error
			mustEquals(t, toMSA(lc), msa{}, false)
		})
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func withTraceWrapper(err error, depth int) error {
	return clues.WithTrace(err, depth)
}

func cluesWithTraceWrapper(err *clues.Err, depth int) error {
	return err.WithTrace(depth)
}
