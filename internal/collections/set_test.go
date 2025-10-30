package collections_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/collections"
	"github.com/stretchr/testify/assert"
)

func TestNewSet(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		s := collections.NewSet[string]()
		assert.NotNil(t, s)
		assert.Equal(t, 0, len(s))
	})

	t.Run("set with initial values", func(t *testing.T) {
		s := collections.NewSet("a", "b", "c")
		assert.Equal(t, 3, len(s))
		assert.True(t, s.Has("a"))
		assert.True(t, s.Has("b"))
		assert.True(t, s.Has("c"))
	})

	t.Run("set with duplicate initial values", func(t *testing.T) {
		s := collections.NewSet("a", "b", "a", "c", "b")
		assert.Equal(t, 3, len(s), "duplicates should be deduplicated")
		assert.True(t, s.Has("a"))
		assert.True(t, s.Has("b"))
		assert.True(t, s.Has("c"))
	})
}

func TestSetAdd(t *testing.T) {
	t.Run("add to empty set", func(t *testing.T) {
		s := collections.NewSet[string]()
		s.Add("a")
		assert.Equal(t, 1, len(s))
		assert.True(t, s.Has("a"))
	})

	t.Run("add multiple values", func(t *testing.T) {
		s := collections.NewSet[string]()
		s.Add("a", "b", "c")
		assert.Equal(t, 3, len(s))
		assert.True(t, s.Has("a"))
		assert.True(t, s.Has("b"))
		assert.True(t, s.Has("c"))
	})

	t.Run("add duplicate values", func(t *testing.T) {
		s := collections.NewSet("a")
		s.Add("a")
		assert.Equal(t, 1, len(s), "adding duplicate should not increase size")
		assert.True(t, s.Has("a"))
	})
}

func TestSetHas(t *testing.T) {
	s := collections.NewSet("red", "green", "blue")

	t.Run("has existing value", func(t *testing.T) {
		assert.True(t, s.Has("red"))
		assert.True(t, s.Has("green"))
		assert.True(t, s.Has("blue"))
	})

	t.Run("does not have non-existing value", func(t *testing.T) {
		assert.False(t, s.Has("yellow"))
		assert.False(t, s.Has(""))
	})
}

func TestSetMembers(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		s := collections.NewSet[string]()
		members := s.Members()
		assert.NotNil(t, members)
		assert.Equal(t, 0, len(members))
	})

	t.Run("non-empty set", func(t *testing.T) {
		s := collections.NewSet("a", "b", "c")
		members := s.Members()
		assert.Equal(t, 3, len(members))
		// Check all expected members are present (order is not guaranteed)
		assert.Contains(t, members, "a")
		assert.Contains(t, members, "b")
		assert.Contains(t, members, "c")
	})
}

func TestSetString(t *testing.T) {
	t.Run("empty set", func(t *testing.T) {
		s := collections.NewSet[string]()
		str := s.String()
		assert.Equal(t, "[]", str)
	})

	t.Run("non-empty set", func(t *testing.T) {
		s := collections.NewSet("a")
		str := s.String()
		assert.Equal(t, "[a]", str)
	})

	t.Run("set with multiple values", func(t *testing.T) {
		s := collections.NewSet("a", "b", "c")
		str := s.String()
		// String representation includes all members but order is not guaranteed
		assert.Contains(t, str, "a")
		assert.Contains(t, str, "b")
		assert.Contains(t, str, "c")
	})
}

func TestSetWithDifferentTypes(t *testing.T) {
	t.Run("int set", func(t *testing.T) {
		s := collections.NewSet(1, 2, 3)
		assert.True(t, s.Has(1))
		assert.True(t, s.Has(2))
		assert.True(t, s.Has(3))
		assert.False(t, s.Has(4))
	})

	t.Run("float64 set", func(t *testing.T) {
		s := collections.NewSet(1.5, 2.5, 3.5)
		assert.True(t, s.Has(1.5))
		assert.True(t, s.Has(2.5))
		assert.False(t, s.Has(4.5))
	})
}
