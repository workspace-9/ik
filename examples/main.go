package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	_ "github.com/proullon/ramsql/driver"
	. "github.com/workspace-9/ik"
)

func main() {
	s := []int{1, 2, 3, 4, 5, 6, 4, 3, 2}
	// Sum the squares of the even numbers in the array
	fmt.Println(Reduce(Map(Filter(slices.Values(s), func(i int) bool {
		return i&0x1 == 0
	}), func(i int) int {
		return i * i
	}), func(cur, accum int) int {
		return cur + accum
	}, 0))

	// Take the first two and collect them into an array
	fmt.Println(Collect(Take(slices.Values(s), 2)))

	db, _ := sql.Open("ramsql", "test")
	db.Exec(`CREATE TABLE TEST(
    a int,
    b text
  );`)
	db.Exec(`INSERT INTO TEST(
    a,
    b
  ) VALUES ($1, $2);`, 314, "hello")
	db.Exec(`INSERT INTO TEST(
    a,
    b
  ) VALUES ($1, $2);`, 159, "goodbye")
	rows, _ := db.Query(`SELECT * FROM TEST;`)
	for s := range Sql(rows) {
		var a int
		var b string
		if err := s(&a, &b); err != nil {
			panic(err)
		}

		fmt.Println(a, b)
	}

	for row := range Csv(io.NopCloser(strings.NewReader(`
a,b,c,d,e,f,g
1,2,3,4,5,6,7
  `))) {
		fmt.Println(row)
	}

	fmt.Println(Collect(Unique(slices.Values(s))))
	fmt.Println(Collect(Sorted(slices.Values(s))))
	fmt.Println(Collect(SortedBy(slices.Values(s), func(a, b int) int {
		if a < b {
			return 1
		} else if a == b {
			return 0
		}
		return -1
	})))

	ch := make(chan string)
	go func() {
		ch <- "to"
		ch <- "be"
		ch <- "or"
		ch <- "not"
		ch <- "to"
		ch <- "be"
		ch <- "that"
		ch <- "is"
		ch <- "the"
		ch <- "question"
		close(ch)
	}()
	for chunk := range Chunk(Chan(ch), 4) {
		fmt.Println(len(chunk), chunk)
	}

	fmt.Println(Collect(SliceRef(s)))

	for tok := range Tee(Elide(Json(io.NopCloser(strings.NewReader(`[1, 2, 3, 4, 5]`)))), func(t json.Token) bool {
		fmt.Printf("%T\n", t)
		return false
	}) {
		fmt.Println(tok)
	}

	fmt.Println(Collect(Prepend(1, slices.Values(s))))
	fmt.Println(Collect(Append(100, slices.Values(s))))

	fmt.Println(IsSorted(slices.Values([]int{1, 2, 3, 4})))
	fmt.Println(IsSorted(slices.Values([]int{1, 2, 4, 3})))
	fmt.Println(IsSorted(slices.Values([]int{4, 3, 2, 1})))
	fmt.Println(Min(slices.Values([]int{-1, 190, -3})))
	fmt.Println(Max(slices.Values([]int{-1, 190, -3})))
	fmt.Println(First(slices.Values([]int{-1, 190, -3}), func(i int) bool { return i&0x1 == 0 }))
}
