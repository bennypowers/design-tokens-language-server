package collections

import "fmt"

// Set is a generic set data structure using a map with zero-size values
type Set[T comparable] map[T]struct{}

// NewSet creates a new Set with the given initial values
func NewSet[T comparable](vs ...T) Set[T] {
	s := Set[T]{}
	s.Add(vs...)
	return s
}

// Add adds one or more values to the set
func (s Set[T]) Add(vs ...T) {
	for _, v := range vs {
		s[v] = struct{}{}
	}
}

// Has checks if the set contains the given value
func (s Set[T]) Has(v T) bool {
	_, ok := s[v]
	return ok
}

// Members returns all values in the set as a slice
func (s Set[T]) Members() []T {
	r := make([]T, 0, len(s))
	for v := range s {
		r = append(r, v)
	}
	return r
}

// String returns a string representation of the set
func (s Set[T]) String() string {
	return fmt.Sprintf("%v", s.Members())
}
