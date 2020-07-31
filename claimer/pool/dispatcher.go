package pool

import (
	"log"
)

var WorkerChannel = make(chan chan Work)

// Collector receives jobs to send to workers
type Collector struct {
	Work chan Work // receives jobs to send to workers
	End  chan bool
}

func StartDispatcher(workerCount int) Collector {
	var i int
	var workers []Worker
	input := make(chan Work) // channel to recieve work
	end := make(chan bool)   // channel to spin down workers
	collector := Collector{Work: input, End: end}

	for i < workerCount {
		i++
		log.Println("starting worker: ", i)
		worker := Worker{
			ID:            i,
			Channel:       make(chan Work),
			WorkerChannel: WorkerChannel,
			End:           make(chan bool)}
		worker.Start()
		workers = append(workers, worker) // stores worker
	}

	// start collector
	go func() {
		for {
			select {
			case <-end:
				for _, w := range workers {
					w.Stop() // stop worker
				}
				return
			case work := <-input:
				worker := <-WorkerChannel // wait for available channel
				worker <- work            // dispatch work to worker
			}
		}
	}()

	return collector
}
