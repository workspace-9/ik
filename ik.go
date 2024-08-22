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

// Collect s into a slice.
func Collect[T any](s iter.Seq[T]) []T {
	return Reduce(s, func(t T, s []T) []T {
		return append(s, t)
	}, nil)
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
// for now := range Prepend(time.Now(), Chan(time.Tick(duration))) {
//   // do ticker-y logic
// }
func Prepend[T any](t T, s iter.Seq[T]) iter.Seq[T] {
  return func(yield func(T) bool) {
    if !yield(t) {
      return
    }

    s(yield)
  }
}
