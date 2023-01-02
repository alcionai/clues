package clues

import (
	"fmt"

	"github.com/pkg/errors"
)

// Err augments an error with labels (a categorization system) and
// data (a map of contextual data used to record the state of the
// process at the time the error occurred, primarily for use in
// upstream logging and other telemetry),
type Err struct {
	// e holds the base error.
	e error

	// labels contains a map of the labels applied
	// to this error.  Can be used to identify error
	// categorization without applying an error type.
	labels map[string]struct{}

	// data is the record of contextual data produced,
	// presumably, at the time the error is created or wrapped.
	data map[string]any
}

func newErr(e error) *Err {
	return &Err{e: e}
}

// ------------------------------------------------------------
// labels
// ------------------------------------------------------------

func (err *Err) HasLabel(label string) bool {
	if err == nil {
		return false
	}

	_, ok := err.labels[label]
	return ok
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
		e = newErr(err)
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
		e = newErr(err)
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
		e = newErr(err)
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
		e = newErr(err)
	}

	return e.WithMap(m)
}

// Values returns all of the contextual data in the error.  Each
// error in the stack is unwrapped and all maps are unioned.
// In case of collision, lower level error data take least
// priority.
func (err *Err) Values() map[string]any {
	if err == nil {
		return map[string]any{}
	}

	child := err.Unwrap()
	vals := ErrValues(child)

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

	e, ok := err.(*Err)
	if !ok {
		return map[string]any{}
	}

	return e.Values()
}

// ------------------------------------------------------------
// eror interface compliance
// ------------------------------------------------------------

func (err *Err) Error() string {
	return err.e.Error()
}

func (err *Err) Format(s fmt.State, verb rune) {
	f, ok := err.e.(fmt.Formatter)
	if !ok {
		return
	}
	f.Format(s, verb)
}

// Cause returns the Cause() of the base error, if it implements
// the causer interface:
//
//	type causer interface {
//	       Cause() error
//	}
//
// If the error does not implement Cause, returns the base error.
func (err *Err) Cause() error {
	if err.e == nil {
		return nil
	}

	f, ok := err.e.(interface{ Cause() error })
	if !ok {
		return err.e
	}
	return f.Cause()
}

// Unwrap provides compatibility for Go 1.13 error chains.
// Unwrap returns the Unwrap()ped base error, if it implements
// the unwrapper interface:
//
//	type unwrapper interface {
//	       Unwrap() error
//	}
//
// If the error does not implement Unwrap, returns the base error
func (err *Err) Unwrap() error {
	if err.e == nil {
		return nil
	}

	f, ok := err.e.(interface{ Unwrap() error })
	if !ok {
		return err.e
	}

	return f.Unwrap()
}

// ------------------------------------------------------------
// constructors
// ------------------------------------------------------------

func New(msg string) *Err {
	return newErr(errors.New(msg))
}

func Newf(template string, values ...any) *Err {
	return newErr(errors.Errorf(template, values...))
}

func Wrap(err error, msg string) *Err {
	if err == nil {
		return nil
	}
	return newErr(errors.Wrap(err, msg))
}

func Wrapf(err error, template string, values ...any) *Err {
	if err == nil {
		return nil
	}
	return newErr(errors.Wrapf(err, template, values...))
}

func WithStack(err error) *Err {
	if err == nil {
		return nil
	}
	return newErr(errors.WithStack(err))
}
