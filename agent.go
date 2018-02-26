package periodic

import (
	"bytes"
	"github.com/Lupino/go-periodic/protocol"
	"sync"
)

type Feeder interface {
	FeedCommand(protocol.Command, []byte)
	FeedError(error)
}

// Agent for client.
type Agent struct {
	conn    protocol.Conn
	ID      []byte
	locker  *sync.RWMutex
	waiter  *sync.RWMutex
	waiting bool
	data    []byte
	cmd     protocol.Command
	recived bool
	err     error
}

// NewAgent create an agent.
func NewAgent(conn protocol.Conn, ID []byte) *Agent {
	agent := new(Agent)
	agent.conn = conn
	agent.ID = ID
	agent.locker = new(sync.RWMutex)
	agent.waiter = new(sync.RWMutex)
	agent.waiting = false
	agent.data = nil
	agent.cmd = protocol.UNKNOWN
	agent.recived = false
	agent.err = nil
	return agent
}

// Send command and data to server.
func (a *Agent) Send(cmd protocol.Command, data []byte) error {
	buf := bytes.NewBuffer(nil)
	buf.Write(a.ID)
	buf.WriteByte(byte(cmd))
	if data != nil {
		buf.Write(data)
	}
	return a.conn.Send(buf.Bytes())
}

// Receive command or data from server.
func (a *Agent) Receive() (cmd protocol.Command, data []byte, err error) {
	for {
		a.locker.Lock()
		if a.recived || a.err != nil {
			a.locker.Unlock()
			break
		}
		a.waiting = true
		a.locker.Unlock()
		a.waiter.Lock()
	}

	a.locker.Lock()
	cmd = a.cmd
	data = a.data
	err = a.err
	a.recived = false
	a.locker.Unlock()
	return
}

// FeedCommand feed command from a connection or other.
func (a *Agent) FeedCommand(cmd protocol.Command, data []byte) {
	a.locker.Lock()
	defer a.locker.Unlock()
	a.data = data
	a.cmd = cmd
	a.recived = true

	if a.waiting {
		a.waiting = false
		a.waiter.Unlock()
	}
}

// FeedError feed error when the agent cause a error.
func (a *Agent) FeedError(err error) {
	a.locker.Lock()
	defer a.locker.Unlock()
	a.err = err
	if a.waiting {
		a.waiting = false
		a.waiter.Unlock()
	}
}
