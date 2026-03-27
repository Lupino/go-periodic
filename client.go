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
	agents *sync.Map
	conn   protocol.Conn
	connMu *sync.RWMutex
	// Use atomic.Bool for lock-free state checks.
	alive          *atomic.Bool
	closed         *atomic.Bool
	reconnecting   *atomic.Bool
	agentLastID    *uint32
	processTask    func(string, []byte)
	afterReconnect func()
	clientType     protocol.ClientType
	connectNetwork string
	connectAddress string
	connectRSA     *protocol.RSAConnParam
}

// NewClient creates a new client.
func NewClient() *Client {
	c := new(Client)
	c.ensureState()
	return c
}

func (c *Client) ensureState() {
	if c.agents == nil {
		c.agents = &sync.Map{}
	}
	if c.alive == nil {
		c.alive = &atomic.Bool{}
	}
	if c.closed == nil {
		c.closed = &atomic.Bool{}
	}
	if c.reconnecting == nil {
		c.reconnecting = &atomic.Bool{}
	}
	if c.agentLastID == nil {
		c.agentLastID = new(uint32)
	}
	if c.connMu == nil {
		c.connMu = &sync.RWMutex{}
	}
	if c.clientType == 0 {
		c.clientType = protocol.TYPECLIENT
	}
}

// initClient initializes the base client.
func (c *Client) initClient(conn net.Conn, clientType protocol.ClientType) {
	c.ensureState()
	c.agents = &sync.Map{} // Re-initialize the sync.Map
	c.alive.Store(true)    // Set alive state atomically
	atomic.StoreUint32(c.agentLastID, 0)
	clientConn := protocol.NewClientConn(conn)
	clientConn.Send(clientType.Bytes())
	clientConn.Receive()
	c.setConn(clientConn)
	c.closed.Store(false)
}

// Clone clones the base client.
// Note: It continues to share the same underlying connection and agents.
func (c *Client) Clone() *Client {
	c.ensureState()
	var c1 = new(Client)
	c1.agents = c.agents
	c1.alive = c.alive
	c1.closed = c.closed
	c1.reconnecting = c.reconnecting
	c1.agentLastID = c.agentLastID
	c1.connMu = c.connMu
	c1.conn = c.conn
	c1.processTask = c.processTask
	c1.afterReconnect = c.afterReconnect
	c1.clientType = c.clientType
	c1.connectNetwork = c.connectNetwork
	c1.connectAddress = c.connectAddress
	c1.connectRSA = c.connectRSA
	return c1
}

func (c *Client) setConn(conn protocol.Conn) {
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()
}

func (c *Client) getConn() protocol.Conn {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()
	return conn
}

func (c *Client) closeConn() {
	conn := c.getConn()
	if conn.Conn != nil {
		_ = conn.Close()
	}
}

// removeAgent removes an agent by an agentID.
func (c *Client) removeAgent(agentID []byte) {
	c.ensureState()
	c.agents.Delete(string(agentID)) // sync.Map handles internal locking
}

// newAgent creates a new agent with a unique short ID.
// Optimization: Uses atomic increment and verifies ID availability in sync.Map.
func (c *Client) newAgent() *Agent {
	c.ensureState()
	var agentID string
	idBuf := make([]byte, 4)

	for i := 0; i < 0xFFFF0000; i++ {
		// Atomically increment the last ID.
		newID := atomic.AddUint32(c.agentLastID, 1)

		if newID > 0xFFFF0000 {
			atomic.StoreUint32(c.agentLastID, 1)
			newID = 1
		}

		binary.BigEndian.PutUint32(idBuf, newID)
		agentID = string(idBuf)

		// Confirm the ID is not currently in use.
		if _, ok := c.agents.Load(agentID); !ok {
			break
		}
	}

	agent := NewAgent(c.getConn(), []byte(agentID))
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
	c.ensureState()
	for c.alive.Load() { // Atomic check for alive state
		conn := c.getConn()
		payload, err := conn.Receive()
		if err != nil {
			if c.alive.Load() && !c.closed.Load() {
				log.Printf("Receive error: %v, try reconnect\n", err)
				c.tryReconnect(err)
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
	c.ensureState()
	for c.alive.Load() {
		if !c.Ping() && !c.closed.Load() {
			c.tryReconnect(io.EOF)
		}
		time.Sleep(time.Second)
	}
}

func (c *Client) parseConnect(addr string, args ...protocol.RSAConnParam) error {
	parts := strings.SplitN(addr, "://", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid address %q, expected format network://address", addr)
	}
	c.connectNetwork = parts[0]
	c.connectAddress = parts[1]
	c.connectRSA = nil
	if len(args) > 0 {
		arg := args[0]
		c.connectRSA = &arg
	}
	return nil
}

func (c *Client) dialAndInit() error {
	conn, err := net.Dial(c.connectNetwork, c.connectAddress)
	if err != nil {
		return err
	}
	if c.connectRSA != nil && len(c.connectRSA.PrivateKeyPath) > 0 {
		rsaConn, err := protocol.NewClientRSAConn(conn, *c.connectRSA)
		if err != nil {
			_ = conn.Close()
			return err
		}
		c.initClient(rsaConn, c.clientType)
	} else {
		c.initClient(conn, c.clientType)
	}
	return nil
}

func (c *Client) failAgents(err error) {
	c.agents.Range(func(key, value interface{}) bool {
		agent := value.(*Agent)
		agent.FeedError(err)
		return true
	})
}

func (c *Client) tryReconnect(cause error) {
	c.ensureState()
	if c.closed.Load() || !c.alive.Load() {
		return
	}
	if c.connectNetwork == "" || c.connectAddress == "" {
		return
	}
	if c.reconnecting.Swap(true) {
		return
	}
	defer c.reconnecting.Store(false)

	c.failAgents(cause)
	c.closeConn()

	wait := time.Second
	for c.alive.Load() && !c.closed.Load() {
		if err := c.dialAndInit(); err == nil {
			go c.receiveLoop()
			if c.afterReconnect != nil {
				c.afterReconnect()
			}
			return
		}
		time.Sleep(wait)
		if wait < 10*time.Second {
			wait *= 2
		}
	}
}

// Connect to a periodic server.
func (c *Client) Connect(addr string, args ...protocol.RSAConnParam) error {
	c.ensureState()
	if err := c.parseConnect(addr, args...); err != nil {
		return err
	}
	if err := c.dialAndInit(); err != nil {
		return err
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
	if args, ok := opts["args"].(string); ok {
		job.Args = args
	}
	if schedat, ok := opts["schedat"].(int64); ok {
		job.SchedAt = schedat
	}
	if timeout, ok := opts["timeout"].(int32); ok {
		job.Timeout = timeout
	}

	ret, data, err := c.sendCommandAndReceive(protocol.SUBMITJOB, job.Bytes())
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("SubmitJob error: %s", data)
}

// RunJob to periodic server and get a result.
func (c *Client) RunJob(funcName, name string, opts map[string]interface{}) (err error, ret []byte) {
	job := types.Job{Func: funcName, Name: name}
	if args, ok := opts["args"].(string); ok {
		job.Args = args
	}
	if timeout, ok := opts["timeout"].(int32); ok {
		job.Timeout = timeout
	} else {
		job.Timeout = 10
	}
	cmd, ret, err := c.sendCommandAndReceive(protocol.RUNJOB, job.Bytes())
	if err != nil {
		return err, nil
	}
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
		if err != nil {
			return err
		}
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
	if err != nil {
		return nil, err
	}
	stats := strings.Split(string(data), "\n")
	sort.Strings(stats)

	lines := make([][]string, 0)
	for _, stat := range stats {
		if stat == "" {
			continue
		}
		line := strings.Split(stat, ",")
		lines = append(lines, line)
	}
	return lines, nil
}

// DropFunc drops unused function from periodic server.
func (c *Client) DropFunc(funcName string) error {
	ret, data, err := c.sendCommandAndReceive(protocol.DROPFUNC, encode8(funcName))
	if err != nil {
		return err
	}
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
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("RemoveJob error: %s", data)
}

// Close the base client.
func (c *Client) Close() {
	c.ensureState()
	c.closed.Store(true)
	if !c.alive.Swap(false) { // Only close if it was alive
		return
	}
	c.closeConn()

	// Range over sync.Map to notify all agents.
	c.agents.Range(func(key, value interface{}) bool {
		agent := value.(*Agent)
		agent.FeedError(io.EOF) // Ensure agents aren't blocked on Receive
		return true
	})
}
