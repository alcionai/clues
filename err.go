package clues

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	return &Err{e: e, msg: msg}
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

// With adds the key,value pair to the Err's data map.
func (err *Err) With(key string, value any) *Err {
	if err == nil {
		return nil
	}

	if len(err.data) == 0 {
		err.data = values{}
	}

	err.data[key] = value
	return err
}

// With adds the key,value pair to the Err's data map.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func With(err error, key string, value any) *Err {
	if err == nil {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = toErr(err, "")
	}

	return e.With(key, value)
}

// WithAll adds every two values as a key,value pair to
// the Err's data map.
func (err *Err) WithAll(kvs ...any) *Err {
	if err == nil {
		return nil
	}

	if len(err.data) == 0 {
		err.data = values{}
	}

	for i := 0; i < len(kvs); i += 2 {
		key := marshal(kvs[i])

		var value any
		if i+1 < len(kvs) {
			value = kvs[i+1]
		}

		err.data[key] = value
	}

	return err
}

// WithAll adds every two values as a key,value pair to
// the Err's data map.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func WithAll(err error, kvs ...any) *Err {
	if err == nil {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = toErr(err, "")
	}

	return e.WithAll(kvs...)
}

// WithMap copies the map to the Err's data map.
func (err *Err) WithMap(m map[string]any) *Err {
	if err == nil {
		return nil
	}

	if len(err.data) == 0 {
		err.data = values{}
	}

	for k, v := range m {
		err.data[k] = v
	}

	return err
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

// Values returns all of the contextual data in the error.  Each
// error in the stack is unwrapped and all maps are unioned.
// In case of collision, lower level error data take least
// priority.
func (err *Err) Values() values {
	if err == nil {
		return values{}
	}

	vals := make(values)

	for _, se := range err.stack {
		for k, v := range InErr(se) {
			vals[k] = v
		}
	}

	for k, v := range InErr(err.e) {
		vals[k] = v
	}

	for k, v := range err.data {
		vals[k] = v
	}

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

func (err *Err) Format(s fmt.State, verb rune) {
	f, ok := err.e.(fmt.Formatter)
	if !ok {
		return
	}

	f.Format(s, verb)
}

// Is overrides the standard Is check for Err.e, allowing us to check
// the conditional for both Err.e and Err.next.  This allows clues to
// Stack() maintain multiple error pointers without failing the otherwise
// linear errors.Is check.
func (err *Err) Is(target error) bool {
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
