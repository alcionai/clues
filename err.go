package clues

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
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
	data values
}

func toErr(e error, msg string) *Err {
	_, file, line, _ := runtime.Caller(2)

	return &Err{
		e:        e,
		location: fmt.Sprintf("%s:%d", file, line),
		msg:      msg,
	}
}

func toStack(e error, stack []error) *Err {
	return &Err{e: e, stack: stack}
}

// ------------------------------------------------------------
// labels
// ------------------------------------------------------------

func (err *Err) HasLabel(label string) bool {
	if err == nil {
		return false
	}

	if _, ok := err.labels[label]; ok {
		return true
	}

	return HasLabel(err.e, label)
}

func HasLabel(err error, label string) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(*Err); ok {
		return e.HasLabel(label)
	}

	return HasLabel(Unwrap(err), label)
}

func (err *Err) Label(label string) *Err {
	if len(err.labels) == 0 {
		err.labels = map[string]struct{}{}
	}

	err.labels[label] = struct{}{}
	return err
}

func Label(err error, label string) *Err {
	if err == nil {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = toErr(err, "")
	}

	return e.Label(label)
}

// ------------------------------------------------------------
// data
// ------------------------------------------------------------

// With adds every pair of values as a key,value pair to
// the Err's data map.
func (err *Err) With(kvs ...any) *Err {
	if err == nil {
		return nil
	}

	if len(err.data) == 0 {
		err.data = values{}
	}

	return &Err{
		e:    err,
		data: err.data.add(normalize(kvs...)),
	}
}

// With adds every two values as a key,value pair to
// the Err's data map.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func With(err error, kvs ...any) *Err {
	if err == nil {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = toErr(err, "")
	}

	return e.With(kvs...)
}

// WithMap copies the map to the Err's data map.
func (err *Err) WithMap(m map[string]any) *Err {
	if err == nil {
		return nil
	}

	if len(err.data) == 0 {
		err.data = values{}
	}

	return &Err{
		e:    err,
		data: err.data.add(m),
	}
}

// WithMap copies the map to the Err's data map.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func WithMap(err error, m map[string]any) *Err {
	if err == nil {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = toErr(err, "")
	}

	return e.WithMap(m)
}

// WithClues is syntactical-sugar that assumes you're using
// the clues package to store structured data in the context.
// The values in the default namespace are retrieved and added
// to the error.
//
// clues.Stack(err).WithClues(ctx) adds the same data as
// clues.Stack(err).WithMap(clues.Values(ctx)).
func (err *Err) WithClues(ctx context.Context) *Err {
	return err.WithMap(In(ctx))
}

// WithClues is syntactical-sugar that assumes you're using
// the clues package to store structured data in the context.
// The values in the default namespace are retrieved and added
// to the error.
//
// clues.WithClues(err, ctx) adds the same data as
// clues.WithMap(err, clues.Values(ctx)).
func WithClues(err error, ctx context.Context) *Err {
	return WithMap(err, In(ctx))
}

// Values returns a copy of all of the contextual data in
// the error.  Each error in the stack is unwrapped and all
// maps are unioned. In case of collision, lower level error
// data take least priority.
func (err *Err) Values() values {
	if err == nil {
		return values{}
	}

	vals := values{}

	for _, se := range err.stack {
		maps.Copy(vals, InErr(se))
	}

	maps.Copy(vals, InErr(err.e))
	maps.Copy(vals, err.data)

	return vals
}

// InErr returns the map of contextual values in the error.
// Each error in the stack is unwrapped and all maps are
// unioned. In case of collision, lower level error data
// take least priority.
func InErr(err error) values {
	if err == nil {
		return values{}
	}

	if e, ok := err.(*Err); ok {
		return e.Values()
	}

	return InErr(Unwrap(err))
}

// ------------------------------------------------------------
// eror interface compliance
// ------------------------------------------------------------

var _ error = &Err{}

func (err *Err) Error() string {
	if err == nil {
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

func format(err error, s fmt.State, verb rune) {
	if err == nil {
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
	if err == nil {
		return
	}

	if verb == 'v' && s.Flag('+') {
		formatPlusV(err, s, verb)
		return
	}

	formatReg(err, s, verb)
}

func write(
	s fmt.State,
	verb rune,
	msgs ...string,
) {
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

// Is overrides the standard Is check for Err.e, allowing us to check
// the conditional for both Err.e and Err.next.  This allows clues to
// Stack() maintain multiple error pointers without failing the otherwise
// linear errors.Is check.
func (err *Err) Is(target error) bool {
	if err == nil {
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
// the conditional for both Err.e and Err.next.  This allows clues to
// Stack() maintain multiple error pointers without failing the otherwise
// linear errors.As check.
func (err *Err) As(target any) bool {
	if err == nil {
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
	if err == nil {
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
	if err == nil {
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

func New(msg string) *Err {
	return toErr(nil, msg)
}

// Wrap returns a  clues.Err with a new message wrapping the old error.
func Wrap(err error, msg string) *Err {
	if err == nil {
		return nil
	}

	return toErr(err, msg)
}

// Stack returns the error as a clues.Err.  If additional errors are
// provided, the entire stack is flattened and returned as a single
// error chain.  All messages and stored structure is aggregated into
// the returned err.
//
// Ex: Stack(sentinel, errors.New("base")).Error() => "sentinel: base"
func Stack(errs ...error) *Err {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return toErr(errs[0], "")
	}

	return toStack(errs[0], errs[1:])
}
