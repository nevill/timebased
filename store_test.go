package timebased

import (
	"github.com/gomodule/redigo/redis"
	"testing"
	"time"
)

func TestMemStore(t *testing.T) {
	s := NewMemStore(time.Second, 3)
	testWithStore(s, t)
}

func testWithStore(s Store, t *testing.T) {
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

func TestRedisStore(t *testing.T) {
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		t.Fatal(err)
	}
	s := NewRedisStore(time.Minute, 3, "testing", conn)
	testWithStore(s, t)
}
