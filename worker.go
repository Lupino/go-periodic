package periodic

import (
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/gammazero/deque"
	"github.com/gammazero/workerpool"
	"sync"
	"time"
)

// Worker defined a periodic worker.
type Worker struct {
	Client
	// Use sync.Map for thread-safe task storage.
	tasks      sync.Map
	agentQueue *deque.Deque[*Agent]
	wp         *workerpool.WorkerPool
}

// NewWorker creates a worker with a specified pool size.
func NewWorker(size int) *Worker {
	w := new(Worker)
	// tasks is initialized as an empty sync.Map.
	w.tasks = sync.Map{}

	w.processTask = func(msgId string, data []byte) {
		// Create a specific agent for this job assignment.
		agent := NewAgent(w.conn, []byte(msgId))
		agent.Send(protocol.JOBASSIGNED, nil)

		job, err := NewJob(w, data)
		if err != nil {
			return
		}

		// Use sync.Map Load for concurrent-safe task retrieval.
		if taskVal, ok := w.tasks.Load(job.FuncName); ok {
			task := taskVal.(func(Job))
			w.wp.Submit(func() {
				// Signal server to grab another job after this one is assigned.
				defer agent.Send(protocol.GRABJOB, nil)
				task(job)
			})
		} else {
			// If the function is not found locally, tell the server we can't do it.
			w.RemoveFunc(job.FuncName)
			job.Fail()
			agent.Send(protocol.GRABJOB, nil)
		}
	}

	w.agentQueue = new(deque.Deque[*Agent])
	w.wp = workerpool.New(size)

	return w
}

// encode8 encodes a string into a length-prefixed byte slice.
// Optimization: Uses pre-allocated slice instead of bytes.Buffer.
func encode8(dat string) []byte {
	length := len(dat)
	if length > 255 {
		length = 255
	}
	buf := make([]byte, length+1)
	buf[0] = byte(length)
	copy(buf[1:], dat[:length])
	return buf
}

// AddFunc registers a function to the periodic server.
func (w *Worker) AddFunc(funcName string, task func(Job)) error {
	ret, data, err := w.sendCommandAndReceive(protocol.CANDO, encode8(funcName))
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		w.tasks.Store(funcName, task) // Atomic store.
		return nil
	}
	return fmt.Errorf("AddFunc error: %s", data)
}

// Broadcast registers a broadcast function to the periodic server.
func (w *Worker) Broadcast(funcName string, task func(Job)) error {
	ret, data, err := w.sendCommandAndReceive(protocol.BROADCAST, encode8(funcName))
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		w.tasks.Store(funcName, task) // Atomic store.
		return nil
	}
	return fmt.Errorf("Broadcast error: %s", data)
}

// RemoveFunc unregisters a function from the periodic server.
func (w *Worker) RemoveFunc(funcName string) error {
	ret, data, err := w.sendCommandAndReceive(protocol.CANTDO, encode8(funcName))
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		w.tasks.Delete(funcName) // Atomic delete.
		return nil
	}
	return fmt.Errorf("RemoveFunc error: %s", data)
}

// Work enters the loop to request and process tasks.
func (w *Worker) Work() {
	// Initialize agents based on the worker pool size.
	for i := 0; i < w.wp.Size(); i++ {
		var agent = w.newAgent()
		w.agentQueue.PushBack(agent)
	}

	// Use a Ticker instead of time.After to save resources.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		// Use atomic load from Client to check if still connected.
		if !w.alive.Load() {
			break
		}

		// Rotate through agents to grab jobs if the pool is not full.
		if w.wp.WaitingQueueSize() < 1 {
			agent := w.agentQueue.PopFront()
			agent.Send(protocol.GRABJOB, nil)
			w.agentQueue.PushBack(agent)
		}

		select {
		case <-ticker.C:
			// Continue the loop on every tick.
		}
	}
}
