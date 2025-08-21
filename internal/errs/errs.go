package errs

import "reflect"

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

// IsNilIfac returns true if the error is nil, or if it is a
// non-nil interface containing a nil value.
func IsNilIface(err error) bool {
	if err == nil {
		return true
	}

	val := reflect.ValueOf(err)

	return (val.Kind() == reflect.Pointer ||
		val.Kind() == reflect.Interface) && val.IsNil()
}
