package datastructures_test

import (
	"fmt"
	"testing"

	"github.com/alecthomas/assert/v2"
	ds "github.com/nstefan13/bittorrent/datastructures"
	"github.com/nstefan13/bittorrent/helpers"
)

func TestLazyArray(t *testing.T) {
	t.Run("returns correct length and computes values using generator", func(t *testing.T) {
		called := make([]bool, 3)
		arr := ds.NewLazyArray(3, func(i int) int {
			called[i] = true
			return i * 10
		})

		helpers.AssertEqual(t, 3, arr.Len())
		helpers.AssertEqual(t, false, called[0])
		helpers.AssertEqual(t, false, called[1])
		helpers.AssertEqual(t, false, called[2])

		helpers.AssertEqual(t, 0, arr.Get(0))
		helpers.AssertEqual(t, true, called[0])
		helpers.AssertEqual(t, false, called[1])

		helpers.AssertEqual(t, 20, arr.Get(2))
		helpers.AssertEqual(t, true, called[2])
		helpers.AssertEqual(t, false, called[1])

		helpers.AssertEqual(t, 10, arr.Get(1))
		helpers.AssertEqual(t, true, called[1])
	})

	t.Run("range over function", func(t *testing.T) {
		called := make([]bool, 3)
		arr := ds.NewLazyArray(3, func(i int) int {
			called[i] = true
			return i * 10
		})

		var indices []int
		var values []int

		for i, v := range arr.All() {
			indices = append(indices, i)
			values = append(values, v)
		}

		helpers.AssertEqual(t, []int{0, 1, 2}, indices)
		helpers.AssertEqual(t, []int{0, 10, 20}, values)
		helpers.AssertEqual(t, true, called[0])
		helpers.AssertEqual(t, true, called[1])
		helpers.AssertEqual(t, true, called[2])
	})

	t.Run("Values() yields just values", func(t *testing.T) {
		arr := ds.NewLazyArray(3, func(i int) int {
			return i * 10
		})

		var values []int
		for v := range arr.Values() {
			values = append(values, v)
		}

		helpers.AssertEqual(t, []int{0, 10, 20}, values)
	})

	t.Run("NewLazyArrayFromSlice() creates a LazyArray from a standard slice", func(t *testing.T) {
		slice := []int{100, 200, 300}
		arr := ds.NewLazyArrayFromSlice(slice)

		helpers.AssertEqual(t, 3, arr.Len())
		helpers.AssertEqual(t, 100, arr.Get(0))
		helpers.AssertEqual(t, 200, arr.Get(1))
		helpers.AssertEqual(t, 300, arr.Get(2))
	})

	t.Run("accessing a negative index panics", func(t *testing.T) {
		arr := ds.NewLazyArray(5, func(i int) int {
			return i
		})

		assert.Panics(t, func() {
			arr.Get(-1)
		})
	})

	t.Run("accessing an index equal to or greater than length panics", func(t *testing.T) {
		arr := ds.NewLazyArray(3, func(i int) string {
			return fmt.Sprintf("item-%d", i)
		})

		assert.Panics(t, func() {
			arr.Get(3)
		})
		assert.Panics(t, func() {
			arr.Get(10)
		})
	})

	t.Run("empty LazyArray has length 0 and any access panics", func(t *testing.T) {
		arr := ds.NewLazyArray(0, func(i int) int {
			return i * 10
		})

		helpers.AssertEqual(t, 0, arr.Len())
		assert.Panics(t, func() {
			arr.Get(0)
		})
	})
}
