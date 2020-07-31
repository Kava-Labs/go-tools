package pool

import (
	"fmt"
	"log"
)

type Work struct {
	ID  int
	Job string
	// TODO: add mnemonic/keybase for each worker
}

// Worker that processes
type Worker struct {
	ID            int
	WorkerChannel chan chan Work // used to communicate between dispatcher and workers
	Channel       chan Work
	End           chan bool
}

// Start starts the worker
func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerChannel <- w.Channel // when the worker is available place channel in queue
			select {
			case job := <-w.Channel: // worker has received job
				fmt.Println("do work")
				DoWork(job.Job, w.ID) // do work
			case <-w.End:
				return
			}
		}
	}()
}

// Stop ends the worker
func (w *Worker) Stop() {
	log.Printf("worker [%d] is stopping", w.ID)
	w.End <- true
}
