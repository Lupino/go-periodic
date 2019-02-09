package periodic

import (
	"bytes"
	"encoding/binary"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
)

// Job defined a job type.
type Job struct {
	bc       *BaseClient
	Raw      types.Job
	FuncName string
	Name     string
	Args     string
	Handle   []byte
}

// NewJob create a job
func NewJob(bc *BaseClient, data []byte) (job Job, err error) {
	var raw types.Job
	raw, err = types.NewJob(data)
	if err != nil {
		return
	}

	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(raw.Func)))
	buf.WriteString(raw.Func)
	buf.WriteByte(byte(len(raw.Name)))
	buf.WriteString(raw.Name)

	job = Job{
		bc:       bc,
		Raw:      raw,
		FuncName: raw.Func,
		Name:     raw.Name,
		Args:     raw.Args,
		Handle:   buf.Bytes(),
	}
	return
}

// Done tell periodic server the job done.
func (j *Job) Done(data ...[]byte) error {
	agent := j.bc.NewAgent()
	defer j.bc.RemoveAgent(agent.ID)
	var dat = []byte("")
	if len(data) == 1 {
		dat = data[0]
	}
	agent.Send(protocol.WORKDONE, bytes.Join([][]byte{j.Handle, dat}, []byte("")))
	return nil
}

// Fail tell periodic server the job fail.
func (j *Job) Fail() error {
	agent := j.bc.NewAgent()
	defer j.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.WORKFAIL, j.Handle)
	return nil
}

// SchedLater tell periodic server to sched job later on delay.
// SchedLater(delay int)
// SchedLater(delay, counter int) sched with a incr the counter
func (j *Job) SchedLater(opts ...int) error {
	delay := opts[0]
	agent := j.bc.NewAgent()
	defer j.bc.RemoveAgent(agent.ID)
	buf := bytes.NewBuffer(nil)
	buf.Write(j.Handle)
	h64 := make([]byte, 8)
	binary.BigEndian.PutUint64(h64, uint64(delay))
	buf.Write(h64)

	h16 := make([]byte, 2)
	if len(opts) == 2 {
		binary.BigEndian.PutUint16(h16, uint16(opts[1]))
	} else {
		binary.BigEndian.PutUint16(h16, uint16(0))
	}
	buf.Write(h16)
	agent.Send(protocol.SCHEDLATER, buf.Bytes())
	return nil
}

// Acquire acquire the lock from periodic server
func (j *Job) Acquire(name string, count int) (error, bool) {
	agent := j.bc.NewAgent()
	defer j.bc.RemoveAgent(agent.ID)

	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)

	h16 := make([]byte, 2)
	binary.BigEndian.PutUint16(h16, uint16(count))
	buf.Write(h16)
	buf.Write(j.Handle)

	agent.Send(protocol.ACQUIRE, buf.Bytes())

	ret, data, _ := agent.Receive()
	if ret == protocol.ACQUIRED && data[0] == 1 {
		return nil, true
	}
	return nil, false
}

// Release release lock
func (j *Job) Release(name string) error {
	agent := j.bc.NewAgent()
	defer j.bc.RemoveAgent(agent.ID)

	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	buf.Write(j.Handle)

	agent.Send(protocol.RELEASE, buf.Bytes())
	return nil
}
