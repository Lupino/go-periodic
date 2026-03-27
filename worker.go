package periodic

import (
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/gammazero/deque"
	"github.com/gammazero/workerpool"
	"log"
	"sync"
	"time"
)

// Worker defined a periodic worker.
type Worker struct {
	Client
	// Use sync.Map for thread-safe task storage.
	tasks        sync.Map
	agentQueueMu sync.Mutex
	agentQueue   *deque.Deque[*Agent]
	wp           *workerpool.WorkerPool
}

type workerTask struct {
	run       func(Job)
	broadcast bool
}

// NewWorker creates a worker with a specified pool size.
func NewWorker(size int) *Worker {
	if size < 1 {
		size = 1
	}
	w := new(Worker)
	w.ensureState()
	w.clientType = protocol.TYPEWORKER

	// tasks is initialized as an empty sync.Map.
	w.tasks = sync.Map{}

	w.processTask = func(msgId string, data []byte) {
		// Create a specific agent for this job assignment.
		agent := NewAgent(w.getConn(), []byte(msgId))
		agent.Send(protocol.JOBASSIGNED, nil)

		job, err := NewJob(w, data)
		if err != nil {
			return
		}

		// Use sync.Map Load for concurrent-safe task retrieval.
		if taskVal, ok := w.tasks.Load(job.FuncName); ok {
			task := taskVal.(workerTask)
			w.wp.Submit(func() {
				// Signal server to grab another job after this one is assigned.
				defer agent.Send(protocol.GRABJOB, nil)
				task.run(job)
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
	w.afterReconnect = w.restoreRegistrations

	return w
}

func (w *Worker) resetAgentQueue() {
	w.agentQueueMu.Lock()
	defer w.agentQueueMu.Unlock()
	w.agentQueue = new(deque.Deque[*Agent])
	for i := 0; i < w.wp.Size(); i++ {
		w.agentQueue.PushBack(w.newAgent())
	}
}

func (w *Worker) restoreRegistrations() {
	w.resetAgentQueue()
	w.tasks.Range(func(key, value interface{}) bool {
		funcName := key.(string)
		task := value.(workerTask)
		cmd := protocol.CANDO
		if task.broadcast {
			cmd = protocol.BROADCAST
		}
		ret, data, err := w.sendCommandAndReceive(cmd, encode8(funcName))
		if err != nil {
			log.Printf("restore func %s failed: %v\n", funcName, err)
			return true
		}
		if ret != protocol.SUCCESS {
			log.Printf("restore func %s rejected: %s\n", funcName, data)
		}
		return true
	})
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
		w.tasks.Store(funcName, workerTask{run: task}) // Atomic store.
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
		w.tasks.Store(funcName, workerTask{run: task, broadcast: true}) // Atomic store.
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
	w.resetAgentQueue()

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
			w.agentQueueMu.Lock()
			if w.agentQueue.Len() == 0 {
				w.agentQueueMu.Unlock()
				continue
			}
			agent := w.agentQueue.PopFront()
			w.agentQueue.PushBack(agent)
			w.agentQueueMu.Unlock()
			agent.Send(protocol.GRABJOB, nil)
		}

		select {
		case <-ticker.C:
			// Continue the loop on every tick.
		}
	}
}
