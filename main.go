package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var cache = map[int]Book{}
var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func main() {
	wg := &sync.WaitGroup{}
	m := &sync.RWMutex{}
	chCache := make(chan Book)
	chDb := make(chan Book)
	for i := 0; i < 10; i++ {
		id := rnd.Intn(10) + 1
		wg.Add(2)
		//make the two go routines senders
		go func(id int, wg *sync.WaitGroup, m *sync.RWMutex, ch chan<- Book) {
			if b, ok := queryCache(id, m); ok {
				ch <- b //send to cache channel
			}
			wg.Done()
		}(id, wg, m, chCache)
		go func(id int, wg *sync.WaitGroup, m *sync.RWMutex, ch chan<- Book) {
			if b, ok := queryDatabase(id, m); ok {
				ch <- b //send to db channel
			}
			wg.Done()
		}(id, wg, m, chDb)

		//new go routine that acts as the receiver
		go func(chCache, chDb <-chan Book) {
			select {
			case b := <-chCache:
				//message from cache
				fmt.Println("from cache")
				fmt.Println(b)
				<-chDb //throw away the message from the db, since we got it from the cache
			case b := <-chDb:
				//message from db
				fmt.Println("from database")
				fmt.Println(b)
			}
		}(chCache, chDb)
		time.Sleep(150 * time.Millisecond)
	}
	wg.Wait()
}

func queryCache(id int, m *sync.RWMutex) (Book, bool) {
	m.RLock() //queryCache now owns the memory location
	b, ok := cache[id]
	m.RUnlock() //release the memory location
	return b, ok
}

func queryDatabase(id int, m *sync.RWMutex) (Book, bool) {
	time.Sleep(100 * time.Millisecond)
	for _, b := range books {
		if b.ID == id {
			m.Lock() //queryDatabase now owns the memory location
			cache[id] = b
			m.Unlock()
			return b, true
		}
	}

	return Book{}, false
}
