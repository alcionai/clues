package cluerr

import (
	"context"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
)

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
// attributes
// ------------------------------------------------------------

// With adds every pair of values as a key,value pair to
// the Err's data map.
func (err *Err) With(kvs ...any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if len(kvs) > 0 {
		err.data = err.data.AddValues(context.Background(), stringify.Normalize(kvs...))
	}

	return err
}

// WithMap copies the map to the Err's data map.
func (err *Err) WithMap(m map[string]any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if len(m) > 0 {
		err.data = err.data.AddValues(context.Background(), stringify.NormalizeMap(m))
	}

	return err
}

// ------------------------------------------------------------
// stacktrace
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
//
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func SkipCaller(err error, depth int) *Err {
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

	_, _, err.file = node.GetDirAndFile(depth + 1)
	err.caller = node.GetCaller(depth + 1)

	return err
}

// NoTrace prevents the error from appearing in the trace stack.
// This is particularly useful for global sentinels that get stacked
// or wrapped into other error cases.
func (err *Err) NoTrace() *Err {
	if isNilErrIface(err) {
		return nil
	}

	err.file = ""
	err.caller = ""

	return err
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newErr generates a new *Err from the parameters.
// traceDepth should always be `1` or `depth+1`.
func newErr(
	e error,
	msg string,
	m map[string]any,
	traceDepth int,
) *Err {
	_, _, file := node.GetDirAndFile(traceDepth + 1)

	return &Err{
		e:      e,
		file:   file,
		caller: node.GetCaller(traceDepth + 1),
		msg:    msg,
		// no ID needed for err data nodes
		data: &node.Node{Values: m},
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
	_, _, file := node.GetDirAndFile(traceDepth + 1)

	return &Err{
		e:      e,
		file:   file,
		caller: node.GetCaller(traceDepth + 1),
		stack:  stack,
		// no ID needed for err nodes
		data: &node.Node{},
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
