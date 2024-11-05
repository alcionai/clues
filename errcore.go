package clues

import (
	"fmt"
	"strings"

	"github.com/alcionai/clues/internal/node"
	"github.com/alcionai/clues/internal/stringify"
	"golang.org/x/exp/maps"
)

// ErrCore is a minimized version of an Err{}. It produces a concrete, storable
// version of the clues error data.  Rather than expose the underlying error
// structure that's used for building metadata, an error core synthesizes the
// hierarchical storage of errors and data nodes into a flat, easily consumed
// set of properties.
type ErrCore struct {
	Msg      string              `json:"msg"`
	Labels   map[string]struct{} `json:"labels"`
	Values   map[string]any      `json:"values"`
	Comments node.CommentHistory `json:"comments"`
}

// Core transforms the error into an ErrCore.
// ErrCore is a minimized version of an Err{}. It produces a concrete, storable
// version of the clues error data.  Rather than expose the underlying error
// structure that's used for building metadata, an error core synthesizes the
// hierarchical storage of errors and data nodes into a flat, easily consumed
// set of properties.
func (err *Err) Core() *ErrCore {
	if isNilErrIface(err) {
		return nil
	}

	return &ErrCore{
		Msg:      err.Error(),
		Labels:   err.Labels(),
		Values:   err.values(),
		Comments: err.Comments(),
	}
}

// ToCore transforms the error into an ErrCore.
// ErrCore is a minimized version of an Err{}. It produces a concrete, storable
// version of the clues error data.  Rather than expose the underlying error
// structure that's used for building metadata, an error core synthesizes the
// hierarchical storage of errors and data nodes into a flat, easily consumed
// set of properties.
func ToCore(err error) *ErrCore {
	if isNilErrIface(err) {
		return nil
	}

	e, ok := err.(*Err)
	if !ok {
		e = newErr(err, "", nil, 1)
	}

	return e.Core()
}

func (ec *ErrCore) String() string {
	if ec == nil {
		return "<nil>"
	}

	return ec.stringer(false)
}

// stringer handles all the fancy formatting of an errorCore.
func (ec *ErrCore) stringer(fancy bool) string {
	sep := ", "
	ls := strings.Join(maps.Keys(ec.Labels), sep)

	vsl := []string{}
	for k, v := range ec.Values {
		vsl = append(vsl, k+":"+stringify.Marshal(v, true))
	}

	vs := strings.Join(vsl, sep)

	csl := []string{}
	for _, c := range ec.Comments {
		csl = append(csl, c.Caller+" - "+c.File+" - "+c.Message)
	}

	cs := strings.Join(csl, sep)

	if fancy {
		return `{msg:"` + ec.Msg + `", labels:[` + ls + `], values:{` + vs + `}, comments:[` + cs + `]}`
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

	if len(cs) > 0 {
		s = append(s, "["+cs+"]")
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
