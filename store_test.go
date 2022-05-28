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

	if ctr := s.collect(); len(ctr) > 0 {
		t.Fatalf("Expect to get an empty counter, but got: %v\n", ctr)
	}

	s.rotate()

	if ctr := s.collect(); ctr["p3"] != 7 {
		t.Fatalf("Expect to get 7, but got: %d\n", ctr["p3"])
	}

	c2 := counter{
		"p1": 13,
		"p2": 11,
	}

	s.ingest(c2)

	s.rotate()

	if ctr := s.collect(); ctr["p2"] != 16 {
		t.Fatalf("Expect to get 16, but got: %d\n", ctr["p2"])
	}
}
