package periodic

import (
	"github.com/Lupino/go-periodic/protocol"
	"io/ioutil"
	// "log"
	"bytes"
	"net"
	"strings"
	"time"
)

// Worker defined a client.
type Worker struct {
	bc    *BaseClient
	tasks map[string]func(Job)
	alive bool
	size  int
}

// NewWorker create a client.
func NewWorker(size int) *Worker {
	w := new(Worker)
	w.tasks = make(map[string]func(Job))
	w.alive = true
	w.size = size
	return w
}

// Connect a periodic server.
func (w *Worker) Connect(addr string, key ...string) error {
	parts := strings.SplitN(addr, "://", 2)
	conn, err := net.Dial(parts[0], parts[1])
	if err != nil {
		return err
	}
	if len(key) > 0 && len(key[0]) > 0 {
		if keyBuf, err := ioutil.ReadFile(key[0]); err != nil {
			return err
		} else {
			w.bc = NewBaseClient(protocol.NewXORConn(conn, keyBuf), protocol.TYPEWORKER)
		}
	} else {
		w.bc = NewBaseClient(conn, protocol.TYPEWORKER)
	}
	go w.bc.ReceiveLoop()
	return nil
}

// Ping a periodic server.
func (w *Worker) Ping() bool {
	agent := w.bc.NewAgent()
	defer w.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.PING, nil)
	ret, _, _ := agent.Receive()
	if ret == protocol.PONG {
		return true
	}
	return false
}

// GrabJob from periodic server.
func (w *Worker) GrabJob(agent *Agent, ch chan Job, waiter chan bool) {
	go func() {
		for {
			ret, data, _ := agent.Receive()
			if ret != protocol.JOBASSIGN {
				continue
			}
			j, e := NewJob(w.bc, data)
			if e != nil {
				continue
			}
			ch <- j
		}
	}()
	go func() {
		for {
			if len(ch) == 0 {
				agent.Send(protocol.GRABJOB, nil)
			}
			select {
			case <-waiter:
				break
			case <-time.After(1 * time.Second):
				break
			}
		}
	}()
}

func encode8(dat string) []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(dat)))
	buf.WriteString(dat)
	return buf.Bytes()
}

// AddFunc to periodic server.
func (w *Worker) AddFunc(funcName string, task func(Job)) error {
	agent := w.bc.NewAgent()
	defer w.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.CANDO, encode8(funcName))
	w.tasks[funcName] = task
	return nil
}

// Broadcast to all worker.
func (w *Worker) Broadcast(funcName string, task func(Job)) error {
	agent := w.bc.NewAgent()
	defer w.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.BROADCAST, encode8(funcName))
	w.tasks[funcName] = task
	return nil
}

// RemoveFunc to periodic server.
func (w *Worker) RemoveFunc(funcName string) error {
	agent := w.bc.NewAgent()
	defer w.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.CANTDO, encode8(funcName))
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
	var agent = w.bc.NewAgent()
	var ch = make(chan Job, 100)
	var waiter = make(chan bool, 10)
	w.GrabJob(agent, ch, waiter)
	for w.alive {
		job = <-ch
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

// Close the client.
func (w *Worker) Close() {
	w.alive = false
	w.bc.Close()
}
