package options

import "testing"

func TestEmptyString(t *testing.T) {
	type name string

	predicate := EmptyString[name]()
	if !predicate("") {
		t.Fatal("EmptyString()(empty) = false, want true")
	}
	if predicate("value") {
		t.Fatal("EmptyString()(non-empty) = true, want false")
	}
}

func TestSlicePredicates(t *testing.T) {
	type names []string

	tests := []struct {
		name      string
		value     names
		predicate Predicate[names]
		want      bool
	}{
		{name: "nil is nil", value: nil, predicate: NilSlice[names](), want: true},
		{name: "empty is not nil", value: names{}, predicate: NilSlice[names](), want: false},
		{name: "populated is not nil", value: names{"value"}, predicate: NilSlice[names](), want: false},
		{name: "nil is empty", value: nil, predicate: EmptySlice[names](), want: true},
		{name: "empty is empty", value: names{}, predicate: EmptySlice[names](), want: true},
		{name: "populated is not empty", value: names{"value"}, predicate: EmptySlice[names](), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.predicate(test.value); got != test.want {
				t.Fatalf("predicate(%#v) = %t, want %t", test.value, got, test.want)
			}
		})
	}
}

func TestMapPredicates(t *testing.T) {
	type labels map[string]int

	tests := []struct {
		name      string
		value     labels
		predicate Predicate[labels]
		want      bool
	}{
		{name: "nil is nil", value: nil, predicate: NilMap[labels](), want: true},
		{name: "empty is not nil", value: labels{}, predicate: NilMap[labels](), want: false},
		{name: "populated is not nil", value: labels{"value": 1}, predicate: NilMap[labels](), want: false},
		{name: "nil is empty", value: nil, predicate: EmptyMap[labels](), want: true},
		{name: "empty is empty", value: labels{}, predicate: EmptyMap[labels](), want: true},
		{name: "populated is not empty", value: labels{"value": 1}, predicate: EmptyMap[labels](), want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.predicate(test.value); got != test.want {
				t.Fatalf("predicate(%#v) = %t, want %t", test.value, got, test.want)
			}
		})
	}
}
