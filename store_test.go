package timebased

import (
	"testing"
	"time"
)

func TestMemStore(t *testing.T) {
	s := NewMemStore(time.Second, 3)

	c1 := counter{
		"p2": 5,
		"p3": 7,
	}

	s.ingest(c1)

	if elems, _ := s.collect(); len(elems) > 0 {
		t.Fatalf("Expect to get an empty list, but got: %v\n", elems)
	}

	s.rotate()

	if elems, _ := s.collect(); len(elems) == 0 {
		t.Fatalf("Expect to get an array of elements, but got: %v\n", elems)
	} else {
		for i := range elems {
			if elems[i].Key == "p3" && elems[i].Count != 7 {
				t.Fatalf("Expect to get 7, but got: %d\n", elems[i].Count)
			}
		}
	}

	c2 := counter{
		"p1": 13,
		"p2": 11,
	}

	s.ingest(c2)

	s.rotate()

	if elems, _ := s.collect(); len(elems) == 0 {
		t.Fatalf("Expect to get an array of elements, but got: %v\n", elems)
	} else {
		for i := range elems {
			if elems[i].Key == "p2" && elems[i].Count != 16 {
				t.Fatalf("Expect to get 16, but got: %d\n", elems[i].Count)
			}
		}
	}
}
