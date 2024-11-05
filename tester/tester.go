package tester

import (
	"context"
	"slices"

	"github.com/alcionai/clues"
	"github.com/alcionai/clues/cluerr"
	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
)

// ---------------------------------------------------------------------------
// types and interfaces
// ---------------------------------------------------------------------------

type anyVal struct{}

// AnyVal will pass the test so long as the key for this value exists.
var AnyVal = anyVal{}

// AllPass will always pass the test.
var AllPass = "all labels will pass if provided this magic string"

type expectGot struct {
	expect string
	got    string
}

// errLogfer allows us to pass in a mock testing.T
type errLogfer interface {
	Error(args ...any)
	Errorf(format string, args ...any)
	Log(args ...any)
	Logf(format string, args ...any)
}

// ---------------------------------------------------------------------------
// assertions
// ---------------------------------------------------------------------------

// Contains checks whether the errOrCtx (which should contain
// either an error or context.Context) contains the provided
// key:value pairs.
//
// Returns true if the test fails.
func Contains(
	t errLogfer,
	errOrCtx any,
	kvs ...any,
) bool {
	if slices.Contains(kvs, any(AllPass)) {
		t.Log("AllPass found; passing test")
		return false
	}

	// some sanity prechecks
	if len(kvs) == 0 {
		t.Error("no key:value properties provided to test")
		return true
	}

	if len(kvs)%2 == 1 {
		t.Error("odd count of key:value parameters")
		return true
	}

	n, ok := getNode(t, errOrCtx)
	if !ok {
		return true
	}

	var (
		values      = n.Map()
		badVals     = map[string]expectGot{}
		foundKeys   = map[string]struct{}{}
		missingKeys = map[string]struct{}{}
	)

	// iterate through each k,v pair looking for matches.
	for i := 0; i < len(kvs); i += 2 {
		k, v := stringify.Marshal(kvs[i], false), kvs[i+1]

		gotV, found := values[k]
		if !found {
			missingKeys[k] = struct{}{}
			continue
		}

		foundKeys[k] = struct{}{}

		if v == AnyVal {
			continue
		}

		var (
			expected = stringify.Marshal(v, false)
			got      = stringify.Marshal(gotV, false)
		)

		if expected != got {
			badVals[k] = expectGot{expected, got}
		}
	}

	// early pass check
	if len(badVals) == 0 && len(missingKeys) == 0 {
		return false
	}

	showContainsResults(t, values, badVals, foundKeys, missingKeys)

	return true
}

// Contains checks whether the errOrCtx (which should contain
// either an error or context.Context) contains the provided
// map.
//
// Returns true if the test fails.
func ContainsMap(
	t errLogfer,
	errOrCtx any,
	m map[string]any,
) bool {
	if len(m) == 0 {
		t.Error("no map properties provided to test")
		return true
	}

	n, ok := getNode(t, errOrCtx)
	if !ok {
		return true
	}

	var (
		values      = n.Map()
		badVals     = map[string]expectGot{}
		foundKeys   = map[string]struct{}{}
		missingKeys = map[string]struct{}{}
	)

	// iterate through each k,v pair looking for matches.
	for k, v := range m {
		gotV, found := values[k]
		if !found {
			missingKeys[k] = struct{}{}
			continue
		}

		foundKeys[k] = struct{}{}

		if v == AnyVal {
			continue
		}

		var (
			expected = stringify.Marshal(v, false)
			got      = stringify.Marshal(gotV, false)
		)

		if expected != got {
			badVals[k] = expectGot{expected, got}
		}
	}

	// early pass check
	if len(badVals) == 0 && len(missingKeys) == 0 {
		return false
	}

	showContainsResults(t, values, badVals, foundKeys, missingKeys)

	return true
}

// ContainsLabels checks whether the error(which should contain
// a cluerr.Err) contains the labels. If provided zero labels to
// check against, asserts that the error contains zero labels.
// Can be provided tester.AllPass to skip the check for a single
// test case.
//
// Returns true if the test fails.
func ContainsLabels(
	t errLogfer,
	err error,
	expected ...string,
) bool {
	// support an always-pass case
	if slices.Contains(expected, AllPass) {
		t.Log("AllPass found; passing test")
		return false
	}

	labels := cluerr.Labels(err)

	if err == nil {
		if len(expected) > 0 {
			t.Error("expected labels, but error is nil")
		}

		return len(expected) != 0
	}

	if len(expected) == 0 && len(labels) > 0 {
		t.Errorf("expected no labels in error, got:\t%v", labels)
		return true
	}

	extraLabels := map[string]struct{}{}

	for l := range labels {
		if !slices.Contains(expected, l) {
			extraLabels[l] = struct{}{}
		}
	}

	var errored bool

	for _, expect := range expected {
		if _, ok := labels[expect]; !ok {
			t.Error("missing label:", expect)
			errored = true
		}
	}

	if errored {
		t.Log("Unchecked labels")

		for extra := range extraLabels {
			t.Log("-", extra)
		}

		t.Log("")
	}

	return errored
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func showContainsResults(
	t errLogfer,
	values map[string]any,
	badVals map[string]expectGot,
	foundKeys map[string]struct{},
	missingKeys map[string]struct{},
) {
	// sanity showcase: print out all unchecked values
	if len(foundKeys) < len(values) {
		t.Log("Unchecked attributes")

		for k, v := range values {
			if _, ok := foundKeys[k]; !ok {
				t.Log("-", k+":", stringify.Marshal(v, false))
			}
		}

		t.Log("")
	}

	// failure showcase
	for k := range missingKeys {
		t.Error("missing entry with key ", k)
	}

	for k, eg := range badVals {
		t.Errorf(
			"unexpected value:\n\tkey: %s\n\texpected: %s\n\tgot: %s\n",
			k,
			eg.expect,
			eg.got)
	}
}

func getNode(
	t errLogfer,
	eoc any,
) (*node.Node, bool) {
	if noder, ok := eoc.(node.Noder); ok {
		return noder.Node(), true
	}

	if ctx, ok := eoc.(context.Context); ok {
		return clues.In(ctx), true
	}

	t.Error("tester can only check error and context.Context values")

	return nil, false
}
