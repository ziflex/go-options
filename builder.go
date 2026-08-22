package options

// Builder describes an option for configuration type C with value type V.
// Create builders with New.
type Builder[C, V any] struct {
	setter     func(*C, V)
	value      V
	hasValue   bool
	name       string
	validators []Validator[V]
}

// New creates a builder that uses setter to apply its value.
func New[C, V any](setter func(*C, V)) Builder[C, V] {
	return Builder[C, V]{setter: setter}
}

// Value binds value to the option. Repeated calls replace the previous value.
func (b Builder[C, V]) Value(value V) Builder[C, V] {
	b.value = value
	b.hasValue = true

	return b
}

// Named adds field context to validation failures that do not already identify
// a field. Repeated calls replace the previous name.
func (b Builder[C, V]) Named(name string) Builder[C, V] {
	b.name = name

	return b
}

// Validators appends validators to the option in execution order.
func (b Builder[C, V]) Validators(validators ...Validator[V]) Builder[C, V] {
	combined := make([]Validator[V], 0, len(b.validators)+len(validators))
	combined = append(combined, b.validators...)
	combined = append(combined, validators...)
	b.validators = combined

	return b
}

// Build produces the configured option. If Value was not called, the option
// reports a validation failure when applied.
func (b Builder[C, V]) Build() Option[C] {
	setter := b.setter
	value := b.value
	hasValue := b.hasValue
	name := b.name
	validators := append([]Validator[V](nil), b.validators...)

	return func(config *C, report Report) {
		if !hasValue {
			report(ValidationError{Reason: "option value was not provided"})
			return
		}

		valid := true
		validatorReport := func(err ValidationError) {
			valid = false
			if err.Field == "" {
				err.Field = name
			}
			report(err)
		}

		for _, validator := range validators {
			if validator == nil {
				continue
			}

			validator(value, validatorReport)
		}

		if valid {
			setter(config, value)
		}
	}
}
