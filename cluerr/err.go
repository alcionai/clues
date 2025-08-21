package cluerr

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/alcionai/clues/internal/errs"
	"github.com/alcionai/clues/internal/node"
)

// Err augments an error with labels (a categorization system) and
// data (a map of contextual data used to record the state of the
// process at the time the error occurred, primarily for use in
// upstream logging and other telemetry),
type Err struct {
	// e holds the base error.
	e error

	// stack is a chain of errors that this error is stacked onto.
	// stacks may contain other stacks, forming a tree.
	// Funcs that examine or flatten the tree will walk its structure
	// in pre-order traversal.
	stack []error

	// the name of the file where the caller func is found.
	file string
	// the name of the func where the error (or wrapper) was generated.
	caller string

	// msg is the message for this error.
	msg string

	// labels contains a map of the labels applied
	// to this error.  Can be used to identify error
	// categorization without applying an error type.
	labels map[string]struct{}

	// data is the record of contextual data produced,
	// presumably, at the time the error is created or wrapped.
	data *node.Node
}

// Node retrieves the node values from the error.
func (err *Err) Node() *node.Node {
	if errs.IsNilIface(err) {
		return &node.Node{}
	}

	return err.Values()
}

// ------------------------------------------------------------
// tree operations
// ------------------------------------------------------------

// ancestors builds out the ancestor lineage of this
// particular error.  This follows standard layout rules
// already established elsewhere:
// * the first entry is the oldest ancestor, the last is
// the current error.
// * Stacked errors get visited before wrapped errors.
func ancestors(err error) []error {
	return stackAncestorsOntoSelf(err)
}

// a recursive function, purely for building out ancestorStack.
func stackAncestorsOntoSelf(err error) []error {
	if err == nil {
		return []error{}
	}

	errStack := []error{}

	ce, ok := err.(*Err)

	if ok {
		for _, e := range ce.stack {
			errStack = append(errStack, stackAncestorsOntoSelf(e)...)
		}
	}

	unwrapped := unwrap(err)
	if unwrapped != nil {
		errStack = append(errStack, stackAncestorsOntoSelf(unwrapped)...)
	}

	errStack = append(errStack, err)

	return errStack
}

// ------------------------------------------------------------
// eror interface compliance and stringers
// ------------------------------------------------------------

var _ error = &Err{}

// Error allows Err to be used as a standard error interface.
func (err *Err) Error() string {
	if errs.IsNilIface(err) {
		return "<nil>"
	}

	msg := []string{}

	if len(err.msg) > 0 {
		msg = append(msg, err.msg)
	}

	if err.e != nil {
		msg = append(msg, err.e.Error())
	}

	for _, se := range err.stack {
		msg = append(msg, se.Error())
	}

	return strings.Join(msg, ": ")
}

// format is the fallback formatting of an error
func format(err error, s fmt.State, verb rune) {
	if errs.IsNilIface(err) {
		return
	}

	f, ok := err.(fmt.Formatter)
	if ok {
		f.Format(s, verb)
	} else {
		write(s, verb, err.Error())
	}
}

// For all formatting besides %+v, the error printout should closely
// mimic that of err.Error().
func formatReg(err *Err, s fmt.State, verb rune) {
	if errs.IsNilIface(err) {
		return
	}

	write(s, verb, err.msg)

	if len(err.msg) > 0 && err.e != nil {
		fmt.Fprint(s, ": ")
	}

	format(err.e, s, verb)

	if (len(err.msg) > 0 || err.e != nil) && len(err.stack) > 0 {
		fmt.Fprint(s, ": ")
	}

	for _, e := range err.stack {
		format(e, s, verb)
	}
}

// in %+v formatting, we output errors FIFO (ie, read from the
// bottom of the stack first).
func formatPlusV(err *Err, s fmt.State, verb rune) {
	if errs.IsNilIface(err) {
		return
	}

	for i := len(err.stack) - 1; i >= 0; i-- {
		e := err.stack[i]
		format(e, s, verb)
	}

	if len(err.stack) > 0 && err.e != nil {
		fmt.Fprint(s, "\n")
	}

	format(err.e, s, verb)

	if (len(err.stack) > 0 || err.e != nil) && len(err.msg) > 0 {
		fmt.Fprint(s, "\n")
	}

	write(s, verb, err.msg)

	parts := []string{}
	if len(err.caller) > 0 {
		parts = append(parts, err.caller)
	}

	if len(err.file) > 0 {
		parts = append(parts, err.file)
	}

	write(s, verb, "\n\t%s", strings.Join(parts, " - "))
}

// Format ensures stack traces are printed appropariately.
//
//	%s    same as err.Error()
//	%v    equivalent to %s
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+v   Prints filename, function, and line number for each error in the stack.
func (err *Err) Format(s fmt.State, verb rune) {
	if errs.IsNilIface(err) {
		return
	}

	if verb == 'v' && s.Flag('+') {
		formatPlusV(err, s, verb)
		return
	}

	formatReg(err, s, verb)
}

func write(s fmt.State, verb rune, msgs ...string) {
	if len(msgs) == 0 || len(msgs[0]) == 0 {
		return
	}

	switch verb {
	case 'v':
		if s.Flag('+') {
			if len(msgs) == 1 {
				fmt.Fprint(s, msgs[0])
			} else if len(msgs[1]) > 0 {
				fmt.Fprintf(s, msgs[0], msgs[1])
			}

			return
		}

		fallthrough

	case 's':
		fmt.Fprint(s, msgs[0])

	case 'q':
		fmt.Fprintf(s, "%q", msgs[0])
	}
}

// ------------------------------------------------------------
// common interface compliance
// ------------------------------------------------------------

// Is overrides the standard Is check for Err.e, allowing us to check
// the conditional for both Err.e and Err.stack.  This allows clues to
// Stack() multiple error pointers without failing the otherwise linear
// errors.Is check.
func (err *Err) Is(target error) bool {
	if errs.IsNilIface(err) {
		return false
	}

	if errors.Is(err.e, target) {
		return true
	}

	for _, se := range err.stack {
		if errors.Is(se, target) {
			return true
		}
	}

	return false
}

// As overrides the standard As check for Err.e, allowing us to check
// the conditional for both Err.e and Err.stack.  This allows clues to
// Stack() multiple error pointers without failing the otherwise linear
// errors.As check.
func (err *Err) As(target any) bool {
	if errs.IsNilIface(err) {
		return false
	}

	if errors.As(err.e, target) {
		return true
	}

	for _, se := range err.stack {
		if errors.As(se, target) {
			return true
		}
	}

	return false
}

// Unwrap provides compatibility for Go 1.13 error chains.
// Unwrap returns the Unwrap()ped base error, if it implements
// the unwrapper interface:
//
//	type unwrapper interface {
//	       Unwrap() error
//	}
//
// If the error does not implement Unwrap, returns the base error.
func (err *Err) Unwrap() error {
	if errs.IsNilIface(err) {
		return nil
	}

	return err.e
}

// unwrap attempts to unwrap any generic error.
func unwrap(err error) error {
	if errs.IsNilIface(err) {
		return nil
	}

	if e, ok := err.(*Err); ok {
		return e.e
	}

	u, ok := err.(interface{ Unwrap() error })
	if !ok {
		return nil
	}

	ue := u.Unwrap()

	return ue
}

// ------------------------------------------------------------
// nodes and node attributes
// ------------------------------------------------------------

// WithClues is syntactical-sugar that assumes you're using
// the clues package to store structured data in the context.
// The values in the default namespace are retrieved and added
// to the error.
//
// clues.Stack(err).WithClues(ctx) adds the same data as
// clues.Stack(err).WithMap(clues.Values(ctx)).
//
// If the context contains a clues LabelCounter, that counter is
// passed to the error.  WithClues must always be called first in
// order to count labels.
func (err *Err) WithClues(ctx context.Context) *Err {
	if errs.IsNilIface(err) {
		return nil
	}

	dn := node.FromCtx(ctx)
	e := err.WithMap(dn.Map())

	return e
}

// CluesIn returns the structured data in the error.
// Each error in the stack is unwrapped and all maps are
// unioned. In case of collision, lower level error data
// take least priority.
func CluesIn(err error) *node.Node {
	if errs.IsNilIface(err) {
		return &node.Node{}
	}

	return &node.Node{Values: cluesIn(err)}
}

func cluesIn(err error) map[string]any {
	if errs.IsNilIface(err) {
		return map[string]any{}
	}

	if e, ok := err.(*Err); ok {
		return e.values()
	}

	return cluesIn(unwrap(err))
}

// Values returns a copy of all of the contextual data in
// the error.  Each error in the stack is unwrapped and all
// maps are unioned. In case of collision, lower level error
// data take least priority.
func (err *Err) Values() *node.Node {
	if errs.IsNilIface(err) {
		return &node.Node{}
	}

	return &node.Node{Values: err.values()}
}

func (err *Err) values() map[string]any {
	if errs.IsNilIface(err) {
		return map[string]any{}
	}

	vals := map[string]any{}
	maps.Copy(vals, err.data.Map())
	maps.Copy(vals, cluesIn(err.e))

	for _, se := range err.stack {
		maps.Copy(vals, cluesIn(se))
	}

	return vals
}
