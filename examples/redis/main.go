package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/nevill/timebased"
)

func main() {
	log.SetFlags(log.LstdFlags)

	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		log.Fatal(err)
	}

	// rolling every one minute, holding counters for 3 minutes [roundToMinute(now) - 3m, roundToMinute(now)]
	tbCounter := timebased.NewCounterWithStore(
		time.Second*30,
		timebased.NewRedisStore(time.Minute, 3, "productviews", conn),
	)

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			time.Sleep(time.Second * 2)
			elems, err := tbCounter.Report()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Report: %v\n", elems)
		}
	}()

	var ProductIds = []int{1, 2, 3, 4}
	log.Println("Start producer.")
	for {
		<-timebased.AlignToInterval(time.Second * 10)
		key := fmt.Sprintf("p%d", ProductIds[rand.Intn(len(ProductIds))])
		tbCounter.Inc(key)
		log.Println("Count:", key)
	}
}
