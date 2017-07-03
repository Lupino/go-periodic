package periodic

import (
	"bytes"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
	"strconv"
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
	parts := bytes.SplitN(data, protocol.NullChar, 2)
	raw, err = types.NewJob(parts[1])
	if err != nil {
		return
	}
	job = Job{
		bc:       bc,
		Raw:      raw,
		FuncName: raw.Func,
		Name:     raw.Name,
		Args:     raw.Args,
		Handle:   parts[0],
	}
	return
}

// Done tell periodic server the job done.
func (j *Job) Done() error {
	agent := j.bc.NewAgent()
	defer j.bc.RemoveAgent(agent.ID)
	agent.Send(protocol.WORKDONE, j.Handle)
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
	buf.Write(protocol.NullChar)
	buf.WriteString(strconv.Itoa(delay))
	if len(opts) == 2 {
		buf.Write(protocol.NullChar)
		buf.WriteString(strconv.Itoa(opts[1]))
	}
	agent.Send(protocol.SCHEDLATER, buf.Bytes())
	return nil
}
