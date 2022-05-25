package periodic

import (
	"bytes"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
	"io/ioutil"
	"net"
	"sort"
	"strings"
)

// Client defined a client.
type Client struct {
	BaseClient
}

// NewClient create a client.
func NewClient() *Client {
	return new(Client)
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
			c.Init(protocol.NewXORConn(conn, keyBuf), protocol.TYPECLIENT)
		}
	} else {
		c.Init(conn, protocol.TYPECLIENT)
	}
	go c.ReceiveLoop()
	return nil
}

// Ping a periodic server.
func (c *Client) Ping() bool {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
	agent.Send(protocol.PING, nil)
	ret, _, _ := agent.Receive()
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
func (c *Client) SubmitJob(funcName, name string, opts map[string]interface{}) error {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
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
	agent.Send(protocol.SUBMITJOB, job.Bytes())
	ret, data, _ := agent.Receive()
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
func (c *Client) RunJob(funcName, name string, opts map[string]interface{}) (err error, ret []byte) {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
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
	agent.Send(protocol.RUNJOB, job.Bytes())
	cmd, ret, err := agent.Receive()
	if cmd == protocol.NO_WORKER {
		err = fmt.Errorf("Error: no worker %s", funcName)
	}
	return
}

// Status return a status from periodic server.
func (c *Client) Status() ([][]string, error) {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
	agent.Send(protocol.STATUS, nil)

	_, data, _ := agent.Receive()
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
func (c *Client) DropFunc(Func string) error {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
	agent.Send(protocol.DROPFUNC, encode8(Func))
	ret, data, _ := agent.Receive()
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Drop func %s error: %s", Func, data)
}

// RemoveJob to periodic server.
func (c *Client) RemoveJob(funcName, name string) error {
	agent := c.NewAgent()
	defer c.RemoveAgent(agent.ID)
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(funcName)))
	buf.WriteString(funcName)
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	agent.Send(protocol.REMOVEJOB, buf.Bytes())
	ret, data, _ := agent.Receive()
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("RemoveJob error: %s", data)
}
