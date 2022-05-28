package timebased

import (
	"container/list"
	"sort"
	"sync"
	"time"
)

//TODO
// 1. add variable to set if round to minute to start

type counter map[string]uint64

type Element struct {
	Key   string
	Count uint64
}

func NewCounterWithStore(interval time.Duration, store *MemStore) *TimedCounter {
	c := newTimedCounter(interval)
	c.store = store
	c.start()
	store.start()
	return c
}

// TimedCounter every interval passes, try to store and reset the counter.
type TimedCounter struct {
	store       *MemStore
	keyChannel  chan string
	stopChannel chan interface{}
	current     counter
	interval    time.Duration
	mutex       *sync.Mutex
}

func newTimedCounter(interval time.Duration) *TimedCounter {
	return &TimedCounter{
		keyChannel:  make(chan string),
		stopChannel: make(chan interface{}),
		interval:    interval,
		current:     counter{},
		mutex:       &sync.Mutex{},
	}
}

func (tc *TimedCounter) Inc(key string) {
	tc.keyChannel <- key
}

// increase the key's count stored in tc.current
func (tc *TimedCounter) increase(key string) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	if val, ok := tc.current[key]; ok {
		tc.current[key] = val + 1
	} else {
		tc.current[key] = 1
	}
}

// set tc.current to a new counter, start next cycle of counting
func (tc *TimedCounter) nextCycle() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	if tc.store != nil {
		tc.store.Store(tc.current)
	}
	tc.current = counter{}
}

func (tc *TimedCounter) start() {
	go func() {
		ticker := time.NewTicker(tc.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tc.nextCycle()
			case key := <-tc.keyChannel:
				tc.increase(key)
			case <-tc.stopChannel:
				return
			}
		}
	}()
}

func (tc *TimedCounter) Stop() {
	close(tc.stopChannel)
}

func (tc *TimedCounter) Report() (result []Element) {
	if tc.store == nil {
		return
	}
	for k, v := range tc.store.collect() {
		result = append(result, Element{Key: k, Count: v})
	}

	sort.SliceStable(result, func(i, j int) bool { return result[i].Count >= result[j].Count })
	return
}

// MemStore maintains a list of counter, every interval passes,
// the oldest (header) counter will be removed, and append a new one at the end of the list.
type MemStore struct {
	capacity     int
	interval     time.Duration
	list         *list.List
	current      counter
	mutex        *sync.Mutex
	inputChannel chan counter
	stopChannel  chan interface{}
}

func NewMemStore(interval time.Duration, cap int) *MemStore {
	s := &MemStore{
		capacity:     cap + 1, // add one more element for the counter which is still counting
		interval:     interval,
		list:         list.New(),
		current:      counter{},
		mutex:        &sync.Mutex{},
		inputChannel: make(chan counter),
		stopChannel:  make(chan interface{}),
	}
	s.list.PushBack(s.current)
	return s
}

func (s *MemStore) ingest(ctr counter) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	slot := s.current
	for key, change := range ctr {
		if val, ok := slot[key]; ok {
			slot[key] = val + change
		} else {
			slot[key] = change
		}
	}
}

func (s *MemStore) rotate() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.list.Len() >= s.capacity {
		e := s.list.Front()
		s.list.Remove(e)
	}
	s.current = counter{}
	s.list.PushBack(s.current)
}

func (s *MemStore) start() {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.rotate()
			case <-s.stopChannel:
				return
			case ctr := <-s.inputChannel:
				s.ingest(ctr)
			}
		}
	}()
}

func (s *MemStore) Stop() {
	close(s.stopChannel)
}

func (s *MemStore) Store(ctr counter) {
	s.inputChannel <- ctr
}

// collect sorts the keys based on number of count in descending order.
func (s *MemStore) collect() counter {
	total := counter{}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// don't count the last element
	for e := s.list.Front(); e != nil && e.Next() != nil; e = e.Next() {
		for k, change := range e.Value.(counter) {
			if val, ok := total[k]; ok {
				total[k] = val + change
			} else {
				total[k] = change
			}
		}
	}
	return total
}
