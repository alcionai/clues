package clues

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
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
	data *dataNode
}

func asErr(err error, msg string, m map[string]any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = toErr(err, msg, m)
	}

	return e
}

func toErr(e error, msg string, m map[string]any) *Err {
	return &Err{
		e:        e,
		location: getTrace(3),
		msg:      msg,
		data:     &dataNode{id: makeNodeID(), vs: m},
	}
}

func toStack(e error, stack []error) *Err {
	return &Err{
		e:        e,
		location: getTrace(3),
		stack:    stack,
		data:     &dataNode{id: makeNodeID(), vs: map[string]any{}},
	}
}

func getTrace(depth int) string {
	_, file, line, _ := runtime.Caller(depth)
	return fmt.Sprintf("%s:%d", file, line)
}

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

// ------------------------------------------------------------
// labels
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
	return asErr(err, "", nil).Label(label)
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
// data
// ------------------------------------------------------------

// With adds every pair of values as a key,value pair to
// the Err's data map.
func (err *Err) With(kvs ...any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if len(kvs) > 0 {
		err.data = err.data.add(normalize(kvs...))
	}

	return err
}

// With adds every two values as a key,value pair to
// the Err's data map.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func With(err error, kvs ...any) *Err {
	return asErr(err, "", nil).With(kvs...)
}

// WithTrace sets the error trace to a certain depth.
// A depth of 0 traces to the func where WithTrace is
// called.  1 sets the trace to its parent, etc.
// Error traces are already generated for the location
// where clues.Wrap or clues.Stack was called.  This
// call is for cases where Wrap or Stack calls are handled
// in a helper func and are not reporting the actual
// error origin.
func (err *Err) WithTrace(depth int) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if depth < 0 {
		depth = 0
	}

	err.location = getTrace(depth + 2)

	return err
}

// WithTrace sets the error trace to a certain depth.
// A depth of 0 traces to the func where WithTrace is
// called.  1 sets the trace to its parent, etc.
// Error traces are already generated for the location
// where clues.Wrap or clues.Stack was called.  This
// call is for cases where Wrap or Stack calls are handled
// in a helper func and are not reporting the actual
// error origin.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func WithTrace(err error, depth int) *Err {
	if isNilErrIface(err) {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		return toErr(err, "", map[string]any{})
	}

	// needed both here and in withTrace() to
	// correct for the extra call depth.
	if depth < 0 {
		depth = 0
	}

	return e.WithTrace(depth + 1)
}

// WithMap copies the map to the Err's data map.
func (err *Err) WithMap(m map[string]any) *Err {
	if isNilErrIface(err) {
		return nil
	}

	if len(m) > 0 {
		err.data = err.data.add(m)
	}

	return err
}

// WithMap copies the map to the Err's data map.
// If err is not an *Err intance, returns the error wrapped
// into an *Err struct.
func WithMap(err error, m map[string]any) *Err {
	return asErr(err, "", m).WithMap(m)
}

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

// OrNil is a workaround for golang's infamous "an interface
// holding a nil value is not nil" gotcha.  You can use it at
// the end of error formatting chains to ensure a correct nil
// return value.
func (err *Err) OrNil() error {
	if isNilErrIface(err) {
		return nil
	}

	return err
}

// Values returns a copy of all of the contextual data in
// the error.  Each error in the stack is unwrapped and all
// maps are unioned. In case of collision, lower level error
// data take least priority.
func (err *Err) Values() *dataNode {
	if isNilErrIface(err) {
		return &dataNode{vs: map[string]any{}}
	}

	return &dataNode{vs: err.values()}
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

// InErr returns the map of contextual values in the error.
// Each error in the stack is unwrapped and all maps are
// unioned. In case of collision, lower level error data
// take least priority.
func InErr(err error) *dataNode {
	if isNilErrIface(err) {
		return &dataNode{vs: map[string]any{}}
	}

	return &dataNode{vs: inErr(err)}
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
// eror interface compliance
// ------------------------------------------------------------

var _ error = &Err{}

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
// the conditional for both Err.e and Err.next.  This allows clues to
// Stack() maintain multiple error pointers without failing the otherwise
// linear errors.As check.
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

func New(msg string) *Err {
	return toErr(nil, msg, nil)
}

// NewWC is equivalent to clues.New("msg").WithClues(ctx)
func NewWC(ctx context.Context, msg string) *Err {
	return toErr(nil, msg, nil).WithClues(ctx)
}

// Wrap returns a clues.Err with a new message wrapping the old error.
func Wrap(err error, msg string) *Err {
	if isNilErrIface(err) {
		return nil
	}

	return toErr(err, msg, nil)
}

// WrapWC is equivalent to clues.Wrap(err, "msg").WithClues(ctx)
// Wrap returns a clues.Err with a new message wrapping the old error.
func WrapWC(ctx context.Context, err error, msg string) *Err {
	if isNilErrIface(err) {
		return nil
	}

	return Wrap(err, msg).WithClues(ctx)
}

// Stack returns the error as a clues.Err.  If additional errors are
// provided, the entire stack is flattened and returned as a single
// error chain.  All messages and stored structure is aggregated into
// the returned err.
//
// Ex: Stack(sentinel, errors.New("base")).Error() => "sentinel: base"
func Stack(errs ...error) *Err {
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
		return toErr(filtered[0], "", nil)
	}

	return toStack(filtered[0], filtered[1:])
}

// StackWC is equivalent to clues.Stack(errs...).WithClues(ctx)
func StackWC(ctx context.Context, errs ...error) *Err {
	stack := Stack(errs...)
	if stack == nil {
		return nil
	}

	return stack.WithClues(ctx)
}

// returns true if the error is nil, or is a non-nil interface containing a nil value.
func isNilErrIface(err error) bool {
	if err == nil {
		return true
	}

	val := reflect.ValueOf(err)

	return ((val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface) && val.IsNil())
}

// ---------------------------------------------------------------------------
// error core
// ---------------------------------------------------------------------------

// ErrCore is a minimized version of an Err{}.  Primarily intended for
// serializing a flattened version of the error stack
type ErrCore struct {
	Msg    string              `json:"msg"`
	Labels map[string]struct{} `json:"labels"`
	Values map[string]any      `json:"values"`
}

// Core transforms the Err to an ErrCore, flattening all the errors in
// the stack into a single struct.
func (err *Err) Core() *ErrCore {
	if isNilErrIface(err) {
		return nil
	}

	return &ErrCore{
		Msg:    err.Error(),
		Labels: err.Labels(),
		Values: err.values(),
	}
}

// ToCore transforms the Err to an ErrCore, flattening all the errors in
// the stack into a single struct
func ToCore(err error) *ErrCore {
	if isNilErrIface(err) {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = toErr(err, "", nil)
	}

	return e.Core()
}

func (ec *ErrCore) String() string {
	if ec == nil {
		return "<nil>"
	}

	return ec.stringer(false)
}

func (ec *ErrCore) stringer(fancy bool) string {
	sep := ", "
	ls := strings.Join(maps.Keys(ec.Labels), sep)

	vsl := []string{}
	for k, v := range ec.Values {
		vsl = append(vsl, k+":"+marshal(v, true))
	}

	vs := strings.Join(vsl, sep)

	if fancy {
		return `{msg:"` + ec.Msg + `", labels:[` + ls + `], values:{` + vs + `}}`
	}

	s := []string{}

	if len(ec.Msg) > 0 {
		s = append(s, `"`+ec.Msg+`"`)
	}

	if len(ls) > 0 {
		s = append(s, "["+ls+"]")
	}

	if len(vs) > 0 {
		s = append(s, "{"+vs+"}")
	}

	return "{" + strings.Join(s, ", ") + "}"
}

// Format provides cleaner printing of an ErrCore struct.
//
//	%s    only populated values are printed, without printing the property name.
//	%v    same as %s.
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+v    prints the full struct, including empty values and property names.
func (ec *ErrCore) Format(s fmt.State, verb rune) {
	if ec == nil {
		write(s, verb, "<nil>")
		return
	}

	if verb == 'v' {
		if s.Flag('+') {
			write(s, verb, ec.stringer(true))
			return
		}

		if s.Flag('#') {
			write(s, verb, ec.stringer(true))
			return
		}
	}

	write(s, verb, ec.stringer(false))
}
