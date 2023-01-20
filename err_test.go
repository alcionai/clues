package clues_test

import (
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
