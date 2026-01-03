package periodic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
	"io"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client defines the base client.
type Client struct {
	// Use sync.Map for high-concurrency agent management.
	agents      sync.Map
	conn        protocol.Conn
	// Use atomic.Bool for lock-free state checks.
	alive       atomic.Bool
	agentLastId uint32
	processTask func(string, []byte)
}

// NewClient creates a new client.
func NewClient() *Client {
	return new(Client)
}

// initClient initializes the base client.
func (c *Client) initClient(conn net.Conn, clientType protocol.ClientType) {
	c.agents = sync.Map{} // Re-initialize the sync.Map
	c.alive.Store(true)   // Set alive state atomically
	c.agentLastId = 0
	c.conn = protocol.NewClientConn(conn)
	c.conn.Send(clientType.Bytes())
	c.conn.Receive()
}

// Clone clones the base client.
// Note: It continues to share the same underlying connection and agents.
func (c *Client) Clone() *Client {
	var c1 = new(Client)
	c1.agents = c.agents
	c1.alive = c.alive
	c1.conn = c.conn
	return c1
}

// removeAgent removes an agent by an agentID.
func (c *Client) removeAgent(agentID []byte) {
	c.agents.Delete(string(agentID)) // sync.Map handles internal locking
}

// newAgent creates a new agent with a unique short ID.
// Optimization: Uses atomic increment and verifies ID availability in sync.Map.
func (c *Client) newAgent() *Agent {
	var agentID string
	idBuf := make([]byte, 4)

	for i := 0; i < 0xFFFF0000; i++ {
		// Atomically increment the last ID.
		newID := atomic.AddUint32(&c.agentLastId, 1)

		if newID > 0xFFFF0000 {
			atomic.StoreUint32(&c.agentLastId, 1)
			newID = 1
		}

		binary.BigEndian.PutUint32(idBuf, newID)
		agentID = string(idBuf)

		// Confirm the ID is not currently in use.
		if _, ok := c.agents.Load(agentID); !ok {
			break
		}
	}

	agent := NewAgent(c.conn, []byte(agentID))
	c.agents.Store(agentID, agent)

	return agent
}

func (c *Client) sendCommandAndReceive(cmd protocol.Command, data []byte) (protocol.Command, []byte, error) {
	agent := c.newAgent()
	defer c.removeAgent(agent.ID)
	if err := agent.Send(cmd, data); err != nil {
		return 0, nil, err
	}
	return agent.Receive()
}

func (c *Client) sendCommand(cmd protocol.Command, data []byte) {
	agent := c.newAgent()
	defer c.removeAgent(agent.ID)
	agent.Send(cmd, data)
}

// receiveLoop listens for incoming data and dispatches to agents.
func (c *Client) receiveLoop() {
	for c.alive.Load() { // Atomic check for alive state
		payload, err := c.conn.Receive()
		if err != nil {
			if c.alive.Load() {
				log.Printf("Receive error: %v\n", err)
				c.Close()
			}
			break
		}
		agentID, cmd, data := protocol.ParseCommand(payload)
		idStr := string(agentID)

		if cmd == protocol.JOBASSIGN {
			if c.processTask != nil {
				c.processTask(idStr, data)
			}
			continue
		}

		// Use Load from sync.Map for thread-safe access.
		if val, ok := c.agents.Load(idStr); ok {
			agent := val.(*Agent)
			agent.FeedCommand(cmd, data)
		} else {
			log.Printf("Agent: %s not found.\n", agentID)
		}
	}
}

// checkHealth checks connection health.
func (c *Client) checkHealth() {
	for c.alive.Load() {
		c.Ping()
		time.Sleep(time.Second)
	}
}

// Connect to a periodic server.
func (c *Client) Connect(addr string, args ...protocol.RSAConnParam) error {
	parts := strings.SplitN(addr, "://", 2)
	conn, err := net.Dial(parts[0], parts[1])
	if err != nil {
		return err
	}
	if len(args) > 0 && len(args[0].PrivateKeyPath) > 0 {
		rsaConn, err := protocol.NewClientRSAConn(conn, args[0])
		if err != nil {
			return err
		}
		c.initClient(rsaConn, protocol.TYPECLIENT)
	} else {
		c.initClient(conn, protocol.TYPECLIENT)
	}
	go c.receiveLoop()
	go c.checkHealth()
	return nil
}

// Ping a periodic server.
func (c *Client) Ping() bool {
	ret, _, err := c.sendCommandAndReceive(protocol.PING, nil)
	return err == nil && ret == protocol.PONG
}

// SubmitJob to periodic server.
func (c *Client) SubmitJob(funcName, name string, opts map[string]interface{}) error {
	job := types.Job{Func: funcName, Name: name}
	if args, ok := opts["args"].(string); ok { job.Args = args }
	if schedat, ok := opts["schedat"].(int64); ok { job.SchedAt = schedat }
	if timeout, ok := opts["timeout"].(int32); ok { job.Timeout = timeout }

	ret, data, err := c.sendCommandAndReceive(protocol.SUBMITJOB, job.Bytes())
	if err != nil { return err }
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("SubmitJob error: %s", data)
}

// RunJob to periodic server and get a result.
func (c *Client) RunJob(funcName, name string, opts map[string]interface{}) (err error, ret []byte) {
	job := types.Job{Func: funcName, Name: name}
	if args, ok := opts["args"].(string); ok { job.Args = args }
	if timeout, ok := opts["timeout"].(int32); ok {
		job.Timeout = timeout
	} else {
		job.Timeout = 10
	}
	cmd, ret, err := c.sendCommandAndReceive(protocol.RUNJOB, job.Bytes())
	if err != nil { return err, nil }
	if cmd == protocol.NO_WORKER {
		err = fmt.Errorf("Error: no worker %s", funcName)
	}
	return err, ret
}

// RecvData from periodic server.
func (c *Client) RecvData(funcName, name string, cb func(data []byte) error) error {
	job := types.Job{Func: funcName, Name: name}
	agent := c.newAgent()
	defer c.removeAgent(agent.ID)
	agent.Send(protocol.RECVDATA, job.Bytes())
	for {
		ret, data, err := agent.Receive()
		if err != nil { return err }
		if ret == protocol.NO_WORKER {
			return fmt.Errorf("Error: no worker %s", funcName)
		}
		if len(data) == 3 && string(data) == "EOF" {
			return nil
		}
		if err := cb(data); err != nil {
			return err
		}
	}
}

// Status returns status from periodic server.
func (c *Client) Status() ([][]string, error) {
	_, data, err := c.sendCommandAndReceive(protocol.STATUS, nil)
	if err != nil { return nil, err }
	stats := strings.Split(string(data), "\n")
	sort.Strings(stats)

	lines := make([][]string, 0)
	for _, stat := range stats {
		if stat == "" { continue }
		line := strings.Split(stat, ",")
		lines = append(lines, line)
	}
	return lines, nil
}

// DropFunc drops unused function from periodic server.
func (c *Client) DropFunc(funcName string) error {
	ret, data, err := c.sendCommandAndReceive(protocol.DROPFUNC, encode8(funcName))
	if err != nil { return err }
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Drop func %s error: %s", funcName, data)
}

// RemoveJob from periodic server.
func (c *Client) RemoveJob(funcName, name string) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(funcName)))
	buf.WriteString(funcName)
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	ret, data, err := c.sendCommandAndReceive(protocol.REMOVEJOB, buf.Bytes())
	if err != nil { return err }
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("RemoveJob error: %s", data)
}

// Close the base client.
func (c *Client) Close() {
	if !c.alive.Swap(false) { // Only close if it was alive
		return
	}

	// Range over sync.Map to notify all agents.
	c.agents.Range(func(key, value interface{}) bool {
		agent := value.(*Agent)
		agent.FeedError(io.EOF) // Ensure agents aren't blocked on Receive
		return true
	})
}
