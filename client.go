package periodic

import (
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
	"io/ioutil"
	"net"
	"sort"
	"strconv"
	"strings"
)

// Client defined a client.
type Client struct {
	bc *BaseClient
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
			c.bc = NewBaseClient(protocol.NewXORConn(conn, keyBuf), protocol.TYPECLIENT)
		}
	} else {
		c.bc = NewBaseClient(conn, protocol.TYPECLIENT)
	}
	go c.bc.ReceiveLoop()
	return nil
}

// Ping a periodic server.
func (c *Client) Ping() bool {
	agent := c.bc.NewAgent()
	defer c.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.PING, nil)
	ret, _, _ := agent.Receive()
	if ret == protocol.PONG {
		return true
	}
	return false
}

// SubmitJob to periodic server.
//  opts = map[string]string{
//    "schedat": schedat,
//    "args": args,
//  }
func (c *Client) SubmitJob(funcName, name string, opts map[string]string) error {
	agent := c.bc.NewAgent()
	defer c.bc.RemoveAgent(agent.ID)
	job := types.Job{
		Func: funcName,
		Name: name,
	}
	if args, ok := opts["args"]; ok {
		job.Args = args
	}
	if schedat, ok := opts["schedat"]; ok {
		i64, _ := strconv.ParseInt(schedat, 10, 64)
		job.SchedAt = i64
	}
	agent.Send(protocol.SUBMITJOB, job.Bytes())
	ret, data, _ := agent.Receive()
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("SubmitJob error: %s", data)
}

// RunJob to periodic server and get an result.
//  opts = map[string]string{
//    "args": args,
//  }
func (c *Client) RunJob(funcName, name string, opts map[string]string) (err error, ret []byte) {
	agent := c.bc.NewAgent()
	defer c.bc.RemoveAgent(agent.ID)
	job := types.Job{
		Func: funcName,
		Name: name,
	}
	if args, ok := opts["args"]; ok {
		job.Args = args
	}
	agent.Send(protocol.RUNJOB, job.Bytes())
	_, ret, err = agent.Receive()
	return
}

// Status return a status from periodic server.
func (c *Client) Status() ([][]string, error) {
	agent := c.bc.NewAgent()
	defer c.bc.RemoveAgent(agent.ID)
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
	agent := c.bc.NewAgent()
	defer c.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.DROPFUNC, encode8(Func))
	ret, data, _ := agent.Receive()
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Drop func %s error: %s", Func, data)
}

// RemoveJob to periodic server.
func (c *Client) RemoveJob(funcName, name string) error {
	agent := c.bc.NewAgent()
	defer c.bc.RemoveAgent(agent.ID)
	job := types.Job{
		Func: funcName,
		Name: name,
	}
	agent.Send(protocol.REMOVEJOB, job.Bytes())
	ret, data, _ := agent.Receive()
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("RemoveJob error: %s", data)
}

// Close the client.
func (c *Client) Close() {
	c.bc.Close()
}
