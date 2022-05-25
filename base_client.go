package periodic

import (
	"bytes"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
	"github.com/ventu-io/go-shortid"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
)

// BaseClient defined base client.
type BaseClient struct {
	agents map[string]Feeder
	conn   protocol.Conn
	locker *sync.RWMutex
	alive  bool
}

// Init init the base client.
func (c *BaseClient) Init(conn net.Conn, clientType protocol.ClientType) {
	c.agents = make(map[string]Feeder)
	c.alive = true
	c.locker = new(sync.RWMutex)
	c.conn = protocol.NewClientConn(conn)
	c.conn.Send(clientType.Bytes())
	c.conn.Receive()
}

// RemoveAgent remove a agent by a agentID
func (c *BaseClient) RemoveAgent(agentID []byte) {
	c.locker.Lock()
	defer c.locker.Unlock()
	delete(c.agents, string(agentID))
}

// NewAgent create a new agent with an shortid
func (c *BaseClient) NewAgent() *Agent {
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

func (c *BaseClient) sendCommandAndReceive(cmd protocol.Command, data []byte) (protocol.Command, []byte, error) {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
	agent.Send(cmd, data)
	return agent.Receive()
}

func (c *BaseClient) sendCommand(cmd protocol.Command, data []byte) {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
	agent.Send(cmd, data)
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
			log.Printf("Agent: %s not found.\n", agentID)
			c.locker.Unlock()
			continue
		}
		agent.FeedCommand(cmd, data)
		c.locker.Unlock()
	}
}

// Connect a periodic server.
func (c *BaseClient) Connect(addr string, key ...string) error {
	parts := strings.SplitN(addr, "://", 2)
	conn, err := net.Dial(parts[0], parts[1])
	if err != nil {
		return err
	}
	if len(key) > 0 && len(key[0]) > 0 {
		if keyBuf, err := ioutil.ReadFile(key[0]); err != nil {
			return err
		} else {
			c.Init(protocol.NewXORConn(conn, keyBuf), protocol.TYPECLIENT)
		}
	} else {
		c.Init(conn, protocol.TYPECLIENT)
	}
	go c.ReceiveLoop()
	return nil
}

// Ping a periodic server.
func (c *BaseClient) Ping() bool {
	ret, _, _ := c.sendCommandAndReceive(protocol.PING, nil)
	if ret == protocol.PONG {
		return true
	}
	return false
}

// SubmitJob to periodic server.
//  opts = map[string]interface{}{
//    "schedat": schedat,
//    "args": args,
//    "timeout": timeout,
//  }
func (c *BaseClient) SubmitJob(funcName, name string, opts map[string]interface{}) error {
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
//  opts = map[string]interface{}{
//    "args": args,
//    "timeout": timeout,
//  }
func (c *BaseClient) RunJob(funcName, name string, opts map[string]interface{}) (err error, ret []byte) {
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
func (c *BaseClient) Status() ([][]string, error) {
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
func (c *BaseClient) DropFunc(funcName string) error {
	ret, data, _ := c.sendCommandAndReceive(protocol.DROPFUNC, encode8(funcName))
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Drop func %s error: %s", funcName, data)
}

// RemoveJob to periodic server.
func (c *BaseClient) RemoveJob(funcName, name string) error {
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
func (c *BaseClient) Close() {
	c.locker.Lock()
	defer c.locker.Unlock()
	for _, agent := range c.agents {
		agent.FeedError(io.EOF)
	}
	c.alive = false
}
