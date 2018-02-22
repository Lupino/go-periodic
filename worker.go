package periodic

import (
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

// Worker defined a client.
type Worker struct {
	bc    *BaseClient
	tasks map[string]func(Job)
	alive bool
	wg    sync.WaitGroup
	size  int
}

// NewWorker create a client.
func NewWorker(size int) *Worker {
	w := new(Worker)
	w.tasks = make(map[string]func(Job))
	w.alive = true
	w.wg = sync.WaitGroup{}
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
func (w *Worker) GrabJob(agent *Agent) (job Job, err error) {
	agent.Send(protocol.GRABJOB, nil)
	c1 := make(chan Job, 1)
	c2 := make(chan error, 1)
	go func() {
		ret, data, _ := agent.Receive()
		if ret != protocol.JOBASSIGN {
			e := fmt.Errorf("GrabJob failed!")
			c2 <- e
			return
		}
		j, e := NewJob(w.bc, data)
		if e != nil {
			c2 <- e
			return
		}
		c1 <- j
	}()
	select {
	case job = <-c1:
		break
	case err = <-c2:
		break
	case <-time.After(1 * time.Second):
		err = fmt.Errorf("GrabJob timeout!")
		break
	}
	return
}

// AddFunc to periodic server.
func (w *Worker) AddFunc(funcName string, task func(Job)) error {
	agent := w.bc.NewAgent()
	defer w.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.CANDO, []byte(funcName))
	w.tasks[funcName] = task
	return nil
}

// RemoveFunc to periodic server.
func (w *Worker) RemoveFunc(funcName string) error {
	agent := w.bc.NewAgent()
	defer w.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.CANTDO, []byte(funcName))
	delete(w.tasks, funcName)
	return nil
}

// Work do the task.
func (w *Worker) Work() {
	var err error
	var job Job
	var task func(Job)
	var ok bool
	if w.size < 1 {
		w.size = 1
	}
	var sem = make(chan struct{}, w.size)
	var agent = w.bc.NewAgent()
	for w.alive {
		sem <- struct{}{}
		job, err = w.GrabJob(agent)
		if err != nil {
			log.Printf("GrabJob Error: %s\n", err)
			<-sem
			continue
		}
		task, ok = w.tasks[job.FuncName]
		if !ok {
			w.RemoveFunc(job.FuncName)
			job.Fail()
			<-sem
			continue
		}
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			task(job)
			<-sem
		}()
	}
}

// Close the client.
func (w *Worker) Close() {
	w.alive = false
	w.wg.Wait()
	w.bc.Close()
}
