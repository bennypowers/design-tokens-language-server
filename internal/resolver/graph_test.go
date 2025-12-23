package resolver_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/resolver"
	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDependencyGraph(t *testing.T) {
	t.Run("simple dependency graph", func(t *testing.T) {
		tokenList := []*tokens.Token{
			{
				Name:          "color-base",
				Value:         "#FF6B35",
				SchemaVersion: schema.Draft,
				RawValue:      "#FF6B35",
			},
			{
				Name:          "color-primary",
				Value:         "{color.base}",
				SchemaVersion: schema.Draft,
				RawValue:      "{color.base}",
			},
		}

		graph := resolver.BuildDependencyGraph(tokenList)
		assert.NotNil(t, graph)

		// color-primary depends on color-base
		deps := graph.GetDependencies("color-primary")
		assert.Len(t, deps, 1)
		assert.Equal(t, "color-base", deps[0])

		// color-base has no dependencies
		deps = graph.GetDependencies("color-base")
		assert.Len(t, deps, 0)
	})

	t.Run("chained dependencies", func(t *testing.T) {
		tokenList := []*tokens.Token{
			{Name: "a", Value: "value", SchemaVersion: schema.Draft, RawValue: "value"},
			{Name: "b", Value: "{a}", SchemaVersion: schema.Draft, RawValue: "{a}"},
			{Name: "c", Value: "{b}", SchemaVersion: schema.Draft, RawValue: "{b}"},
		}

		graph := resolver.BuildDependencyGraph(tokenList)

		assert.Equal(t, []string{"a"}, graph.GetDependencies("b"))
		assert.Equal(t, []string{"b"}, graph.GetDependencies("c"))
	})

	t.Run("detect circular dependencies", func(t *testing.T) {
		tokenList := []*tokens.Token{
			{Name: "a", Value: "{b}", SchemaVersion: schema.Draft, RawValue: "{b}"},
			{Name: "b", Value: "{c}", SchemaVersion: schema.Draft, RawValue: "{c}"},
			{Name: "c", Value: "{a}", SchemaVersion: schema.Draft, RawValue: "{a}"},
		}

		graph := resolver.BuildDependencyGraph(tokenList)
		hasCycle := graph.HasCycle()
		assert.True(t, hasCycle, "should detect circular dependency")

		cycle := graph.FindCycle()
		assert.NotNil(t, cycle)
		assert.Greater(t, len(cycle), 0, "should return the cycle path")
	})
}

func TestTopologicalSort(t *testing.T) {
	t.Run("sort simple dependencies", func(t *testing.T) {
		tokenList := []*tokens.Token{
			{Name: "c", Value: "{b}", SchemaVersion: schema.Draft, RawValue: "{b}"},
			{Name: "b", Value: "{a}", SchemaVersion: schema.Draft, RawValue: "{a}"},
			{Name: "a", Value: "value", SchemaVersion: schema.Draft, RawValue: "value"},
		}

		graph := resolver.BuildDependencyGraph(tokenList)
		sorted, err := graph.TopologicalSort()
		require.NoError(t, err)

		// Should be sorted so dependencies come first
		// a has no deps, b depends on a, c depends on b
		// Expected order: a, b, c
		assert.Len(t, sorted, 3)

		aIndex := -1
		bIndex := -1
		cIndex := -1
		for i, name := range sorted {
			switch name {
			case "a":
				aIndex = i
			case "b":
				bIndex = i
			case "c":
				cIndex = i
			}
		}

		assert.Less(t, aIndex, bIndex, "a should come before b")
		assert.Less(t, bIndex, cIndex, "b should come before c")
	})

	t.Run("fail on circular dependencies", func(t *testing.T) {
		tokenList := []*tokens.Token{
			{Name: "a", Value: "{b}", SchemaVersion: schema.Draft, RawValue: "{b}"},
			{Name: "b", Value: "{a}", SchemaVersion: schema.Draft, RawValue: "{a}"},
		}

		graph := resolver.BuildDependencyGraph(tokenList)
		_, err := graph.TopologicalSort()
		assert.Error(t, err, "should fail on circular dependency")
		assert.ErrorIs(t, err, schema.ErrCircularReference)
	})
}
