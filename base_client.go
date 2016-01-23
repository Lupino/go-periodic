package periodic

import (
	"github.com/Lupino/periodic/protocol"
	"github.com/satori/go.uuid"
	"io"
	"log"
	"net"
	"sync"
)

// BaseClient defined base client.
type BaseClient struct {
	agents map[string]*Agent
	conn   protocol.Conn
	locker *sync.RWMutex
	alive  bool
}

// NewBaseClient create a base client.
func NewBaseClient(conn net.Conn, clientType protocol.ClientType) *BaseClient {
	c := new(BaseClient)
	c.agents = make(map[string]*Agent)
	c.alive = true
	c.locker = new(sync.RWMutex)
	c.conn = protocol.NewClientConn(conn)
	c.conn.Send(clientType.Bytes())
	return c
}

// RemoveAgent remove a agent by a agentID
func (c *BaseClient) RemoveAgent(agentID []byte) {
	c.locker.Lock()
	defer c.locker.Unlock()
	delete(c.agents, string(agentID))
}

// NewAgent create a new agent with an uuid
func (c *BaseClient) NewAgent() *Agent {
	c.locker.Lock()
	defer c.locker.Unlock()
	agentID := uuid.NewV4().Bytes()
	agent := NewAgent(c.conn, agentID)
	c.agents[string(agentID)] = agent
	return agent
}

// ReceiveLoop a loop on receive data.
func (c *BaseClient) ReceiveLoop() {
	c.alive = true
	for c.alive {
		payload, err := c.conn.Receive()
		if err != nil {
			log.Fatal(err)
		}
		agentID, cmd, data := protocol.ParseCommand(payload)
		c.locker.Lock()
		agent, ok := c.agents[string(agentID)]
		if !ok {
			log.Printf("Agent: %s not found.", agentID)
			continue
		}
		agent.FeedCommand(cmd, data)
		c.locker.Unlock()
	}
}

// Close the base client.
func (c *BaseClient) Close() {
	c.locker.Lock()
	defer c.locker.Unlock()
	for _, agent := range c.agents {
		agent.FeedError(io.EOF)
	}
	c.alive = false
}
