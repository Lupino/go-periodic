package periodic

import (
	"github.com/Lupino/go-periodic/protocol"
	"sync/atomic"
)

// agentPacket wraps the response from the server.
type agentPacket struct {
	data []byte
	cmd  protocol.Command
	err  error
}

// Agent represents a multiplexed session over a connection.
type Agent struct {
	conn   protocol.Conn
	ID     []byte
	reader chan agentPacket

	// Use atomic.Bool for thread-safe state management without mutexes.
	isWaiting atomic.Bool
	isClosed  atomic.Bool
}

// NewAgent creates a new agent instance.
func NewAgent(conn protocol.Conn, ID []byte) *Agent {
	agent := &Agent{
		conn:   conn,
		ID:     ID,
		reader: make(chan agentPacket, 10), // Buffered to prevent blocking receiveLoop
	}
	return agent
}

// Send command and data to the server using a single memory allocation.
func (a *Agent) Send(cmd protocol.Command, data []byte) error {
	idLen := len(a.ID)
	// Pre-allocate the exact size needed: ID + Command(1 byte) + Data
	packet := make([]byte, idLen+1+len(data))

	copy(packet[0:idLen], a.ID)
	packet[idLen] = byte(cmd)
	if len(data) > 0 {
		copy(packet[idLen+1:], data)
	}

	// Set waiting state atomically before sending
	a.isWaiting.Store(true)
	return a.conn.Send(packet)
}

// Receive blocks until a command or data is received for this agent.
func (a *Agent) Receive() (cmd protocol.Command, data []byte, err error) {
	// Wait for data from the channel
	packet, ok := <-a.reader
	if !ok {
		return protocol.UNKNOWN, nil, nil
	}

	// Reset waiting state atomically
	a.isWaiting.Store(false)
	return packet.cmd, packet.data, packet.err
}

// FeedCommand pushes a server response into the agent's channel.
func (a *Agent) FeedCommand(cmd protocol.Command, dat []byte) {
	// Safety check: don't feed to a closed agent
	if a.isClosed.Load() {
		return
	}

	a.reader <- agentPacket{
		cmd:  cmd,
		data: dat,
		err:  nil,
	}
}

// FeedError propagates errors (like connection loss) to the agent.
// Fix: Correctly passes the 'err' argument to the reader.
func (a *Agent) FeedError(err error) {
	if a.isClosed.Swap(true) {
		return // Already handled
	}

	a.reader <- agentPacket{
		cmd:  protocol.UNKNOWN,
		data: nil,
		err:  err,
	}
}
