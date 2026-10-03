package options

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"
)

func testEmptyCollections[C, E any](t *testing.T, constructor func(...Validator[E]) Validator[C], nilValue, empty, populated C) {
	t.Helper()
	calls := 0
	reject := Validator[E](func(E) error {
		calls++
		return errors.New("invalid")
	})
	validator := constructor(reject)
	for _, value := range []C{nilValue, empty} {
		if err := validator(value); err != nil {
			t.Fatalf("empty collection failed: %v", err)
		}
	}
	if calls != 0 {
		t.Fatalf("empty collections invoked validators %d times", calls)
	}
	for _, validator := range []Validator[C]{constructor(), constructor(nil, nil)} {
		if validator == nil {
			t.Fatal("expected a callable no-op validator")
		}
		for _, value := range []C{nilValue, empty, populated} {
			if err := validator(value); err != nil {
				t.Fatalf("no-op validator failed: %v", err)
			}
		}
	}
}

// Inspect only the aggregate's immediate location wrappers, keeping child
// wrapping and joins intact.
func locatedCollectionFailures(t *testing.T, err error) []ValidationError {
	t.Helper()
	aggregate, ok := err.(ValidationError)
	if !ok || aggregate.Field != "" || aggregate.Value != "" || !aggregate.OmitValue {
		t.Fatalf("expected a fieldless omitted aggregate, got %#v", err)
	}
	joined, ok := aggregate.Reason.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("aggregate reason = %T, want joined failures", aggregate.Reason)
	}
	var failures []ValidationError
	for _, child := range joined.Unwrap() {
		located, ok := child.(ValidationError)
		if !ok || located.Field == "" || located.Value != "" || !located.OmitValue {
			t.Fatalf("expected an omitted location wrapper, got %#v", child)
		}
		failures = append(failures, located)
	}
	return failures
}

func TestSliceEach(t *testing.T) {
	t.Run("empty inputs and validator lists", func(t *testing.T) {
		testEmptyCollections(t, SliceEach[[]int], nil, []int{}, []int{1})
	})

	t.Run("execution order and all failures", func(t *testing.T) {
		first := errors.New("first")
		second := errors.New("second")
		var order []string
		validator := SliceEach[[]int](
			nil,
			func(value int) error {
				order = append(order, fmt.Sprintf("%d:first", value))
				if value != 1 {
					return first
				}
				return nil
			},
			func(value int) error {
				order = append(order, fmt.Sprintf("%d:second", value))
				if value != 1 {
					return second
				}
				return nil
			},
		)
		input := []int{0, 1, 2}
		err := validator(input)
		wantOrder := []string{"0:first", "0:second", "1:first", "1:second", "2:first", "2:second"}
		if !reflect.DeepEqual(order, wantOrder) || !reflect.DeepEqual(input, []int{0, 1, 2}) {
			t.Fatalf("execution order = %v, input = %v", order, input)
		}
		failures := locatedCollectionFailures(t, err)
		wantLocations := []string{"[0]", "[0]", "[2]", "[2]"}
		if len(failures) != len(wantLocations) {
			t.Fatalf("failure count = %d, want %d", len(failures), len(wantLocations))
		}
		for i, failure := range failures {
			wantReason := []error{first, second}[i%2]
			if failure.Field != wantLocations[i] || failure.Reason != wantReason {
				t.Fatalf("failure %d = %+v", i, failure)
			}
		}
	})

	t.Run("snapshot and reuse", func(t *testing.T) {
		failure := errors.New("invalid")
		provided := []Validator[int]{func(value int) error {
			if value < 0 {
				return failure
			}
			return nil
		}}
		validator := SliceEach[[]int](provided...)
		provided[0] = func(int) error { return errors.New("replacement") }
		for range 2 {
			if err := validator([]int{-1, -2}); !errors.Is(err, failure) || len(locatedCollectionFailures(t, err)) != 2 {
				t.Fatal("validator lost its snapshot or accumulated failures")
			}
			if err := validator([]int{1}); err != nil {
				t.Fatalf("valid reuse retained failures: %v", err)
			}
		}
	})

	t.Run("defined slice and elements", func(t *testing.T) {
		type count int
		type counts []count
		input := counts{1, -2}
		err := SliceEach[counts](Positive[count]())(input)
		failures := locatedCollectionFailures(t, err)
		if len(failures) != 1 || failures[0].Field != "[1]" || err.Error() != "[1]: must be positive: value=-2" {
			t.Fatalf("defined slice validation = %v", err)
		}
		if !reflect.DeepEqual(input, counts{1, -2}) {
			t.Fatal("input changed")
		}
	})
}

func TestMapEach(t *testing.T) {
	for _, test := range []struct {
		name        string
		constructor func(...Validator[int]) Validator[map[int]int]
		prefix      string
	}{
		{"keys", MapKeys[map[int]int], "key"},
		{"values", MapValues[map[int]int], ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run("empty inputs and validator lists", func(t *testing.T) {
				testEmptyCollections(t, test.constructor, nil, map[int]int{}, map[int]int{1: 1})
			})

			t.Run("per-entry order and all failures", func(t *testing.T) {
				first := errors.New("first")
				second := errors.New("second")
				order := make(map[int][]int)
				validator := test.constructor(
					nil,
					func(value int) error {
						order[value] = append(order[value], 1)
						if value != 1 {
							return first
						}
						return nil
					},
					func(value int) error {
						order[value] = append(order[value], 2)
						if value != 1 {
							return second
						}
						return nil
					},
				)
				input := map[int]int{0: 0, 1: 1, 2: 2}
				err := validator(input)
				wantOrder := map[int][]int{0: {1, 2}, 1: {1, 2}, 2: {1, 2}}
				if !reflect.DeepEqual(order, wantOrder) || !reflect.DeepEqual(input, map[int]int{0: 0, 1: 1, 2: 2}) {
					t.Fatalf("execution order = %v, input = %v", order, input)
				}
				failures := locatedCollectionFailures(t, err)
				byLocation := make(map[string][]error)
				for _, failure := range failures {
					byLocation[failure.Field] = append(byLocation[failure.Field], failure.Reason)
				}
				want := map[string][]error{
					test.prefix + "[0]": {first, second},
					test.prefix + "[2]": {first, second},
				}
				if !reflect.DeepEqual(byLocation, want) || !errors.Is(err, first) || !errors.Is(err, second) {
					t.Fatalf("located failures = %v, want %v", byLocation, want)
				}
			})

			t.Run("snapshot and reuse", func(t *testing.T) {
				failure := errors.New("invalid")
				provided := []Validator[int]{func(value int) error {
					if value < 0 {
						return failure
					}
					return nil
				}}
				validator := test.constructor(provided...)
				provided[0] = func(int) error { return errors.New("replacement") }
				for range 2 {
					err := validator(map[int]int{-1: -1, -2: -2})
					if !errors.Is(err, failure) || len(locatedCollectionFailures(t, err)) != 2 {
						t.Fatal("validator lost its snapshot or accumulated failures")
					}
					if err := validator(map[int]int{1: 1}); err != nil {
						t.Fatalf("valid reuse retained failures: %v", err)
					}
				}
			})
		})
	}
}

func TestMapDefinedTypes(t *testing.T) {
	type name string
	type count int
	type limits map[name]count
	input := limits{"": 1, "timeout": -2}
	for _, test := range []struct {
		name      string
		validator Validator[limits]
		location  string
		message   string
	}{
		{"keys", MapKeys[limits](NotBlank[name]()), `key[""]`, `key[""]: must not be blank: value=""`},
		{"values", MapValues[limits](Positive[count]()), `["timeout"]`, `["timeout"]: must be positive: value=-2`},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.validator(input)
			failures := locatedCollectionFailures(t, err)
			if len(failures) != 1 || failures[0].Field != test.location || err.Error() != test.message {
				t.Fatalf("defined map validation = %v", err)
			}
			if !reflect.DeepEqual(input, limits{"": 1, "timeout": -2}) {
				t.Fatal("input changed")
			}
		})
	}
}

func TestMapValuesNaNKey(t *testing.T) {
	input := map[float64]int{math.NaN(): -7}
	var observed []int
	err := MapValues[map[float64]int](func(value int) error {
		observed = append(observed, value)
		return Positive[int]()(value)
	})(input)
	failures := locatedCollectionFailures(t, err)
	if !reflect.DeepEqual(observed, []int{-7}) || len(failures) != 1 || failures[0].Field != "[NaN]" {
		t.Fatalf("observed = %v, error = %v", observed, err)
	}
}

func namedCollectionError[C any](t *testing.T, value C, validator Validator[C]) error {
	t.Helper()
	setterCalls := 0
	option := New(func(_ *struct{}, _ C) { setterCalls++ }).
		Value(value).Named("option").Validators(validator).Build()
	_, err := Apply(option)
	if setterCalls != 0 {
		t.Fatal("failed collection validation invoked the setter")
	}
	return err
}

func TestCollectionErrorComposition(t *testing.T) {
	cause := &validationReason{message: "invalid"}
	direct := ValidationError{Field: "child", Value: "explicit", Reason: cause}
	pointer := &ValidationError{Field: "child", Value: "explicit", Reason: cause}
	omitted := ValidationError{OmitValue: true, Reason: cause}
	for _, collection := range []struct {
		name     string
		location string
		validate func(*testing.T, error) (error, error)
	}{
		{"slice", "[0]", func(t *testing.T, child error) (error, error) {
			validator := SliceEach[[]int](func(int) error { return child })
			return validator([]int{1}), namedCollectionError(t, []int{1}, validator)
		}},
		{"map keys", `key["a\"b\n"]`, func(t *testing.T, child error) (error, error) {
			validator := MapKeys[map[string]int](func(string) error { return child })
			input := map[string]int{"a\"b\n": 1}
			return validator(input), namedCollectionError(t, input, validator)
		}},
		{"map values", `["a\"b\n"]`, func(t *testing.T, child error) (error, error) {
			validator := MapValues[map[string]int](func(int) error { return child })
			input := map[string]int{"a\"b\n": 1}
			return validator(input), namedCollectionError(t, input, validator)
		}},
	} {
		for _, child := range []struct {
			name string
			err  error
		}{
			{"plain", cause},
			{"direct", direct},
			{"pointer", pointer},
			{"wrapped", fmt.Errorf("context: %w", pointer)},
			{"joined", errors.Join(direct, pointer)},
			{"omitted", omitted},
		} {
			t.Run(collection.name+"/"+child.name, func(t *testing.T) {
				raw, named := collection.validate(t, child.err)
				failures := locatedCollectionFailures(t, raw)
				if len(failures) != 1 || failures[0].Field != collection.location || failures[0].Reason != child.err {
					t.Fatalf("child error was not preserved at its location: %v", raw)
				}
				wantMessage := collection.location + ": " + child.err.Error()
				if raw.Error() != wantMessage || named.Error() != "option: "+wantMessage {
					t.Fatalf("raw = %q, named = %q", raw.Error(), named.Error())
				}
				for _, err := range []error{raw, named} {
					var found *validationReason
					if !errors.Is(err, child.err) || !errors.Is(err, cause) || !errors.As(err, &found) || found != cause {
						t.Fatal("original child or cause is not discoverable")
					}
					if child.name == "pointer" || child.name == "wrapped" || child.name == "joined" {
						var foundPointer *ValidationError
						if !errors.As(err, &foundPointer) || foundPointer != pointer {
							t.Fatal("original pointer child is not discoverable")
						}
					}
				}
				var outer ValidationError
				if !errors.As(named, &outer) || outer.Field != "option" || outer.Value != "" || !outer.OmitValue {
					t.Fatalf("missing outer option context: %v", named)
				}
				if *pointer != direct {
					t.Fatal("pointer child was mutated")
				}
			})
		}
	}
}

func TestNestedCollectionValidators(t *testing.T) {
	input := [][]int{{1}, {1, 2, -3}}
	validator := SliceEach[[][]int](SliceEach[[]int](Positive[int]()))
	option := New(func(_ *struct{}, _ [][]int) { t.Fatal("setter ran") }).
		Value(input).Named("matrix").Validators(validator).Build()
	_, err := Apply(option)
	if err == nil || err.Error() != "matrix: [1]: [2]: must be positive: value=-3" {
		t.Fatalf("nested error = %v", err)
	}

	failure := errors.New("reserved port")
	nested := MapValues[map[string][]int](SliceEach[[]int](func(value int) error {
		if value == 80 {
			return failure
		}
		return nil
	}))
	err = nested(map[string][]int{"ports": {8080, 80}})
	if err == nil || err.Error() != `["ports"]: [1]: reserved port` || !errors.Is(err, failure) {
		t.Fatalf("mixed nested error = %v", err)
	}
}

type collectionInterceptor func() int

func TestMapValuesFunctions(t *testing.T) {
	functionCalls := 0
	valid := collectionInterceptor(func() int { functionCalls++; return 1 })
	validator := MapValues[map[string]collectionInterceptor](NotNil[collectionInterceptor]())
	input := map[string]collectionInterceptor{"valid": valid, "first": nil, "second": nil}
	err := validator(input)
	failures := locatedCollectionFailures(t, err)
	locations := make(map[string]bool)
	for _, failure := range failures {
		locations[failure.Field] = true
		if failure.Reason.Error() != "must not be nil: value=<nil>" {
			t.Fatalf("function validation failure = %v", failure)
		}
	}
	if !reflect.DeepEqual(locations, map[string]bool{`["first"]`: true, `["second"]`: true}) || functionCalls != 0 {
		t.Fatalf("locations = %v, function calls = %d", locations, functionCalls)
	}
	if len(input) != 3 || input["valid"] == nil || input["first"] != nil || input["second"] != nil {
		t.Fatal("map entries changed")
	}
	if err := validator(map[string]collectionInterceptor{"valid": valid}); err != nil || functionCalls != 0 {
		t.Fatal("valid functions failed or were invoked by validation")
	}
}

func TestCollectionConcurrentReuse(t *testing.T) {
	failure := errors.New("invalid")
	child := Validator[int](func(value int) error {
		if value < 0 {
			return failure
		}
		return nil
	})
	slice := SliceEach[[]int](child)
	keys := MapKeys[map[int]int](child)
	values := MapValues[map[int]int](child)
	for _, test := range []struct {
		name     string
		validate func(int) error
	}{
		{"slice", func(value int) error { return slice([]int{value, value}) }},
		{"keys", func(value int) error { return keys(map[int]int{value: 1, 2 * value: 1}) }},
		{"values", func(value int) error { return values(map[int]int{1: value, 2: value}) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			type result struct {
				value int
				err   error
			}
			results := make(chan result, 16)
			var workers sync.WaitGroup
			for i := range 16 {
				workers.Add(1)
				go func() {
					defer workers.Done()
					value := 1
					if i%2 == 0 {
						value = -1
					}
					results <- result{value: value, err: test.validate(value)}
				}()
			}
			workers.Wait()
			close(results)
			for result := range results {
				if result.value > 0 {
					if result.err != nil {
						t.Errorf("valid invocation retained failures: %v", result.err)
					}
				} else if !errors.Is(result.err, failure) || len(locatedCollectionFailures(t, result.err)) != 2 {
					t.Error("failed invocation lost or accumulated failures")
				}
			}
		})
	}
}

func TestInterceptorCollectionOption(t *testing.T) {
	type config struct{ interceptors []collectionInterceptor }
	functionCalls := 0
	first := collectionInterceptor(func() int { functionCalls++; return 1 })
	second := collectionInterceptor(func() int { functionCalls++; return 2 })
	setterCalls := 0
	withInterceptors := func(values ...collectionInterceptor) Option[config] {
		captured := append([]collectionInterceptor(nil), values...)
		return New(func(cfg *config, value []collectionInterceptor) {
			setterCalls++
			cfg.interceptors = append(cfg.interceptors, value...)
		}).Value(captured).Named("unary interceptors").
			Validators(SliceEach[[]collectionInterceptor](NotNil[collectionInterceptor]())).Build()
	}
	initial := config{interceptors: []collectionInterceptor{first}}
	got, err := ApplyTo(initial, withInterceptors(first, nil, second, nil))
	if err == nil || err.Error() != "unary interceptors: [1]: must not be nil: value=<nil>\n[3]: must not be nil: value=<nil>" {
		t.Fatalf("interceptor error = %v", err)
	}
	if setterCalls != 0 || functionCalls != 0 || len(got.interceptors) != 1 || &got.interceptors[0] != &initial.interceptors[0] {
		t.Fatal("failed validation changed configuration or invoked a function")
	}
	var outer ValidationError
	if !errors.As(err, &outer) {
		t.Fatal("expected outer option context")
	}
	outer.Field = ""
	if len(locatedCollectionFailures(t, outer)) != 2 {
		t.Fatal("expected both nil indices")
	}

	provided := []collectionInterceptor{second, first}
	valid := withInterceptors(provided...)
	provided[0] = nil
	got, err = Apply(valid)
	if err != nil || setterCalls != 1 || functionCalls != 0 || len(got.interceptors) != 2 {
		t.Fatalf("valid interceptors: setter calls = %d, function calls = %d, error = %v", setterCalls, functionCalls, err)
	}
	// Invoke only after validation to observe the order delivered to the setter.
	if got.interceptors[0]() != 2 || got.interceptors[1]() != 1 {
		t.Fatal("setter received reordered functions")
	}
}
