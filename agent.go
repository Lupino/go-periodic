package periodic

import (
	"bytes"
	"github.com/Lupino/go-periodic/protocol"
	"sync"
)

type data struct {
	data []byte
	cmd  protocol.Command
	err  error
}

// Agent for client.
type Agent struct {
	conn    protocol.Conn
	ID      []byte
	locker  *sync.RWMutex
	waiter  *sync.RWMutex
	waiting bool
	recived bool
	reader  chan data
}

// NewAgent create an agent.
func NewAgent(conn protocol.Conn, ID []byte) *Agent {
	agent := new(Agent)
	agent.conn = conn
	agent.ID = ID
	agent.locker = new(sync.RWMutex)
	agent.waiter = new(sync.RWMutex)
	agent.waiting = false
	agent.reader = make(chan data, 10)
	agent.recived = false
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
	dat := <-a.reader
	cmd = dat.cmd
	data = dat.data
	err = dat.err
	return
}

// FeedCommand feed command from a connection or other.
func (a *Agent) FeedCommand(cmd protocol.Command, dat []byte) {
	a.reader <- data{cmd: cmd, data: dat, err: nil}
}

// FeedError feed error when the agent cause a error.
func (a *Agent) FeedError(err error) {
	a.reader <- data{cmd: protocol.UNKNOWN, data: nil, err: nil}
}
