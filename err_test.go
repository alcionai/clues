package clues_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/alcionai/clues"
)

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

var (
	base = errors.New("an error")
	cerr = func() error { return clues.Stack(base) }
	werr = func() error {
		return fmt.Errorf("%w", clues.Wrap(base, "wrapped error with vals").With("z", 0))
	}
)

func TestWith(t *testing.T) {

	table := []struct {
		name    string
		initial error
		k, v    string
		with    [][]string
		expect  msa
	}{
		{"nil error", nil, "k", "v", [][]string{{"k2", "v2"}}, msa{}},
		{"only base error vals", base, "k", "v", nil, msa{"k": "v"}},
		{"empty base error vals", base, "", "", nil, msa{"": ""}},
		{"standard", base, "k", "v", [][]string{{"k2", "v2"}}, msa{"k": "v", "k2": "v2"}},
		{"duplicates", base, "k", "v", [][]string{{"k", "v2"}}, msa{"k": "v2"}},
		{"multi", base, "a", "1", [][]string{{"b", "2"}, {"c", "3"}}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only clue error vals", cerr(), "k", "v", nil, msa{"k": "v"}},
		{"empty clue error vals", cerr(), "", "", nil, msa{"": ""}},
		{"standard cerr", cerr(), "k", "v", [][]string{{"k2", "v2"}}, msa{"k": "v", "k2": "v2"}},
		{"duplicates cerr", cerr(), "k", "v", [][]string{{"k", "v2"}}, msa{"k": "v2"}},
		{"multi cerr", cerr(), "a", "1", [][]string{{"b", "2"}, {"c", "3"}}, msa{"a": "1", "b": "2", "c": "3"}},
		{"only wrapped error vals", werr(), "k", "v", nil, msa{"k": "v", "z": 0}},
		{"empty wrapped error vals", werr(), "", "", nil, msa{"": "", "z": 0}},
		{"standard wrapped", werr(), "k", "v", [][]string{{"k2", "v2"}}, msa{"k": "v", "k2": "v2", "z": 0}},
		{"duplicates wrapped", werr(), "k", "v", [][]string{{"k", "v2"}}, msa{"k": "v2", "z": 0}},
		{"multi wrapped", werr(), "a", "1", [][]string{{"b", "2"}, {"c", "3"}}, msa{"a": "1", "b": "2", "c": "3", "z": 0}},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := clues.With(test.initial, test.k, test.v)
			for _, kv := range test.with {
				err.With(kv[0], kv[1])
			}
			test.expect.equals(t, clues.ErrValues(err))
			test.expect.equals(t, err.Values())
		})
	}
}

func TestWithAll(t *testing.T) {
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
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := clues.WithAll(test.initial, test.k, test.v)
			for _, kv := range test.with {
				err.WithAll(kv...)
			}
			test.expect.equals(t, clues.ErrValues(err))
			test.expect.equals(t, err.Values())
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
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			err := clues.WithMap(test.initial, test.kv)
			err.WithMap(test.with)
			test.expect.equals(t, clues.ErrValues(err))
			test.expect.equals(t, err.Values())
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
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			tctx := clues.AddMap(ctx, test.kv)
			err := clues.WithClues(test.initial, tctx)
			err.WithMap(test.with)
			test.expect.equals(t, clues.ErrValues(err))
			test.expect.equals(t, err.Values())
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
			expect: msa{"k": "v", "k2": "v2"},
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
				"k":  "v",
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
				"k":  "v",
				"k2": "v2",
				"k3": "v3",
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			vs := clues.ErrValues(test.err)
			test.expect.equals(t, vs)
		})
	}
}
