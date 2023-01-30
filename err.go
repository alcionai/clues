package clues

import (
	"context"
	"fmt"
)

// Err augments an error with labels (a categorization system) and
// data (a map of contextual data used to record the state of the
// process at the time the error occurred, primarily for use in
// upstream logging and other telemetry),
type Err struct {
	// e holds the base error.
	e error

	// msg is the message for this error.
	msg string

	// labels contains a map of the labels applied
	// to this error.  Can be used to identify error
	// categorization without applying an error type.
	labels map[string]struct{}

	// data is the record of contextual data produced,
	// presumably, at the time the error is created or wrapped.
	data map[string]any
}

func asErr(e error, msg string) *Err {
	return &Err{e: e, msg: msg}
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
		e = asErr(err, "")
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
		err.data = map[string]any{}
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
		e = asErr(err, "")
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
		err.data = map[string]any{}
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
		e = asErr(err, "")
	}

	return e.WithAll(kvs...)
}

// WithMap copies the map to the Err's data map.
func (err *Err) WithMap(m map[string]any) *Err {
	if err == nil {
		return nil
	}

	if len(err.data) == 0 {
		err.data = map[string]any{}
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
		e = asErr(err, "")
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
	return err.WithMap(Values(ctx))
}

// WithClues is syntactical-sugar that assumes you're using
// the clues package to store structured data in the context.
// The values in the default namespace are retrieved and added
// to the error.
//
// clues.WithClues(err, ctx) adds the same data as
// clues.WithMap(err, clues.Values(ctx)).
func WithClues(err error, ctx context.Context) *Err {
	return WithMap(err, Values(ctx))
}

// Values returns all of the contextual data in the error.  Each
// error in the stack is unwrapped and all maps are unioned.
// In case of collision, lower level error data take least
// priority.
func (err *Err) Values() map[string]any {
	if err == nil {
		return map[string]any{}
	}

	vals := ErrValues(err.e)

	for k, v := range err.data {
		vals[k] = v
	}

	return vals
}

// ErrValues returns all of the contextual data in the error.
// Each error in the stack is unwrapped and all maps are
// unioned. In case of collision, lower level error data
// take least priority.
func ErrValues(err error) map[string]any {
	if err == nil {
		return map[string]any{}
	}

	if e, ok := err.(*Err); ok {
		return e.Values()
	}

	return ErrValues(Unwrap(err))
}

// ------------------------------------------------------------
// eror interface compliance
// ------------------------------------------------------------

var _ error = &Err{}

func (err *Err) Error() string {
	if len(err.msg) == 0 {
		return err.e.Error()
	}
	return err.msg + ": " + err.e.Error()
}

func (err *Err) Format(s fmt.State, verb rune) {
	f, ok := err.e.(fmt.Formatter)
	if !ok {
		return
	}

	f.Format(s, verb)
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
	return asErr(nil, msg)
}

// Wrap returns a  clues.Err with a new message wrapping the old error.
func Wrap(err error, msg string) *Err {
	if err == nil {
		return nil
	}

	return asErr(err, msg)
}

// Stack returns a clues.Err holding the error..
func Stack(err error) *Err {
	if err == nil {
		return nil
	}

	return asErr(err, "")
}
