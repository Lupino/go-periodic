package periodic

import (
	"bytes"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/gammazero/deque"
	"github.com/gammazero/workerpool"
	"time"
)

// Worker defined a client.
type Worker struct {
	Client
	tasks      map[string]func(Job)
	agentQueue *deque.Deque[*Agent]
	wp         *workerpool.WorkerPool
}

// NewWorker create a client.
func NewWorker(size int) *Worker {
	w := new(Worker)
	w.tasks = make(map[string]func(Job))
	w.processTask = func(msgId string, data []byte) {
		agent := NewAgent(w.conn, []byte(msgId))
		agent.Send(protocol.GRABJOB, nil)
		job, err := NewJob(w, data)
		if err != nil {
			return
		}
		task, ok := w.tasks[job.FuncName]
		if ok {
			w.wp.Submit(func() {
				task(job)
			})
		} else {
			w.RemoveFunc(job.FuncName)
			job.Fail()
		}
	}

	w.agentQueue = deque.New[*Agent](size)
	w.wp = workerpool.New(size)

	return w
}

func encode8(dat string) []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(dat)))
	buf.WriteString(dat)
	return buf.Bytes()
}

// AddFunc to periodic server.
func (w *Worker) AddFunc(funcName string, task func(Job)) error {
	ret, data, _ := w.sendCommandAndReceive(protocol.CANDO, encode8(funcName))
	if ret == protocol.SUCCESS {
		w.tasks[funcName] = task
		return nil
	}
	return fmt.Errorf("AddFunc error: %s", data)
}

// Broadcast to all worker.
func (w *Worker) Broadcast(funcName string, task func(Job)) error {
	ret, data, _ := w.sendCommandAndReceive(protocol.BROADCAST, encode8(funcName))
	if ret == protocol.SUCCESS {
		w.tasks[funcName] = task
		return nil
	}
	return fmt.Errorf("Broadcast error: %s", data)
}

// RemoveFunc to periodic server.
func (w *Worker) RemoveFunc(funcName string) error {
	ret, data, _ := w.sendCommandAndReceive(protocol.CANTDO, encode8(funcName))
	if ret == protocol.SUCCESS {
		delete(w.tasks, funcName)
		return nil
	}
	return fmt.Errorf("RemoveFunc error: %s", data)
}

// Work do the task.
func (w *Worker) Work() {
	for i := 0; i < w.wp.Size(); i++ {
		var agent = w.newAgent()
		w.agentQueue.PushBack(agent)
	}
	for {
		agent := w.agentQueue.PopFront()
		w.agentQueue.PushBack(agent)
		if w.wp.WaitingQueueSize() < 1 {
			agent.Send(protocol.GRABJOB, nil)
		}
		select {
		case <-time.After(1 * time.Second):
			break
		}
	}
}
