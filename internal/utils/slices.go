package utils

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// SlicesEqual checks if two slices contain the same elements (order independent)
func SlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int)
	for _, v := range a {
		seen[v]++
	}
	for _, v := range b {
		seen[v]--
		if seen[v] < 0 {
			return false
		}
	}
	for _, count := range seen {
		if count != 0 {
			return false
		}
	}
	return true
}

func MapContainsKey[K comparable, V any](pattern K, input map[K]V) (K, bool) {
	var (
		found K
		ok    bool
	)
	for k := range input {
		ok = k == pattern
		if ok {
			found = k
			break
		}
	}

	return found, ok
}

// MapKeys Generic Function to take any map and return an array of keys
func MapKeys[K comparable, V any](m map[K]V) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

// MapValues Generic Function to take any map and return an array of values
func MapValues[K comparable, V any](m map[K]V) []V {
	r := make([]V, 0, len(m))
	for _, v := range m {
		r = append(r, v)
	}
	return r
}

// FindKeyInMap partial matches string to map keys and returns the actual key
func FindKeyInMap[V any](pattern string, input map[string]V) (string, bool) {
	for key := range input {
		if strings.Contains(strings.ToLower(key), strings.ToLower(pattern)) {
			return key, true
		}
	}

	return "", false
}

// FindIndex finds the index of an element in a slice
func FindIndex[T any](elems []T, predicate func(T) bool) int {
	for i, elem := range elems {
		if predicate(elem) {
			return i
		}
	}
	return -1
}

// Find finds the first element in a slice that satisfies the predicate
func Find[T any](elems []T, predicate func(T) bool) (T, bool) {
	var out T
	for _, elem := range elems {
		if predicate(elem) {
			return elem, true
		}
	}
	return out, false
}

// IndexOf gets the index of an element (comparables only)
func IndexOf[T comparable](elems []T, lookup T) int {
	return FindIndex(elems, func(elem T) bool {
		return elem == lookup
	})
}

// SliceContains checks whether or not an element exists (comparables only)
func SliceContains[T comparable](elems []T, lookup T) bool {
	return IndexOf(elems, lookup) > -1
}

// MapSlice takes a slice and applies a function to each element to output a new slice of a different type
func MapSlice[T any, K any](elems []T, mapper func(T) K) []K {
	var out []K
	for _, elem := range elems {
		out = append(out, mapper(elem))
	}
	return out
}

// ReduceSlice takes a slice and applies a reducer function to each element to output a single value
// assumes the destination is initialized
func ReduceSlice[T any, K any](elems []T, reducer func(K, T) K, dest K) K {
	out := dest

	for _, elem := range elems {
		out = reducer(out, elem)
	}
	return out
}

// Every checks if all elements in a slice satisfy a predicate
func Every[T any](elems []T, predicate func(T) bool) bool {
	for _, elem := range elems {
		if !predicate(elem) {
			return false
		}
	}
	return true
}

// Some checks if any element in a slice satisfies a predicate
func Some[T any](elems []T, predicate func(T) bool) bool {
	for _, elem := range elems {
		if predicate(elem) {
			return true
		}
	}
	return false
}

func FilterSlice[T any](elems []T, filter func(T) bool) []T {
	var filteredResult []T
	for _, elem := range elems {
		if filter(elem) {
			filteredResult = append(filteredResult, elem)
		}
	}

	return filteredResult
}

// UnorderedSliceRemove takes a slice and removes the element at the specified index. Does not maintain order (FAST).
func UnorderedSliceRemove[T any](ctx context.Context, elems []T, index int) []T {
	if index >= len(elems) {
		fmt.Fprintf(os.Stderr, "index out of range: %d", index)
		return elems
	}

	elems[index] = elems[len(elems)-1]
	return elems[:len(elems)-1]
}

// SliceRemove takes a slice and removes the element at the specified index. Maintains order (SLOW).
func SliceRemove[T any](ctx context.Context, elems []T, index int) []T {
	if index >= len(elems) {
		fmt.Fprintf(os.Stderr, "index out of range: %d", index)
		return elems
	}

	return append(elems[:index], elems[index+1:]...)
}

// SliceUnique takes a slice of strings or integers and returns a slice containing only unique values
func SliceUnique[T string | int](elems []T) []T {
	allKeys := make(map[T]bool)
	uniqueElems := []T{}
	for _, elem := range elems {
		if _, value := allKeys[elem]; !value {
			allKeys[elem] = true
			uniqueElems = append(uniqueElems, elem)
		}
	}
	return uniqueElems
}

// SliceDifference takes two slices and returns the elements that exist in one but not the other
func SliceDifference[T comparable](a, b []T) []T {
	set := make(map[T]bool)
	for _, item := range b {
		set[item] = true
	}
	diff := []T{}
	for _, item := range a {
		if _, value := set[item]; !value {
			diff = append(diff, item)
		}
	}
	return diff
}

// ChunkSlice takes a slice and return chunks of slice by given size.
func ChunkSlice[T any](items []T, size int) [][]T {
	var chunks [][]T
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}
