package timebased

import (
	"container/list"
	"container/ring"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"sync"
	"time"
)

type Store interface {
	collect() ([]Element, error)
	start()
	store(ctr counter)
	sorted() bool
}

type Element struct {
	Key   string
	Count uint64
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

func (s *MemStore) sorted() bool { return false }

func (s *MemStore) ingest(ctr counter) {
	if len(ctr) == 0 {
		return
	}
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
		ticker := AlignToInterval(s.interval)
		for {
			select {
			case <-ticker:
				s.rotate()
				ticker = AlignToInterval(s.interval)
			case <-s.stopChannel:
				return
			case ctr := <-s.inputChannel:
				s.ingest(ctr)
			}
		}
	}()
}

func (s *MemStore) stop() {
	close(s.stopChannel)
}

func (s *MemStore) store(ctr counter) {
	s.inputChannel <- ctr
}

func (s *MemStore) collect() ([]Element, error) {
	total := counter{}
	s.mutex.Lock()
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
	s.mutex.Unlock()
	var result []Element
	for k, v := range total {
		result = append(result, Element{Key: k, Count: v})
	}
	return result, nil
}

const keyPrefix = "timebased:store:%s"

type RedisStore struct {
	capacity     int
	interval     time.Duration
	current      string
	list         *ring.Ring
	mutex        *sync.Mutex
	prefix       string
	conn         redis.Conn
	inputChannel chan counter
	stopChannel  chan interface{}
}

func NewRedisStore(interval time.Duration, cap int, name string, conn redis.Conn) *RedisStore {
	prefix := fmt.Sprintf(keyPrefix, name)
	list := ring.New(cap + 1)
	for l, i := list, 0; i <= cap; i++ {
		l.Value = fmt.Sprintf("%d", i)
		l = l.Next()
	}
	return &RedisStore{
		capacity:     cap,
		interval:     interval,
		current:      "0",
		list:         list,
		mutex:        &sync.Mutex{},
		prefix:       prefix,
		conn:         conn,
		inputChannel: make(chan counter),
		stopChannel:  make(chan interface{}),
	}
}

func (s *RedisStore) withKey(name string) string {
	return fmt.Sprintf("%s:%s", s.prefix, name)
}

func (s *RedisStore) sorted() bool { return true }

func (s *RedisStore) ingest(ctr counter) {
	if len(ctr) == 0 {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	var values []interface{}
	for k, v := range ctr {
		values = append(values, v, k)
	}
	tKey := s.withKey("temp")    // temporary set for aggregation
	cKey := s.withKey(s.current) // current set in redis

	s.conn.Do("ZADD", redis.Args{}.Add(tKey).Add(values...)...)
	s.conn.Do("ZUNIONSTORE", cKey, 2, cKey, tKey)
	s.conn.Do("DEL", tKey)
}

func (s *RedisStore) rotate() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.list = s.list.Next()
	s.current = s.list.Value.(string)
	s.conn.Do("DEL", s.withKey(s.current))
}

func (s *RedisStore) collect() ([]Element, error) {
	var keys []interface{}
	s.mutex.Lock()
	// head is accumulating stats, don't include it here.
	for it := s.list.Next(); it != s.list; it = it.Next() {
		keys = append(keys, s.withKey(it.Value.(string)))
	}
	sKey := s.withKey("collect")
	if _, err := s.conn.Do("DEL", sKey); err != nil {
		return nil, err
	}
	args := redis.Args{}.Add(sKey).Add(len(keys)).Add(keys...)
	if _, err := s.conn.Do("ZUNIONSTORE", args...); err != nil {
		return nil, err
	}
	s.mutex.Unlock()

	var result []Element
	// return top 10 elements by default
	vals, err := redis.Values(s.conn.Do("ZRANGE", sKey, 0, 9, "REV", "WITHSCORES"))
	if err == nil {
		err = redis.ScanSlice(vals, &result)
	}
	return result, err
}

func (s *RedisStore) start() {
	go func() {
		ticker := AlignToInterval(s.interval)
		for {
			select {
			case <-ticker:
				s.rotate()
				ticker = AlignToInterval(s.interval)
			case <-s.stopChannel:
				return
			case ctr := <-s.inputChannel:
				s.ingest(ctr)
			}
		}
	}()
}

func (s *RedisStore) stop() {
	close(s.stopChannel)
}

func (s *RedisStore) store(ctr counter) {
	s.inputChannel <- ctr
}
