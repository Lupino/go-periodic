package periodic

import (
	"bytes"
	"github.com/Lupino/periodic/protocol"
	"io"
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
	buf.Write(protocol.NullChar)
	buf.WriteByte(byte(cmd))
	if data != nil {
		buf.Write(protocol.NullChar)
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

type DumpAgent struct {
	conn   protocol.Conn
	ID     []byte
	w      io.Writer
	waiter *sync.RWMutex
}

// NewDumpAgent create an agent.
func NewDumpAgent(conn protocol.Conn, ID []byte, w io.Writer) *DumpAgent {
	agent := new(DumpAgent)
	agent.conn = conn
	agent.ID = ID
	agent.w = w
	agent.waiter = new(sync.RWMutex)
	return agent
}

// Send command and data to server.
func (a *DumpAgent) Send() error {
	buf := bytes.NewBuffer(nil)
	buf.Write(a.ID)
	buf.Write(protocol.NullChar)
	buf.WriteByte(byte(protocol.DUMP))
	return a.conn.Send(buf.Bytes())
}

// FeedCommand feed command from a connection or other.
func (a *DumpAgent) FeedCommand(cmd protocol.Command, data []byte) {
	if bytes.Equal(data, []byte("EOF")) {
		a.waiter.Unlock()
		return
	}
	header, _ := protocol.MakeHeader(data)
	a.w.Write(header)
	a.w.Write(data)
}

// FeedError feed error when the agent cause a error.
func (a *DumpAgent) FeedError(err error) {
	a.waiter.Unlock()
}

// Wait dump complete
func (a *DumpAgent) Wait() {
	a.waiter.Lock()
	a.waiter.Lock()
}
