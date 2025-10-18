package pool

import (
	"os"
	"strconv"
	"sync"
)

var (
	goroutineLimit chan struct{}
	wg             sync.WaitGroup
)

func init() {
	maxGoroutines, err := strconv.Atoi(os.Getenv("MAX_GOROUTINES"))
	if err != nil || maxGoroutines <= 0 {
		maxGoroutines = 50
	}
	goroutineLimit = make(chan struct{}, maxGoroutines)
}

func Submit(task func()) {
	goroutineLimit <- struct{}{}
	wg.Add(1)
	go func() {
		defer func() {
			<-goroutineLimit
			wg.Done()
		}()
		task()
	}()
}

func Wait() {
	wg.Wait()
}
