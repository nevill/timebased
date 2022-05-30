package main

import (
	"fmt"
	"github.com/nevill/timebased"
	"log"
	"math/rand"
	"time"
)

func main() {
	log.SetFlags(log.LstdFlags)

	// rolling every one minute, holding counters for 3 minutes [roundToMinute(now) - 3m, roundToMinute(now)]
	tbCounter := timebased.NewCounterWithStore(time.Second*30, timebased.NewMemStore(time.Minute, 3))

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			time.Sleep(time.Second * 3)
			elems, err := tbCounter.Report()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%v\n", elems)
		}
	}()

	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()
	var ProductIds = []int{1, 2, 3, 4}
	log.Println("start producer")
	for range ticker.C {
		key := fmt.Sprintf("p%d", ProductIds[rand.Intn(len(ProductIds))])
		log.Println("count", key)
		tbCounter.Inc(key)
	}
}
