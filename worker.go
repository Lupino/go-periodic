package periodic

import (
	"bytes"
	"github.com/Lupino/go-periodic/protocol"
	"time"
)

// Worker defined a client.
type Worker struct {
	Client
	tasks map[string]func(Job)
	size  int
	ch    chan Job
}

// NewWorker create a client.
func NewWorker(size int) *Worker {
	w := new(Worker)
	w.tasks = make(map[string]func(Job))
	w.size = size
	w.ch = make(chan Job, size)
	w.processTask = func(msgId string, data []byte) {
		agent := NewAgent(w.conn, []byte(msgId))
		agent.Send(protocol.GRABJOB, nil)
		job, err := NewJob(w, data)
		if err != nil {
			return
		}
		w.ch <- job
	}
	return w
}

// GrabJob from periodic server.
func (w *Worker) GrabJob(agent *Agent, waiter chan bool) {
	for {
		if len(w.ch) == 0 {
			agent.Send(protocol.GRABJOB, nil)
		}
		select {
		case <-waiter:
			break
		case <-time.After(1 * time.Second):
			break
		}
	}
}

func encode8(dat string) []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(dat)))
	buf.WriteString(dat)
	return buf.Bytes()
}

// AddFunc to periodic server.
func (w *Worker) AddFunc(funcName string, task func(Job)) error {
	w.sendCommand(protocol.CANDO, encode8(funcName))
	w.tasks[funcName] = task
	return nil
}

// Broadcast to all worker.
func (w *Worker) Broadcast(funcName string, task func(Job)) error {
	w.sendCommand(protocol.BROADCAST, encode8(funcName))
	w.tasks[funcName] = task
	return nil
}

// RemoveFunc to periodic server.
func (w *Worker) RemoveFunc(funcName string) error {
	w.sendCommand(protocol.CANTDO, encode8(funcName))
	delete(w.tasks, funcName)
	return nil
}

// Work do the task.
func (w *Worker) Work() {
	if w.size < 1 {
		w.size = 1
	}
	for i := 1; i < w.size; i++ {
		go w.work()
	}
	w.work()
}

// work do the task.
func (w *Worker) work() {
	var job Job
	var task func(Job)
	var ok bool
	var agent = w.newAgent()
	var waiter = make(chan bool, 1)
	go w.GrabJob(agent, waiter)
	for w.alive {
		job = <-w.ch
		task, ok = w.tasks[job.FuncName]
		if !ok {
			w.RemoveFunc(job.FuncName)
			job.Fail()
			continue
		}
		task(job)
		waiter <- true
	}
}
