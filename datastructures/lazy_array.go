package datastructures

import "iter"

type LazyArray[T any] struct {
	len int
	get func(i int) T
}

func NewLazyArray[T any](length int, generator func(int) T) LazyArray[T] {
	return LazyArray[T]{
		len: length,
		get: generator,
	}
}

func (a LazyArray[T]) Len() int {
	return a.len
}

func (a LazyArray[T]) Get(i int) T {
	if i < 0 || i >= a.len {
		panic("index out of range")
	}
	return a.get(i)
}

func (a LazyArray[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i := 0; i < a.len; i++ {
			if !yield(i, a.get(i)) {
				return
			}
		}
	}
}

func (a LazyArray[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := 0; i < a.len; i++ {
			if !yield(a.get(i)) {
				return
			}
		}
	}
}


func NewLazyArrayFromSlice[T any](slice []T) LazyArray[T] {
	return NewLazyArray(len(slice), func(i int) T {
		return slice[i]
	})
}
