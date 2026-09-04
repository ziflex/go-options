package options

// Predicate reports whether a default should replace a value.
type Predicate[V any] func(V) bool

// EmptyString returns a predicate that matches empty strings.
func EmptyString[S ~string]() Predicate[S] {
	return func(value S) bool {
		return value == ""
	}
}

// NilSlice returns a predicate that matches nil slices.
func NilSlice[S ~[]E, E any]() Predicate[S] {
	return func(value S) bool {
		return value == nil
	}
}

// EmptySlice returns a predicate that matches nil and non-nil empty slices.
func EmptySlice[S ~[]E, E any]() Predicate[S] {
	return func(value S) bool {
		return len(value) == 0
	}
}

// NilMap returns a predicate that matches nil maps.
func NilMap[M ~map[K]V, K comparable, V any]() Predicate[M] {
	return func(value M) bool {
		return value == nil
	}
}

// EmptyMap returns a predicate that matches nil and non-nil empty maps.
func EmptyMap[M ~map[K]V, K comparable, V any]() Predicate[M] {
	return func(value M) bool {
		return len(value) == 0
	}
}
