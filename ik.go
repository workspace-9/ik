package ik

import (
	"cmp"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"io"
	"iter"
	"slices"
)

// Map s to another type using a mapper func.
func Map[T, U any](s iter.Seq[T], mapper func(T) U) iter.Seq[U] {
	return func(yield func(U) bool) {
		s(func(t T) bool {
			return yield(mapper(t))
		})
	}
}

// Filter out unwanted values from s using filter.
func Filter[T any](s iter.Seq[T], filter func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		s(func(t T) bool {
			if filter(t) {
				return yield(t)
			}

			return true
		})
	}
}

// Reduce the values found in s using reduce and init.
func Reduce[T, U any](s iter.Seq[T], reduce func(t T, u U) U, init U) U {
	for x := range s {
		init = reduce(x, init)
	}
	return init
}

// CollectInto collects s into collection useing collectInto
func CollectInto[T, C any](s iter.Seq[T], collection C, addElement func(element T, collection C) C) C {
	return Reduce(s, addElement, collection)
}

// Collect s into a slice.
func Collect[T any](s iter.Seq[T]) []T {
	return CollectInto(s, make([]T, 0), func(t T, s []T) []T {
		return append(s, t)
	})
}

// Take the first n values from s.
func Take[T any](s iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		i := 0
		s(func(t T) bool {
			i++
			if i <= n {
				return yield(t)
			} else {
				return false
			}
		})
	}
}

// Sql creates an iter.Seq from sql.Rows.
// Each row gives the opportunity to scan the row into a struct.
// Takes ownership of the rows.
func Sql(rows *sql.Rows) iter.Seq[func(...any) error] {
	return func(yield func(func(...any) error) bool) {
		defer rows.Close()
		for rows.Next() {
			if !yield(rows.Scan) {
				return
			}
		}
	}
}

// Csv reads the rows of a csv file.
// Takes ownership of r.
func Csv(r io.ReadCloser) iter.Seq2[[]string, error] {
	return func(yield func([]string, error) bool) {
		reader := csv.NewReader(r)
		defer r.Close()
		for {
			row, err := reader.Read()
			if err == io.EOF {
				return
			}

			if err != nil {
				if !yield(nil, err) {
					return
				}
			} else if !yield(row, nil) {
				return
			}
		}
	}
}

// Unique yields unique values.
func Unique[T comparable](s iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		m := map[T]struct{}{}
		s(func(t T) bool {
			if _, seen := m[t]; !seen {
				m[t] = struct{}{}
				return yield(t)
			}

			return true
		})
	}
}

// Sorted sorts the values in T using their default order.
func Sorted[T cmp.Ordered](s iter.Seq[T]) iter.Seq[T] {
	return slices.Values(slices.Sorted(s))
}

// SortedBy sorts the values in s using order.
func SortedBy[T any](s iter.Seq[T], order func(a, b T) int) iter.Seq[T] {
	ret := Collect(s)
	slices.SortFunc(ret, order)
	return slices.Values(ret)
}

// Chan creats an iter.Seq from a channel.
func Chan[T any](ch <-chan T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for x := range ch {
			if !yield(x) {
				return
			}
		}
	}
}

// Chunk processes s in chunks of size chunkSize.
func Chunk[T any](s iter.Seq[T], chunkSize int) iter.Seq[[]T] {
	if chunkSize < 1 {
		panic("Chunk size too small")
	}

	return func(yield func([]T) bool) {
		var chunk []T = make([]T, chunkSize)
		idx := 0
		s(func(t T) bool {
			chunk[idx] = t
			idx++
			if idx == chunkSize {
				idx = 0
				return yield(chunk)
			}

			return true
		})

		if idx != 0 {
			yield(chunk[:idx])
		}
	}
}

// SliceRef returns an iter.Seq which yields references to values in the slice.
// This is useful when the slice contains large values which shouldn't be
// copied.
func SliceRef[T any](s []T) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for idx := range s {
			if !yield(&s[idx]) {
				return
			}
		}
	}
}

// Enumerate s.
func Enumerate[T any](s iter.Seq[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		idx := 0
		s(func(t T) bool {
			ret := yield(idx, t)
			idx++
			return ret
		})
	}
}

// Json creates an iter.Seq2 of json.Tokens and errors.
func Json(r io.ReadCloser) iter.Seq2[json.Token, error] {
	return func(yield func(json.Token, error) bool) {
		dec := json.NewDecoder(r)
		defer r.Close()

		for {
			tok, err := dec.Token()
			if err != nil {
				if err == io.EOF {
					return
				}

				if !yield(nil, err) {
					return
				}
			} else if !yield(tok, nil) {
				return
			}
		}
	}
}

// Elide attempts to ignore errors in s and panics if they occur.
func Elide[T any](s iter.Seq2[T, error]) iter.Seq[T] {
	return func(yield func(T) bool) {
		s(func(t T, err error) bool {
			if err != nil {
				panic(err)
			}

			return yield(t)
		})
	}
}

// Prepend t to s.
// This helps solve one of my least favorite features of ticker:
//
//	for now := range Prepend(time.Now(), Chan(time.Tick(duration))) {
//	  // do ticker-y logic
//	}
func Prepend[T any](t T, s iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		if !yield(t) {
			return
		}

		s(yield)
	}
}

// Append t to s.
func Append[T any](t T, s iter.Seq[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		var cont bool
		s(func(t T) bool {
			cont = yield(t)
			return cont
		})

		if cont {
			yield(t)
		}
	}
}

// Tee the values in s to yield1, returning a new Seq which can be
// consumed elsewhere.
func Tee[T any](s iter.Seq[T], yield1 func(t T) bool) iter.Seq[T] {
	return func(yield2 func(T) bool) {
		var continue1, continue2 bool = true, true
		s(func(t T) bool {
			if continue1 {
				continue1 = yield1(t)
			}

			if continue2 {
				continue2 = yield2(t)
			}

			return continue1 || continue2
		})
	}
}

// Pair of values.
type Pair[K, V any] struct {
	K K
	V V
}

// Seq2Seq makes a Seq out of a Seq2
func Seq2Seq[K, V any](s iter.Seq2[K, V]) iter.Seq[Pair[K, V]] {
	return func(yield func(Pair[K, V]) bool) {
		s(func(k K, v V) bool {
			return yield(Pair[K, V]{k, v})
		})
	}
}

// Until yields values until until returns false.
func Until[T any](s iter.Seq[T], until func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		s(func(t T) bool {
			if until(t) {
				return yield(t)
			}

			return false
		})
	}
}

// Skip the first n values.
func Skip[T any](s iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		i := 0
		s(func(t T) bool {
			if i < n {
				i++
				return true
			}

			return yield(t)
		})
	}
}

// First returns the first value in s which matches the given predicate.
func First[T any](s iter.Seq[T], predicate func(T) bool) (t T, ok bool) {
	s(func(iterT T) bool {
		if predicate(iterT) {
			t = iterT
			ok = true
			return false
		}

		return true
	})
	return
}

// Max returns the maximum element in s
func Max[T cmp.Ordered](s iter.Seq[T]) (t T, ok bool) {
	s(func(iterT T) bool {
		if !ok || t > iterT {
			ok = true
			t = iterT
		}
		return true
	})

	return
}

// Max returns the maximum element in s
func Min[T cmp.Ordered](s iter.Seq[T]) (t T, ok bool) {
	s(func(iterT T) bool {
		if !ok || t < iterT {
			ok = true
			t = iterT
		}
		return true
	})

	return
}

// IsSorted returns:
// 1: array is sorted in ascending order
// 0: array is unsorted
// -1: array is sorted in descending order
func IsSorted[T cmp.Ordered](s iter.Seq[T]) int {
	return IsSortedBy(s, cmp.Compare[T])
}

// IsSortedBy returns:
// 1: array is sorted in ascending order
// 0: array is unsorted
// -1: array is sorted in descending order
func IsSortedBy[T any](s iter.Seq[T], order func(a, b T) int) int {
	var lastV struct {
		v       T
		ok      bool
		cmpSign struct {
			sign int
			ok   bool
		}
	}
	for v := range s {
		if !lastV.ok {
			lastV.ok = true
			lastV.v = v
			goto loopEnd
		}

		if !lastV.cmpSign.ok {
			lastV.cmpSign.ok = true
			lastV.cmpSign.sign = order(v, lastV.v)
			goto loopEnd
		}

		if order(v, lastV.v) != lastV.cmpSign.sign {
			return 0
		}

	loopEnd:
		lastV.v = v
		lastV.ok = true
	}

	if !lastV.cmpSign.ok {
		return 1
	}

	return lastV.cmpSign.sign
}
