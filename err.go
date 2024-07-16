package clues

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"golang.org/x/exp/maps"
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

	// location is used for printing %+v stack traces
	location string

	// msg is the message for this error.
	msg string

	// labels contains a map of the labels applied
	// to this error.  Can be used to identify error
	// categorization without applying an error type.
	labels map[string]struct{}

	// data is the record of contextual data produced,
	// presumably, at the time the error is created or wrapped.
	data *dataNode
}

// ---------------------------------------------------------------------------
// constructors
// ---------------------------------------------------------------------------

// newErr generates a new *Err from the parameters.
// traceDepth should always be `1` or `depth+1`.
func newErr(
	e error,
	msg string,
	m map[string]any,
	traceDepth int,
) *Err {
	return &Err{
		e:        e,
		location: getTrace(traceDepth + 1),
		msg:      msg,
		data: &dataNode{
			id:     makeNodeID(),
			values: m,
		},
	}
}

// tryExtendErr checks if err is an *Err. If it is, it extends the Err
// with a child containing the provided parameters.  If not, it creates
// a new Err containing the parameters.
// traceDepth should always be `1` or `depth+1`.
func tryExtendErr(
	err error,
	msg string,
	m map[string]any,
	traceDepth int,
) *Err {
	if isNilErrIface(err) {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = newErr(err, msg, m, traceDepth+1)
	}

	return e
}

// newStack creates a new *Err containing the provided stack of errors.
// traceDepth should always be `1` or `depth+1`.
func toStack(
	e error,
	stack []error,
	traceDepth int,
) *Err {
	return &Err{
		e:        e,
		location: getTrace(traceDepth + 1),
		stack:    stack,
		data: &dataNode{
			id:     makeNodeID(),
			values: map[string]any{},
		},
	}
}

// makeStack creates a new *Err from the provided stack of errors.
// nil values are filtered out of the errs slice.  If all errs are nil,
// returns nil.
// traceDepth should always be `1` or `depth+1`.
func makeStack(
	traceDepth int,
	errs ...error,
) *Err {
	filtered := []error{}
	for _, err := range errs {
		if !isNilErrIface(err) {
			filtered = append(filtered, err)
		}
	}

	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return newErr(filtered[0], "", nil, traceDepth+1)
	}

	return toStack(filtered[0], filtered[1:], traceDepth+1)
}

// makeStackWrap creates a new *Err from the provided pair of sentinal
// and wrapped errors.  If sentinel is nil, wraps the wrapped error.
// If wrapped is nil, wraps the sentinel error.  If the message is empty,
// returns a stack(sentinel, wrapped).  Otherwise, makes a stack headed
// by the sentinel error, and wraps the wrapped error in the message.
func makeStackWrap(
	traceDepth int,
	sentinel, wrapped error,
	msg string,
) *Err {
	if isNilErrIface(sentinel) && isNilErrIface(wrapped) {
		return nil
	}

	if len(msg) == 0 {
		return makeStack(traceDepth+1, sentinel, wrapped)
	}

	if isNilErrIface(sentinel) {
		return newErr(wrapped, msg, nil, traceDepth+1)
	}

	if isNilErrIface(wrapped) {
		return newErr(sentinel, msg, nil, traceDepth+1)
	}

	return makeStack(
		1,
		sentinel,
		newErr(wrapped, msg, nil, traceDepth+1))
}

// ------------------------------------------------------------
// getters
// TODO: transform all this to comply with a standard interface
// ------------------------------------------------------------

// ancestors builds out the ancestor lineage of this
// particular error.  This follows standard layout rules
// already established elsewhere:
// * the first entry is the oldest ancestor, the last is
// the current error.
// * Stacked errors get visited before wrapped errors.
//
// TODO: get other ancestor stack builders to work off of this
// func instead of spreading that handling everywhere.
func ancestors(err error) []error {
	return stackAncestorsOntoSelf(err)
}

// a recursive function, purely for building out ancestorStack.
func stackAncestorsOntoSelf(err error) []error {
	if err == nil {
		return []error{}
	}

	errs := []error{}

	ce, ok := err.(*Err)

	if ok {
		for _, e := range ce.stack {
			errs = append(errs, stackAncestorsOntoSelf(e)...)
		}
	}

	unwrapped := Unwrap(err)

	if unwrapped != nil {
		errs = append(errs, stackAncestorsOntoSelf(unwrapped)...)
	}

	errs = append(errs, err)

	return errs
}

// InErr returns the map of contextual values in the error.
// Each error in the stack is unwrapped and all maps are
// unioned. In case of collision, lower level error data
// take least priority.
// TODO: remove this in favor of a type-independent In()
// that returns an interface which both dataNodes and Err
// comply with.
func InErr(err error) *dataNode {
	if isNilErrIface(err) {
		return &dataNode{values: map[string]any{}}
	}

	return &dataNode{values: inErr(err)}
}

func inErr(err error) map[string]any {
	if isNilErrIface(err) {
		return map[string]any{}
	}

	if e, ok := err.(*Err); ok {
		return e.values()
	}

	return inErr(Unwrap(err))
}

// ------------------------------------------------------------
// getters - k:v store
// ------------------------------------------------------------

// Values returns a copy of all of the contextual data in
// the error.  Each error in the stack is unwrapped and all
// maps are unioned. In case of collision, lower level error
// data take least priority.
func (err *Err) Values() *dataNode {
	if isNilErrIface(err) {
		return &dataNode{values: map[string]any{}}
	}

	return &dataNode{values: err.values()}
}

func (err *Err) values() map[string]any {
	if isNilErrIface(err) {
		return map[string]any{}
	}

	vals := map[string]any{}
	maps.Copy(vals, err.data.Map())
	maps.Copy(vals, inErr(err.e))

	for _, se := range err.stack {
		maps.Copy(vals, inErr(se))
	}

	return vals
}

// ------------------------------------------------------------
// getters - labels
// ------------------------------------------------------------

func (err *Err) HasLabel(label string) bool {
	if isNilErrIface(err) {
		return false
	}

	// Check all labels in the error and it's stack since the stack isn't
	// traversed separately. If we don't check the stacked labels here we'll skip
	// checking them completely.
	if _, ok := err.Labels()[label]; ok {
		return true
	}

	return HasLabel(err.e, label)
}

func HasLabel(err error, label string) bool {
	if isNilErrIface(err) {
		return false
	}

	if e, ok := err.(*Err); ok {
		return e.HasLabel(label)
	}

	return HasLabel(Unwrap(err), label)
}

func (err *Err) Label(labels ...string) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if len(err.labels) == 0 {
		err.labels = map[string]struct{}{}
	}

	lc := getLabelCounter(err)
	els := err.Labels()

	for _, label := range labels {
		if lc != nil {
			_, inPrior := els[label]
			_, inCurrent := err.labels[label]
			if !inPrior && !inCurrent {
				lc.Add(label, 1)
			}
		}
		// don't duplicate counts

		err.labels[label] = struct{}{}
	}

	return err
}

func Label(err error, label string) *Err {
	return tryExtendErr(err, "", nil, 1).Label(label)
}

func (err *Err) Labels() map[string]struct{} {
	if isNilErrIface(err) {
		return map[string]struct{}{}
	}

	labels := map[string]struct{}{}

	for _, se := range err.stack {
		maps.Copy(labels, Labels(se))
	}

	if err.e != nil {
		maps.Copy(labels, Labels(err.e))
	}

	maps.Copy(labels, err.labels)

	return labels
}

func Labels(err error) map[string]struct{} {
	for err != nil {
		e, ok := err.(*Err)
		if ok {
			return e.Labels()
		}

		err = Unwrap(err)
	}

	return map[string]struct{}{}
}

// ------------------------------------------------------------
// getters - comments
// ------------------------------------------------------------

// Comments retrieves all comments in the error.
func (err *Err) Comments() comments {
	return Comments(err)
}

// Comments retrieves all comments in the error.
func Comments(err error) comments {
	if isNilErrIface(err) {
		return comments{}
	}

	ancs := ancestors(err)
	result := comments{}

	for _, ancestor := range ancs {
		ce, ok := ancestor.(*Err)
		if !ok {
			continue
		}

		result = append(result, ce.data.Comments()...)
	}

	return result
}

// ------------------------------------------------------------
// eror interface compliance and stringers
// ------------------------------------------------------------

var _ error = &Err{}

// Error allows Err to be used as a standard error interface.
func (err *Err) Error() string {
	if isNilErrIface(err) {
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
	if isNilErrIface(err) {
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
	if isNilErrIface(err) {
		return
	}

	write(s, verb, err.msg)

	if len(err.msg) > 0 && err.e != nil {
		io.WriteString(s, ": ")
	}

	format(err.e, s, verb)

	if (len(err.msg) > 0 || err.e != nil) && len(err.stack) > 0 {
		io.WriteString(s, ": ")
	}

	for _, e := range err.stack {
		format(e, s, verb)
	}
}

// in %+v formatting, we output errors FIFO (ie, read from the
// bottom of the stack first).
func formatPlusV(err *Err, s fmt.State, verb rune) {
	if isNilErrIface(err) {
		return
	}

	for i := len(err.stack) - 1; i >= 0; i-- {
		e := err.stack[i]
		format(e, s, verb)
	}

	if len(err.stack) > 0 && err.e != nil {
		io.WriteString(s, "\n")
	}

	format(err.e, s, verb)

	if (len(err.stack) > 0 || err.e != nil) && len(err.msg) > 0 {
		io.WriteString(s, "\n")
	}

	write(s, verb, err.msg)
	write(s, verb, "\n\t%s", err.location)
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
	if isNilErrIface(err) {
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
				io.WriteString(s, msgs[0])
			} else if len(msgs[1]) > 0 {
				fmt.Fprintf(s, msgs[0], msgs[1])
			}
			return
		}

		fallthrough

	case 's':
		io.WriteString(s, msgs[0])

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
	if isNilErrIface(err) {
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
	if isNilErrIface(err) {
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
	if isNilErrIface(err) {
		return nil
	}

	return err.e
}

// Unwrap provides compatibility for Go 1.13 error chains.
// Unwrap returns the Unwrap()ped base error, if it implements
// the unwrapper interface:
//
//	type unwrapper interface {
//	       Unwrap() error
//	}
//
// If the error does not implement Unwrap, returns the error.
func Unwrap(err error) error {
	if isNilErrIface(err) {
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
// constructors
// ------------------------------------------------------------

// New creates an *Err with the provided Msg.
//
// If you have a `ctx` containing other clues data, it is recommended
// that you call `NewWC(ctx, msg)` to ensure that data gets added to
// the error.
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).
func New(msg string) *Err {
	return newErr(nil, msg, nil, 1)
}

// NewWC creates an *Err with the provided Msg, and additionally
// extracts all of the clues data in the context into the error.
//
// NewWC is equivalent to clues.New("msg").WithClues(ctx).
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).
func NewWC(ctx context.Context, msg string) *Err {
	return newErr(nil, msg, nil, 1).WithClues(ctx)
}

// Wrap extends an error with the provided message.  It is a replacement
// for `errors.Wrap`, and complies with all golang unwrapping behavior.
//
// If you have a `ctx` containing other clues data, it is recommended
// that you call `WrapWC(ctx, err, msg)` to ensure that data gets added to
// the error.
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).  There is
// no Wrapf func in clues; we prefer that callers use Wrap().With()
// instead.
//
// Wrap can be given a `nil` error value, and will return a nil *Err.
// To avoid golang footguns when returning nil structs as interfaces
// (such as error), callers should always return Wrap().OrNil() in cases
// where the input error could be nil.
func Wrap(err error, msg string) *Err {
	if isNilErrIface(err) {
		return nil
	}

	return newErr(err, msg, nil, 1)
}

// WrapWC extends an error with the provided message.  It is a replacement
// for `errors.Wrap`, and complies with all golang unwrapping behavior.
//
// WrapWC is equivalent to clues.Wrap(err, "msg").WithClues(ctx).
//
// If you have a `ctx` containing other clues data, it is recommended
// that you call `WrapWC(ctx, err, msg)` to ensure that data gets added to
// the error.
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).  There is
// no WrapWCf func in clues; we prefer that callers use WrapWC().With()
// instead.
//
// Wrap can be given a `nil` error value, and will return a nil *Err.
// To avoid golang footguns when returning nil structs as interfaces
// (such as error), callers should always return WrapWC().OrNil() in cases
// where the input error could be nil.
func WrapWC(ctx context.Context, err error, msg string) *Err {
	if isNilErrIface(err) {
		return nil
	}

	return newErr(err, msg, nil, 1).WithClues(ctx)
}

// Stack composes a stack of one or more errors.  The first message in the
// parameters is considered the "most recent".  Ex: a construction like
// clues.Stack(errFoo, io.EOF, errSmarf), the resulting Error message would
// be "foo: end-of-file: smarf".
//
// Unwrapping a Stack follows the same order.  This allows callers to inject
// sentinel errors into error chains (ex: clues.Stack(io.EOF, myErr))  without
// losing errors.Is or errors.As checks on lower errors.
//
// If given a single error, Stack acts as a thin wrapper around the error to
// provide an *Err, giving the caller access to all the builder funcs and error
// tracing.  It is always recommended that callers `return clues.Stack(err)`
// instead of the plain `return err`.
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).
//
// Stack can be given one or more `nil` error values.  Nil errors will be
// automatically filtered from the retained stack of errors.  Ex:
// clues.Stack(errFoo, nil, errSmarf) == clues.Stack(errFoo, errSmarf).
// If all input errors are nil, stack will return nil.  To avoid golang
// footguns when returning nil structs as interfaces (such as error), callers
// should always return Stack().OrNil() in cases where the input error could
// be nil.
func Stack(errs ...error) *Err {
	return makeStack(1, errs...)
}

// StackWC composes a stack of one or more errors.  The first message in the
// parameters is considered the "most recent".  Ex: a construction like
// clues.StackWC(errFoo, io.EOF, errSmarf), the resulting Error message would
// be "foo: end-of-file: smarf".
//
// Unwrapping a Stack follows the same order.  This allows callers to inject
// sentinel errors into error chains (ex: clues.StackWC(io.EOF, myErr))  without
// losing errors.Is or errors.As checks on lower errors.
//
// If given a single error, Stack acts as a thin wrapper around the error to
// provide an *Err, giving the caller access to all the builder funcs and error
// tracing.  It is always recommended that callers `return clues.StackWC(err)`
// instead of the plain `return err`.
//
// StackWC is equivalent to clues.Stack(errs...).WithClues(ctx)
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).
//
// Stack can be given one or more `nil` error values.  Nil errors will be
// automatically filtered from the retained stack of errors.  Ex:
// clues.StackWC(ctx, errFoo, nil, errSmarf) == clues.StackWC(ctx, errFoo, errSmarf).
// If all input errors are nil, stack will return nil.  To avoid golang
// footguns when returning nil structs as interfaces (such as error), callers
// should always return StackWC().OrNil() in cases where the input error could
// be nil.
func StackWC(ctx context.Context, errs ...error) *Err {
	err := makeStack(1, errs...)

	if isNilErrIface(err) {
		return nil
	}

	return err.WithClues(ctx)
}

// StackWrap is a quality-of-life shorthand for a common usage of clues errors:
// clues.Stack(sentinel, clues.Wrap(myErr, "my message")).  The result follows
// all standard behavior of stacked and wrapped errors.
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).
//
// StackWrap can be given one or more `nil` error values.  Nil errors will be
// automatically filtered from the retained stack of errors.  Ex:
// clues.StackWrap(errFoo, nil, "msg") == clues.Wrap(errFoo, "msg").
// If both input errors are nil, StackWrap will return nil.  To avoid golang
// footguns when returning nil structs as interfaces (such as error), callers
// should always return StackWrap().OrNil() in cases where the input errors
// could be nil.
func StackWrap(sentinel, wrapped error, msg string) *Err {
	return makeStackWrap(1, sentinel, wrapped, msg)
}

// StackWrapWC is a quality-of-life shorthand for a common usage of clues errors:
// clues.Stack(sentinel, clues.Wrap(myErr, "my message")).WithClues(ctx).
// The result follows all standard behavior of stacked and wrapped errors.
//
// The returned *Err is an error-compliant builder that can aggregate
// additional data using funcs like With(...) or Label(...).
//
// StackWrapWC can be given one or more `nil` error values.  Nil errors will be
// automatically filtered from the retained stack of errors.  Ex:
// clues.StackWrapWC(ctx, errFoo, nil, "msg") == clues.WrapWC(ctx, errFoo, "msg").
// If both input errors are nil, StackWrap will return nil.  To avoid golang
// footguns when returning nil structs as interfaces (such as error), callers
// should always return StackWrap().OrNil() in cases where the input errors
// could be nil.
func StackWrapWC(
	ctx context.Context,
	sentinel, wrapped error,
	msg string,
) *Err {
	err := makeStackWrap(1, sentinel, wrapped, msg)

	if isNilErrIface(err) {
		return nil
	}

	return err.WithClues(ctx)
}

// OrNil is a workaround for golang's infamous "an interface
// holding a nil value is not nil" gotcha.  You should use it
// to ensure the error value to produce is properly nil whenever
// your wrapped or stacked error values could also possibly be
// nil.
//
// ie:
// ```
// return clues.Stack(maybeNilErrValue).OrNil()
// // or
// return clues.Wrap(maybeNilErrValue, "msg").OrNil()
// ```
func (err *Err) OrNil() error {
	if isNilErrIface(err) {
		return nil
	}

	return err
}

// ------------------------------------------------------------
// builders - clues ctx
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
	if isNilErrIface(err) {
		return nil
	}

	dn := In(ctx)
	e := err.WithMap(dn.Map())
	e.data.labelCounter = dn.labelCounter

	return e
}

// WithClues is syntactical-sugar that assumes you're using
// the clues package to store structured data in the context.
// The values in the default namespace are retrieved and added
// to the error.
//
// clues.WithClues(err, ctx) adds the same data as
// clues.WithMap(err, clues.Values(ctx)).
//
// If the context contains a clues LabelCounter, that counter is
// passed to the error.  WithClues must always be called first in
// order to count labels.
func WithClues(err error, ctx context.Context) *Err {
	if isNilErrIface(err) {
		return nil
	}

	return WithMap(err, In(ctx).Map())
}

// ------------------------------------------------------------
// builders - k:v store
// ------------------------------------------------------------

// With adds every pair of values as a key,value pair to
// the Err's data map.
func (err *Err) With(kvs ...any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if len(kvs) > 0 {
		err.data = err.data.addValues(normalize(kvs...))
	}

	return err
}

// With adds every two values as a key,value pair to
// the Err's data map.
// If err is not an *Err intance, a new *Err is generated
// containing the original err.
func With(err error, kvs ...any) *Err {
	return tryExtendErr(err, "", nil, 1).With(kvs...)
}

// WithMap copies the map to the Err's data map.
func (err *Err) WithMap(m map[string]any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if len(m) > 0 {
		err.data = err.data.addValues(m)
	}

	return err
}

// WithMap copies the map to the Err's data map.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func WithMap(err error, m map[string]any) *Err {
	return tryExtendErr(err, "", m, 1).WithMap(m)
}

// ------------------------------------------------------------
// builders - tracing
// ------------------------------------------------------------

// SkipCaller skips <depth> callers when constructing the
// error trace stack.  The caller is the file, line, and func
// where the *clues.Err was generated.
//
// A depth of 0 performs no skips, and returns the same
// caller info as if SkipCaller was not called.  1 skips the
// immediate parent, etc.
//
// Error traces are already generated for the location where
// clues.Wrap or clues.Stack was called.  This func is for
// cases where Wrap or Stack calls are handled in a helper
// func and are not reporting the actual error origin.
func (err *Err) SkipCaller(depth int) *Err {
	if isNilErrIface(err) {
		return nil
	}

	// needed both here and in withTrace() to
	// correct for the extra call depth.
	if depth < 0 {
		depth = 0
	}

	err.location = getTrace(depth + 1)

	return err
}

// SkipCaller skips <depth> callers when constructing the
// error trace stack.  The caller is the file, line, and func
// where the *clues.Err was generated.
//
// A depth of 0 performs no skips, and returns the same
// caller info as if SkipCaller was not called.  1 skips the
// immediate parent, etc.
//
// Error traces are already generated for the location where
// clues.Wrap or clues.Stack was called.  This func is for
// cases where Wrap or Stack calls are handled in a helper
// func and are not reporting the actual error origin.
//
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func WithSkipCaller(err error, depth int) *Err {
	if isNilErrIface(err) {
		return nil
	}

	// needed both here and in withTrace() to
	// correct for the extra call depth.
	if depth < 0 {
		depth = 0
	}

	e, ok := err.(*Err)
	if !ok {
		return newErr(err, "", map[string]any{}, depth+1)
	}

	return e.SkipCaller(depth + 1)
}

// ------------------------------------------------------------
// builders - comments
// ------------------------------------------------------------

// Comments are special case additions to the error.  They're here to, well,
// let you add comments!  Why?  Because sometimes it's not sufficient to have
// an error message describe what that error really means. Even a bunch of
// clues  to describe system state may not be enough.  Sometimes what you need
// in order to debug the situation is a long-form explanation (you do already
// add that to your code, don't you?).  Or, even better, a linear history of
// long-form explanations, each one building on the prior (which you can't
// easily do in code).
//
// Unlike other additions, which are added as top-level key:value pairs to the
// context, the whole history of comments gets retained, persisted in order of
// appearance and prefixed by the file and line in which they appeared. This
// means comments are always added to the error and never clobber each other,
// regardless of their location.
func (err *Err) WithComment(msg string, vs ...any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	return &Err{
		e: err,
		// have to do a new dataNode here, or else comments will duplicate
		data: &dataNode{comment: newComment(1, msg, vs...)},
	}
}

// Comments are special case additions to the error.  They're here to, well,
// let you add comments!  Why?  Because sometimes it's not sufficient to have
// an error message describe what that error really means. Even a bunch of
// clues  to describe system state may not be enough.  Sometimes what you need
// in order to debug the situation is a long-form explanation (you do already
// add that to your code, don't you?).  Or, even better, a linear history of
// long-form explanations, each one building on the prior (which you can't
// easily do in code).
//
// Unlike other additions, which are added as top-level key:value pairs to the
// context, the whole history of comments gets retained, persisted in order of
// appearance and prefixed by the file and line in which they appeared. This
// means comments are always added to the error and never clobber each other,
// regardless of their location.
func WithComment(err error, msg string, vs ...any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	return &Err{
		e: err,
		// have to do a new dataNode here, or else comments will duplicate
		data: &dataNode{comment: newComment(1, msg, vs...)},
	}
}

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

// getLabelCounter retrieves the a labelCounter from the provided
// error.  The algorithm works from the current error up the
// hierarchy, looking into each dataNode tree along the way, and
// eagerly takes the first available counter.
func getLabelCounter(e error) Adder {
	if e == nil {
		return nil
	}

	ce, ok := e.(*Err)
	if !ok {
		return nil
	}

	for i := len(ce.stack) - 1; i >= 0; i-- {
		lc := getLabelCounter(ce.stack[i])
		if lc != nil {
			return lc
		}
	}

	if ce.e != nil {
		lc := getLabelCounter(ce.e)
		if lc != nil {
			return lc
		}
	}

	if ce.data != nil && ce.data.labelCounter != nil {
		return ce.data.labelCounter
	}

	return nil
}

// returns true if the error is nil, or if it is a non-nil interface
// containing a nil value.
func isNilErrIface(err error) bool {
	if err == nil {
		return true
	}

	val := reflect.ValueOf(err)

	return ((val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface) && val.IsNil())
}
