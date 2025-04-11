package tester_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/tester"
)

type mockT struct {
	t         *testing.T
	shouldErr bool
	sawErr    bool
}

func (t *mockT) Error(args ...any) {
	t.sawErr = true

	if !t.shouldErr {
		t.t.Error(append([]any{"unexpected error:"}, args...)...)
	}
}

func (t *mockT) Errorf(format string, args ...any) {
	t.sawErr = true

	if !t.shouldErr {
		t.t.Errorf(
			"unexpected error: "+format,
			append([]any{"unexpected error:"}, args...)...)
	}
}

func (t *mockT) Log(args ...any) {
	t.t.Log(args...)
}

func (t *mockT) Logf(format string, args ...any) {
	t.t.Logf(format, args...)
}

func (t *mockT) verify() {
	if t.shouldErr && !t.sawErr {
		t.t.Error("expected an error, saw none")
	}
}

func TestContains(t *testing.T) {
	table := []struct {
		name         string
		input        any
		want         []any
		expecter     func(t *testing.T) *mockT
		expectFailed bool
	}{
		{
			name:  "nil",
			input: nil,
			want:  nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "nil wants with ctx background",
			input: context.Background(),
			want:  nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "nil wants with new error",
			input: cluerr.New("new"),
			want:  nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "non-cluerr error",
			input: errors.New("new"),
			want:  []any{"foo", "bar"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "ctx with match",
			input: clues.Add(context.Background(), "foo", "bar"),
			want:  []any{"foo", "bar"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "ctx with match and extras",
			input: clues.Add(context.Background(), 1, 2, "foo", "bar"),
			want:  []any{1, 2},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "ctx with bad match",
			input: clues.Add(context.Background(), "foo", "bar"),
			want:  []any{"foo", "fnords"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "ctx with missing key",
			input: clues.Add(context.Background(), 1, 2),
			want:  []any{3, tester.AnyVal},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "error with match",
			input: cluerr.New("new").With("foo", "bar"),
			want:  []any{"foo", "bar"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "error with match and extras",
			input: cluerr.New("new").With(1, 2, "foo", "bar"),
			want:  []any{1, 2},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "error with bad match",
			input: cluerr.New("new").With("foo", "bar"),
			want:  []any{"foo", "fnords"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "error with missing key",
			input: cluerr.New("new").With(1, 2),
			want:  []any{3, tester.AnyVal},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "AllPass",
			input: cluerr.New("new"),
			want:  []any{tester.AllPass},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			et := test.expecter(t)

			failed := tester.Contains(test.input, et, test.want...)

			et.verify()

			assert.Equal(t, test.expectFailed, failed)
		})
	}
}

func TestContainsMap(t *testing.T) {
	table := []struct {
		name         string
		input        any
		want         map[string]any
		expecter     func(t *testing.T) *mockT
		expectFailed bool
	}{
		{
			name:  "nil",
			input: nil,
			want:  nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "nil wants with ctx background",
			input: context.Background(),
			want:  nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "nil wants with new error",
			input: cluerr.New("new"),
			want:  nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "non-cluerr error",
			input: errors.New("new"),
			want:  map[string]any{"foo": "bar"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "ctx with match",
			input: clues.Add(context.Background(), "foo", "bar"),
			want:  map[string]any{"foo": "bar"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "ctx with match and extras",
			input: clues.Add(context.Background(), 1, 2, "foo", "bar"),
			want:  map[string]any{"1": 2},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "ctx with bad match",
			input: clues.Add(context.Background(), "foo", "bar"),
			want:  map[string]any{"foo": "fnords"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "ctx with missing key",
			input: clues.Add(context.Background(), 1, 2),
			want:  map[string]any{"3": tester.AnyVal},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "error with match",
			input: cluerr.New("new").With("foo", "bar"),
			want:  map[string]any{"foo": "bar"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "error with match and extras",
			input: cluerr.New("new").With(1, 2, "foo", "bar"),
			want:  map[string]any{"1": 2},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name:  "error with bad match",
			input: cluerr.New("new").With("foo", "bar"),
			want:  map[string]any{"foo": "fnords"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name:  "error with missing key",
			input: cluerr.New("new").With(1, 2),
			want:  map[string]any{"3": tester.AnyVal},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			et := test.expecter(t)

			failed := tester.ContainsMap(test.input, et, test.want)

			et.verify()

			assert.Equal(t, test.expectFailed, failed)
		})
	}
}

func TestContainsLabels(t *testing.T) {
	table := []struct {
		name         string
		err          error
		want         []string
		expecter     func(t *testing.T) *mockT
		expectFailed bool
	}{
		{
			name: "nil error, nil labels",
			err:  nil,
			want: nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name: "nil error expecting labels",
			err:  nil,
			want: []string{"fisher"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name: "error expecting no labels",
			err:  cluerr.New("new"),
			want: nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name: "error with labels expecting no labels",
			err:  cluerr.New("new").Label("label"),
			want: nil,
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name: "matched labels",
			err:  cluerr.New("new").Label("label"),
			want: []string{"label"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name: "multiple matched labels",
			err:  cluerr.New("new").Label("label", "ihaveseenthefnords"),
			want: []string{"ihaveseenthefnords", "label"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name: "matched labels with extras",
			err:  cluerr.New("new").Label("label", "ihaveseenthefnords"),
			want: []string{"label"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
		},
		{
			name: "missing labels",
			err:  cluerr.New("new").Label("label"),
			want: []string{"slab"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name: "partially mismatched labels",
			err:  cluerr.New("new").Label("label", "ihaveseenthefnords"),
			want: []string{"label", "fisher flannigan fitzbog"},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, true, false}
			},
			expectFailed: true,
		},
		{
			name: "always pass",
			err:  cluerr.New("new").Label("label"),
			want: []string{tester.AllPass},
			expecter: func(t *testing.T) *mockT {
				return &mockT{t, false, false}
			},
			expectFailed: false,
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			et := test.expecter(t)

			failed := tester.ContainsLabels(et, test.err, test.want...)

			et.verify()

			assert.Equal(t, test.expectFailed, failed)
		})
	}
}
