package options

import "reflect"

func isZero[V any](value V) bool {
	reflected := reflect.ValueOf(value)

	return !reflected.IsValid() || reflected.IsZero()
}
