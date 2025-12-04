package utils

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlicesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "equal slices same order",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "equal slices different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"c", "a", "b"},
			expected: true,
		},
		{
			name:     "different slices",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "different lengths",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "duplicates handled correctly",
			a:        []string{"a", "a", "b"},
			b:        []string{"a", "b", "b"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SlicesEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "SlicesEqual(, ) = %v %v", tt.a, tt.b)
		})
	}
}

func TestSliceContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "contains item",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "does not contain item",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
		{
			name:     "contains at start",
			slice:    []string{"a", "b", "c"},
			item:     "a",
			expected: true,
		},
		{
			name:     "contains at end",
			slice:    []string{"a", "b", "c"},
			item:     "c",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceContains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result, "SliceContains(, ) = %v %v", tt.slice, tt.item)
		})
	}
}

func TestFilterSlice(t *testing.T) {
	slice := []string{"one", "two", "three", "four", "five"}
	expected := []string{"one", "three", "five"}

	actual := FilterSlice(slice, func(s string) bool {
		return strings.HasSuffix(s, "e")
	})

	assert.Equal(t, expected, actual)
}

func TestFindIndex(t *testing.T) {
	slice := []string{"one", "two", "three", "four", "five"}

	t.Run("found", func(t *testing.T) {
		idx := FindIndex(slice, func(s string) bool {
			return s == "three"
		})

		assert.Equal(t, 2, idx)
	})

	t.Run("not found", func(t *testing.T) {
		idx := FindIndex(slice, func(s string) bool {
			return s == "six"
		})

		assert.Equal(t, -1, idx)
	})
}

func TestFind(t *testing.T) {
	slice := []string{"one", "two", "three", "four", "five"}

	t.Run("found", func(t *testing.T) {
		found, ok := Find(slice, func(s string) bool {
			return s == "three"
		})

		assert.True(t, ok)
		assert.Equal(t, "three", found)
	})

	t.Run("not found", func(t *testing.T) {
		found, ok := Find(slice, func(s string) bool {
			return s == "six"
		})

		assert.False(t, ok)
		assert.Equal(t, "", found)
	})
}

func TestMapSlice(t *testing.T) {
	type complexType struct {
		Value string
	}

	slice := []complexType{
		{Value: "one"},
		{Value: "two"},
		{Value: "three"},
		{Value: "four"},
		{Value: "five"},
	}

	actual := MapSlice(slice, func(c complexType) string {
		return c.Value
	})

	expected := []string{"one", "two", "three", "four", "five"}
	assert.Equal(t, expected, actual)
}

func TestEvery(t *testing.T) {
	slice := []string{"one", "two", "three", "four", "five"}

	t.Run("every", func(t *testing.T) {
		every := Every(slice, func(s string) bool {
			return strings.HasSuffix(s, "e")
		})

		assert.False(t, every)
	})

	t.Run("every - empty slice", func(t *testing.T) {
		every := Every([]string{}, func(s string) bool {
			return strings.HasSuffix(s, "e")
		})

		assert.True(t, every)
	})
}

func TestSome(t *testing.T) {
	slice := []string{"one", "two", "three", "four", "five"}

	t.Run("some", func(t *testing.T) {
		some := Some(slice, func(s string) bool {
			return strings.HasSuffix(s, "e")
		})

		assert.True(t, some)
	})

	t.Run("some - empty slice", func(t *testing.T) {
		some := Some([]string{}, func(s string) bool {
			return strings.HasSuffix(s, "e")
		})

		assert.False(t, some)
	})
}

func TestReduceSlice(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}

	t.Run("sum", func(t *testing.T) {
		sum := ReduceSlice(slice, func(acc, val int) int {
			return acc + val
		}, 0)

		assert.Equal(t, 15, sum)
	})

	t.Run("product", func(t *testing.T) {
		product := ReduceSlice(slice, func(acc, val int) int {
			return acc * val
		}, 1)

		assert.Equal(t, 120, product)
	})

	t.Run("complex type as a destination", func(t *testing.T) {
		type complexType struct {
			Value int
		}

		sum := ReduceSlice(slice, func(acc complexType, val int) complexType {
			acc.Value += val
			return acc
		}, complexType{})

		assert.Equal(t, 15, sum.Value)
	})
}

func TestSliceUnique(t *testing.T) {
	t.Run("unique integers", func(t *testing.T) {
		input := []int{1, 2, 2, 3, 4, 4, 5}
		expected := []int{1, 2, 3, 4, 5}
		actual := SliceUnique(input)
		assert.Equal(t, expected, actual)
	})

	t.Run("unique strings", func(t *testing.T) {
		input := []string{"apple", "banana", "apple", "cherry", "banana"}
		expected := []string{"apple", "banana", "cherry"}
		actual := SliceUnique(input)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty slice", func(t *testing.T) {
		input := []int{}
		expected := []int{}
		actual := SliceUnique(input)
		assert.Equal(t, expected, actual)
	})

	t.Run("all unique elements", func(t *testing.T) {
		input := []string{"apple", "banana", "cherry"}
		expected := []string{"apple", "banana", "cherry"}
		actual := SliceUnique(input)
		assert.Equal(t, expected, actual)
	})
}

func TestMapContainsKey(t *testing.T) {
	t.Run("string keys", func(t *testing.T) {
		m := map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
		}
		key, found := MapContainsKey("two", m)
		assert.True(t, found)
		assert.Equal(t, "two", key)

		key, found = MapContainsKey("four", m)
		assert.False(t, found)
		assert.Equal(t, "", key)
	})

	t.Run("struct keys", func(t *testing.T) {
		type Key struct {
			ID   int
			Name string
		}
		type Value struct {
			Age int
		}
		m := map[Key]Value{
			{ID: 1, Name: "Alice"}: {Age: 30},
			{ID: 2, Name: "Bob"}:   {Age: 25},
		}
		key, found := MapContainsKey(Key{ID: 1, Name: "Alice"}, m)
		assert.True(t, found)
		assert.Equal(t, Key{ID: 1, Name: "Alice"}, key)

		key, found = MapContainsKey(Key{ID: 3, Name: "Charlie"}, m)
		assert.False(t, found)
		assert.Equal(t, Key{}, key)
	})
}

func TestMapKeys(t *testing.T) {
	t.Run("string keys", func(t *testing.T) {
		m := map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
		}
		keys := MapKeys(m)
		expected := []string{"one", "two", "three"}
		sort.Strings(keys)
		sort.Strings(expected)
		assert.Equal(t, expected, keys)
	})

	t.Run("struct keys", func(t *testing.T) {
		type Key struct {
			ID   int
			Name string
		}
		type Value struct {
			Age int
		}
		m := map[Key]Value{
			{ID: 1, Name: "Alice"}: {Age: 30},
			{ID: 2, Name: "Bob"}:   {Age: 25},
		}
		keys := MapKeys(m)
		expected := []Key{
			{ID: 1, Name: "Alice"},
			{ID: 2, Name: "Bob"},
		}
		assert.ElementsMatch(t, expected, keys)
	})
}

func TestMapValues(t *testing.T) {
	t.Run("string values", func(t *testing.T) {
		m := map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
		}
		values := MapValues(m)
		expected := []int{1, 2, 3}
		sort.Ints(values)
		sort.Ints(expected)
		assert.Equal(t, expected, values)
	})

	t.Run("struct values", func(t *testing.T) {
		type Key struct {
			ID   int
			Name string
		}
		type Value struct {
			Age int
		}
		m := map[Key]Value{
			{ID: 1, Name: "Alice"}: {Age: 30},
			{ID: 2, Name: "Bob"}:   {Age: 25},
		}
		values := MapValues(m)
		expected := []Value{
			{Age: 30},
			{Age: 25},
		}
		assert.ElementsMatch(t, expected, values)
	})
}

func TestFindKeyInMap(t *testing.T) {
	t.Run("string keys", func(t *testing.T) {
		m := map[string]int{
			"one":   1,
			"two":   2,
			"three": 3,
		}
		key, found := FindKeyInMap("tw", m)
		assert.True(t, found)
		assert.Equal(t, "two", key)

		key, found = FindKeyInMap("four", m)
		assert.False(t, found)
		assert.Equal(t, "", key)
	})
}

func TestSliceRemove(t *testing.T) {
	ctx := context.Background()
	generateTestSlice := func() []string {
		return []string{"one", "two", "three", "four", "five"}
	}

	const testIndex = 2

	t.Run("Ordered Remove", func(t *testing.T) {
		slice := generateTestSlice()
		expected := []string{"one", "two", "four", "five"}

		actual := SliceRemove(ctx, slice, testIndex)
		assert.Equal(t, expected, actual)
	})

	t.Run("Ordered Remove - Out of bounds", func(t *testing.T) {
		slice := generateTestSlice()
		expected := generateTestSlice()

		actual := SliceRemove(ctx, slice, len(slice))
		assert.Equal(t, expected, actual)
	})

	t.Run("Unordered Remove", func(t *testing.T) {
		slice := generateTestSlice()
		removed := slice[testIndex]

		assert.Equal(t, SliceContains(slice, removed), true)

		actual := UnorderedSliceRemove(ctx, slice, testIndex)

		assert.Equal(t, SliceContains(actual, removed), false)
	})

	t.Run("Unordered Remove - Out of bounds", func(t *testing.T) {
		slice := generateTestSlice()
		expected := generateTestSlice()

		actual := UnorderedSliceRemove(ctx, slice, len(slice))
		assert.Equal(t, expected, actual)
	})
}
