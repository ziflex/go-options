# go-options

A lightweight, generic functional options library for Go with built-in validation support.

## Overview

`go-options` provides a clean and type-safe way to implement the functional options pattern in Go. It leverages generics to work with any configuration struct and includes a built-in mechanism for reporting and collecting validation errors.

## Installation

```bash
go get github.com/ziflex/go-options
```

## API

### Types

- `Option[T any]`: A function type `func(*T, Report)` used to modify a configuration of type `T`.
- `Report`: A callback function `func(ValidationError)` used within an `Option` to report validation errors.
- `ValidationError`: A struct representing a validation failure, containing `Field`, `Value`, and `Reason`.

### Functions

- `Apply[T any](opts ...Option[T]) (T, error)`: Creates a zero-value instance of `T` and applies the provided options. Returns the populated instance and any collected errors (joined via `errors.Join`).
- `ApplyWithValues[T any](initial T, opts ...Option[T]) (T, error)`: Applies options to an existing instance of `T`.

## Examples

### Basic Usage

```go
package main

import (
	"fmt"
	"github.com/ziflex/go-options"
)

type Config struct {
	Name    string
	Timeout int
}

func WithName(name string) options.Option[Config] {
	return func(c *Config, _ options.Report) {
		c.Name = name
	}
}

func main() {
	// Apply options to a new Config instance
	cfg, err := options.Apply(WithName("my-service"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Name: %s\n", cfg.Name)
}
```

### Validation

`go-options` makes it easy to validate your configuration as it's being built.

```go
func WithTimeout(seconds int) options.Option[Config] {
	return func(c *Config, report options.Report) {
		if seconds < 0 {
			report(options.ValidationError{
				Field:  "Timeout",
				Reason: "timeout cannot be negative",
				Value:  fmt.Sprintf("%d", seconds),
			})
			return
		}
		c.Timeout = seconds
	}
}

func main() {
	_, err := options.Apply(WithTimeout(-1))
	if err != nil {
		fmt.Println("Validation failed:", err)
	}
}
```

### Applying to Existing Config

```go
func main() {
	initial := Config{Name: "default", Timeout: 30}
	
	cfg, err := options.ApplyWithValues(initial, WithName("override"))
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Name: %s, Timeout: %d\n", cfg.Name, cfg.Timeout)
}
```

## License

MIT
