package timebased

import (
	"sort"
	"sync"
	"time"
)

type counter map[string]uint64

func NewCounterWithStore(interval time.Duration, store Store) *TimedCounter {
	c := newTimedCounter(interval)
	c.store = store
	c.start()
	store.start()
	return c
}

// TimedCounter every interval passes, try to store and reset the counter.
type TimedCounter struct {
	store       Store
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
		tc.store.store(tc.current)
	}
	tc.current = counter{}
}

func (tc *TimedCounter) start() {
	go func() {
		ticker := AlignToInterval(tc.interval)
		for {
			select {
			case <-ticker:
				tc.nextCycle()
				ticker = AlignToInterval(tc.interval)
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

func (tc *TimedCounter) Report() (result []Element, err error) {
	if tc.store == nil {
		return
	}
	result, err = tc.store.collect()
	if err != nil {
		return
	}

	if !tc.store.sorted() {
		sort.SliceStable(result, func(i, j int) bool { return result[i].Count >= result[j].Count })
	}
	return
}

func AlignToInterval(interval time.Duration) <-chan time.Time {
	from := time.Now().Local().UnixNano()
	unit := interval.Nanoseconds()
	return time.After(time.Duration((from/unit+1)*unit-from) + time.Microsecond*10)
}
