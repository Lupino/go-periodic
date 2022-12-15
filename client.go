package periodic

import (
	"bytes"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
	"github.com/teris-io/shortid"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// Client defined base client.
type Client struct {
	agents      map[string]*Agent
	conn        protocol.Conn
	locker      *sync.RWMutex
	alive       bool
	processTask func(string, []byte)
}

// NewClient create a client.
func NewClient() *Client {
	return new(Client)
}

// initClient init the base client.
func (c *Client) initClient(conn net.Conn, clientType protocol.ClientType) {
	c.agents = make(map[string]*Agent)
	c.alive = true
	c.locker = new(sync.RWMutex)
	c.conn = protocol.NewClientConn(conn)
	c.conn.Send(clientType.Bytes())
	c.conn.Receive()
}

// Clone clone the base client.
func (c *Client) Clone() *Client {
	var c1 = new(Client)
	c1.agents = c.agents
	c1.alive = c.alive
	c1.locker = c.locker
	c1.conn = c.conn
	return c1
}

// removeAgent remove a agent by a agentID
func (c *Client) removeAgent(agentID []byte) {
	c.locker.Lock()
	defer c.locker.Unlock()
	delete(c.agents, string(agentID))
}

// newAgent create a new agent with an shortid
func (c *Client) newAgent() *Agent {
	c.locker.Lock()
	defer c.locker.Unlock()
	agentID, err := shortid.Generate()
	if err != nil {
		log.Fatal(err)
	}
	agentID = agentID[:4]
	agent := NewAgent(c.conn, []byte(agentID))
	c.agents[agentID] = agent
	return agent
}

func (c *Client) sendCommandAndReceive(cmd protocol.Command, data []byte) (protocol.Command, []byte, error) {
	agent := c.newAgent()
	defer c.removeAgent(agent.ID)
	agent.Send(cmd, data)
	return agent.Receive()
}

func (c *Client) sendCommand(cmd protocol.Command, data []byte) {
	agent := c.newAgent()
	defer c.removeAgent(agent.ID)
	agent.Send(cmd, data)
}

// receiveLoop a loop on receive data.
func (c *Client) receiveLoop() {
	for c.alive {
		payload, err := c.conn.Receive()
		if err != nil {
			log.Fatal(err)
		}
		agentID, cmd, data := protocol.ParseCommand(payload)
		if cmd == protocol.JOBASSIGN {
			c.processTask(string(agentID), data)
			continue
		}
		c.locker.Lock()
		agent, ok := c.agents[string(agentID)]
		if !ok {
			log.Printf("Agent: %s not found.\n", agentID)
			c.locker.Unlock()
			continue
		}
		agent.FeedCommand(cmd, data)
		c.locker.Unlock()
	}
}

// checkHealth check connection health.
func (c *Client) checkHealth() {
	for c.alive {
		c.Ping()
		time.Sleep(time.Second)
	}
}

// Connect a periodic server.
func (c *Client) Connect(addr string, key ...string) error {
	parts := strings.SplitN(addr, "://", 2)
	conn, err := net.Dial(parts[0], parts[1])
	if err != nil {
		return err
	}
	if len(key) > 0 && len(key[0]) > 0 {
		if keyBuf, err := ioutil.ReadFile(key[0]); err != nil {
			return err
		} else {
			c.initClient(protocol.NewXORConn(conn, keyBuf), protocol.TYPECLIENT)
		}
	} else {
		c.initClient(conn, protocol.TYPECLIENT)
	}
	go c.receiveLoop()
	go c.checkHealth()
	return nil
}

// Ping a periodic server.
func (c *Client) Ping() bool {
	ret, _, _ := c.sendCommandAndReceive(protocol.PING, nil)
	if ret == protocol.PONG {
		return true
	}
	return false
}

// SubmitJob to periodic server.
//
//	opts = map[string]interface{}{
//	  "schedat": schedat,
//	  "args": args,
//	  "timeout": timeout,
//	}
func (c *Client) SubmitJob(funcName, name string, opts map[string]interface{}) error {
	job := types.Job{
		Func: funcName,
		Name: name,
	}
	if args, ok := opts["args"]; ok {
		job.Args, _ = args.(string)
	}
	if schedat, ok := opts["schedat"]; ok {
		job.SchedAt, _ = schedat.(int64)
	}
	if timeout, ok := opts["timeout"]; ok {
		job.Timeout, _ = timeout.(int32)
	}
	ret, data, _ := c.sendCommandAndReceive(protocol.SUBMITJOB, job.Bytes())
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("SubmitJob error: %s", data)
}

// RunJob to periodic server and get an result.
//
//	opts = map[string]interface{}{
//	  "args": args,
//	  "timeout": timeout,
//	}
func (c *Client) RunJob(funcName, name string, opts map[string]interface{}) (err error, ret []byte) {
	job := types.Job{
		Func: funcName,
		Name: name,
	}
	if args, ok := opts["args"]; ok {
		job.Args, _ = args.(string)
	}
	if timeout, ok := opts["timeout"]; ok {
		if job.Timeout, ok = timeout.(int32); !ok {
			job.Timeout = 10
		}
	}
	cmd, ret, err := c.sendCommandAndReceive(protocol.RUNJOB, job.Bytes())
	if cmd == protocol.NO_WORKER {
		err = fmt.Errorf("Error: no worker %s", funcName)
	}
	return err, ret
}

// Status return a status from periodic server.
func (c *Client) Status() ([][]string, error) {
	_, data, _ := c.sendCommandAndReceive(protocol.STATUS, nil)
	stats := strings.Split(string(data), "\n")
	sort.Strings(stats)

	lines := make([][]string, 0, 5)
	for _, stat := range stats {
		if stat == "" {
			continue
		}
		line := strings.Split(stat, ",")
		lines = append(lines, line)
	}
	return lines, nil
}

// DropFunc drop unuself function from periodic server.
func (c *Client) DropFunc(funcName string) error {
	ret, data, _ := c.sendCommandAndReceive(protocol.DROPFUNC, encode8(funcName))
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Drop func %s error: %s", funcName, data)
}

// RemoveJob to periodic server.
func (c *Client) RemoveJob(funcName, name string) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(funcName)))
	buf.WriteString(funcName)
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	ret, data, _ := c.sendCommandAndReceive(protocol.REMOVEJOB, buf.Bytes())
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("RemoveJob error: %s", data)
}

// Close the base client.
func (c *Client) Close() {
	c.locker.Lock()
	defer c.locker.Unlock()
	for _, agent := range c.agents {
		agent.FeedError(io.EOF)
	}
	c.alive = false
}
