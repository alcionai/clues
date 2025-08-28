package cluerr

import (
	"maps"

	"github.com/alcionai/clues/internal/errs"
)

// ------------------------------------------------------------
// labels
// ------------------------------------------------------------

func (err *Err) HasLabel(label string) bool {
	if errs.IsNilIface(err) {
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
	if errs.IsNilIface(err) {
		return false
	}

	if e, ok := err.(*Err); ok {
		return e.HasLabel(label)
	}

	return HasLabel(unwrap(err), label)
}

func (err *Err) Label(labels ...string) *Err {
	if errs.IsNilIface(err) {
		return nil
	}

	if len(err.labels) == 0 {
		err.labels = map[string]struct{}{}
	}

	for _, label := range labels {
		err.labels[label] = struct{}{}
	}

	return err
}

func Label(err error, label string) *Err {
	return tryExtendErr(err, "", nil, 1).Label(label)
}

func (err *Err) Labels() map[string]struct{} {
	if errs.IsNilIface(err) {
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

		err = unwrap(err)
	}

	return map[string]struct{}{}
}
