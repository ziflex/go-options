# go-options

A lightweight, generic functional options library for Go with built-in validation support.

## Overview

`go-options` provides a clean and type-safe way to implement the functional options pattern in Go. It leverages generics to work with any configuration struct and composes option and validation failures with ordinary Go errors.

## Installation

```bash
go get github.com/ziflex/go-options
```

## API

### Types

- `Option[T any]`: A type alias for `func(*T) error` used to modify a
  configuration of type `T`. Equivalent project-specific function types can be
  used without explicit conversions.
- `Builder[C, V any]`: A value-based builder that describes one option before producing it with `Build`.
- `Predicate[V any]`: A function type `func(V) bool` that reports whether a
  default should replace an option value.
- `Validator[V any]`: A function type `func(V) error` used to validate an option value without receiving the destination configuration.
- `ValidationError`: A struct representing a validation failure, containing
  `Field`, `Value`, and an error-valued `Reason`.

### Functions

- `Apply(opts...)` creates a zero-value configuration and applies the options.
- `ApplyTo(initial, opts...)` applies options to an existing configuration.
- `Check(check)` adapts an error-returning function into a validator and infers
  its value type.
- `New(setter)` creates an option builder and infers its configuration and value types from the setter.

`Apply` invokes every option and combines returned failures with `errors.Join`.
Builder-produced options resolve defaults before running every validator, but
call their setter only when all validators return `nil`. Validators and custom
options may return `errors.Join` to describe multiple failures. Nil options and
nil validators are ignored.
`ValidationError` unwraps its `Reason`, so callers can inspect underlying causes
with `errors.Is` and `errors.As`.

`ApplyWithValues` remains available as a deprecated alias for `ApplyTo`.

## Examples

### Building options

Option construction has six stages:

- `New` defines how the option modifies its configuration.
- `Value` binds the option value.
- `Default` or `DefaultWhen` optionally supplies a fallback when `Value` is
  omitted or matches the configured fallback policy.
- `Named` optionally sets the option name used when `Build` describes
  validation failures.
- `Validators` optionally appends validators in execution order.
- `Build` produces the final `Option`.

Builder methods return updated values, so base builders can be reused safely.
Conventional `WithX` functions build one option per call:

```go
package main

import (
	"fmt"
	"time"

	"github.com/ziflex/go-options"
)

type Config struct {
	Host    string
	Timeout time.Duration
	Workers int
}

func WithWorkers(workers int) options.Option[Config] {
	return options.New(func(config *Config, value int) {
		config.Workers = value
	}).
		Value(workers).
		Validators(options.Min(1), options.Max(32)).
		Build()
}

func WithTimeout(timeout time.Duration) options.Option[Config] {
	return options.New(
		func(config *Config, value time.Duration) {
			config.Timeout = value
		},
	).
		Value(timeout).
		Named("timeout").
		Validators(options.Min(time.Second)).
		Build()
}

func main() {
	config, err := options.Apply(
		WithTimeout(5*time.Second),
		WithWorkers(4),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(config.Timeout, config.Workers)
}
```

### Default values

`Default` uses Go zero-value semantics. It replaces numeric zero, `""`,
`false`, zero arrays and structs, and nil pointers, interfaces, slices, maps,
channels, and functions. Non-nil empty slices and maps are preserved, so callers
can distinguish an omitted collection from an explicitly supplied empty one:

```go
func WithWorkers(workers int) options.Option[Config] {
	return options.New(func(config *Config, value int) {
		config.Workers = value
	}).Value(workers).Default(4).Build()
}
```

Defaults are resolved before validation, so validators and the setter receive
the same effective value. `DefaultWhen` uses a caller-provided `Predicate`
instead of reflection, which supports domain-specific empty values:

```go
func WithHost(host string) options.Option[Config] {
	return options.New(func(config *Config, value string) {
		config.Host = value
	}).Value(host).DefaultWhen("localhost", func(value string) bool {
		return value == "auto"
	}).Build()
}
```

`Value` and either default method may be called in either order. Repeated
fallback declarations use the latest policy. A nil custom predicate never
matches a supplied value, but its default still satisfies a builder without a
`Value`.

A plain boolean cannot distinguish an omitted value from an explicit `false`.
Consequently, `Value(false).Default(true)` resolves to `true`. Use a pointer or
an optional wrapper when both states must remain distinct.

Built-in predicates follow the same generic constructor style as validators and
support named string, slice, and map types:

| Predicate | Matches |
| --- | --- |
| `EmptyString[S]()` | Empty strings. |
| `NilSlice[S]()` | Nil slices only. |
| `EmptySlice[S]()` | Nil and non-nil zero-length slices. |
| `NilMap[M]()` | Nil maps only. |
| `EmptyMap[M]()` | Nil and non-nil zero-length maps. |

### Custom options

Options may also be implemented directly. Because `Option` is a type alias,
projects can expose their own named option function type and use it directly
with `Apply`. Options produced by a builder can use the same project-specific
type without an explicit conversion:

```go
type ConfigOption func(*Config) error

func WithWorkers(workers int) ConfigOption {
	return func(config *Config) error {
		if workers < 1 {
			return options.ValidationError{
				Field:  "workers",
				Reason: errors.New("must be positive"),
			}
		}

		config.Workers = workers

		return nil
	}
}

func WithTimeout(timeout time.Duration) ConfigOption {
	return options.New(func(config *Config, value time.Duration) {
		config.Timeout = value
	}).Value(timeout).Build()
}

config, err := options.Apply(
	WithWorkers(4),
	WithTimeout(5*time.Second),
)
```

A custom option with independent failures can return
`errors.Join(firstError, secondError)`.

### Named diagnostics

Validators do not require a field name. `Named` identifies the option being
configured. When a validator directly returns a fieldless `ValidationError`,
`Build` copies it, adds the option name, and fills a missing value from the
bound option value. Ordinary errors, validation errors that already identify a
field, and wrapped or joined failures retain their existing wrapper behavior:

```go
func WithNamedWorkers(workers int) options.Option[Config] {
	return options.New(func(config *Config, value int) {
		config.Workers = value
	}).
		Value(workers).
		Named("workers").
		Validators(options.Min(1), options.Max(32)).
		Build()
}
```

### Custom validation

Use `Check` for application-specific rules. It infers the value type from the
function parameter; return `nil` when the value is valid or an error describing
why it is invalid:

```go
var Even = options.Check(func(value int) error {
	if value%2 != 0 {
		return options.ValidationError{
			Reason: errors.New("must be even"),
			Value:  fmt.Sprint(value),
		}
	}

	return nil
})

func WithEvenWorkers(workers int) options.Option[Config] {
	return options.New(func(config *Config, value int) {
		config.Workers = value
	}).
		Value(workers).
		Validators(Even).
		Build()
}
```

Explicit conversion with `options.Validator[int](func(value int) error { ... })`
remains available when type inference is not needed.

Return `errors.Join` when one validator needs to describe multiple failures:

```go
return errors.Join(
	options.ValidationError{Reason: errors.New("must be even")},
	options.ValidationError{Reason: errors.New("must be positive")},
)
```

### Built-in validators

| Validator | Applies to | Behavior |
| --- | --- | --- |
| `NotNil[V]()` | Any value | Rejects nil pointers, interfaces (including typed nils), slices, maps, functions, channels, and unsafe pointers. Non-nilable types always pass. |
| `NotNilPtr[V]()` | Pointers | Reflection-free validator that rejects nil pointers. |
| `NotZero[V]()` | Comparable values | Rejects the zero value of `V`. |
| `Positive[V]()` | Ordered numeric values | Accepts values `> 0`; `NaN` fails. Supports named signed, unsigned, and floating-point types. |
| `NonNegative[V]()` | Ordered numeric values | Accepts values `>= 0`; `NaN` fails. Supports named signed, unsigned, and floating-point types. |
| `Negative[V]()` | Ordered numeric values | Accepts values `< 0`; `NaN` fails. Supports named signed, unsigned, and floating-point types. |
| `NonPositive[V]()` | Ordered numeric values | Accepts values `<= 0`; `NaN` fails. Supports named signed, unsigned, and floating-point types. |
| `NotEmpty[S]()` | Strings | Rejects the empty string. |
| `NotBlank[S]()` | Strings | Rejects empty strings and strings containing only Unicode whitespace. |
| `Min[V](minimum)` | Ordered values | Accepts values `>= minimum`; `NaN` fails. |
| `Max[V](maximum)` | Ordered values | Accepts values `<= maximum`; `NaN` fails. |
| `Between[V](minimum, maximum)` | Ordered values | Accepts values within the inclusive bounds. `NaN` fails, and reversed bounds form an empty interval. |
| `MinLen[S](minimum)` | Strings | Accepts strings whose byte length is at least `minimum`. |
| `MaxLen[S](maximum)` | Strings | Accepts strings whose byte length is at most `maximum`. |
| `SliceNotEmpty[S]()` | Slices | Rejects nil and empty slices. |
| `SliceMinLen[S](minimum)` | Slices | Accepts slices with at least `minimum` elements; nil slices have length zero. |
| `SliceMaxLen[S](maximum)` | Slices | Accepts slices with at most `maximum` elements; nil slices have length zero. |
| `MapNotEmpty[M]()` | Maps | Rejects nil and empty maps. |
| `MapMinLen[M](minimum)` | Maps | Accepts maps with at least `minimum` entries; nil maps have length zero. |
| `MapMaxLen[M](maximum)` | Maps | Accepts maps with at most `maximum` entries; nil maps have length zero. |
| `OneOf[V](allowed...)` | Comparable values | Accepts values in the allowed set; an empty set rejects every value. |
| `NotOneOf[V](disallowed...)` | Comparable values | Accepts values outside the disallowed set; an empty set accepts every value. |

Collection validators are statically typed so unsupported kinds fail at compile
time. The first type argument is sufficient for named collection types:

```go
type Names []string
type Labels map[string]int

var NamesRequired = options.SliceNotEmpty[Names]()
var NamesLimited = options.SliceMaxLen[Names](10)
var LabelsRequired = options.MapNotEmpty[Labels]()
var LabelsLimited = options.MapMaxLen[Labels](10)
```

### Applying to an existing configuration

```go
defaults := Config{Timeout: 30 * time.Second}
config, err := options.ApplyTo(defaults, WithTimeout(5*time.Second))
```

## License

MIT
