package periodic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
)

// Job defined a job type.
type Job struct {
	Worker   *Worker
	Raw      types.Job
	FuncName string
	Name     string
	Args     string
	Handle   []byte
}

// NewJob create a job
func NewJob(bc *Worker, data []byte) (job Job, err error) {
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
		Worker:   bc,
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
	buf := bytes.NewBuffer(nil)
	buf.Write(j.Handle)
	if len(data) == 1 {
		buf.Write(data[0])
	}
	ret, vv, _ := j.Worker.sendCommandAndReceive(protocol.WORKDONE, buf.Bytes())
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Done error: %s", vv)
}

// Fail tell periodic server the job fail.
func (j *Job) Fail() error {
	ret, data, _ := j.Worker.sendCommandAndReceive(protocol.WORKFAIL, j.Handle)
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Fail error: %s", data)
}

// SchedLater tell periodic server to sched job later on delay.
// SchedLater(delay int)
// SchedLater(delay, counter int) sched with a incr the counter
func (j *Job) SchedLater(opts ...int) error {
	delay := opts[0]
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
	ret, data, _ := j.Worker.sendCommandAndReceive(protocol.SCHEDLATER, buf.Bytes())
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("SchedLater error: %s", data)
}

// Acquire acquire the lock from periodic server
func (j *Job) Acquire(name string, count int) (error, bool) {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)

	h16 := make([]byte, 2)
	binary.BigEndian.PutUint16(h16, uint16(count))
	buf.Write(h16)
	buf.Write(j.Handle)

	ret, data, _ := j.Worker.sendCommandAndReceive(protocol.ACQUIRE, buf.Bytes())

	if ret == protocol.ACQUIRED && data[0] == 1 {
		return nil, true
	}
	return nil, false
}

// Release release lock
func (j *Job) Release(name string) error {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(name)))
	buf.WriteString(name)
	buf.Write(j.Handle)

	ret, data, _ := j.Worker.sendCommandAndReceive(protocol.RELEASE, buf.Bytes())
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Release error: %s", data)
}

// WithLock with lock
func (j *Job) WithLock(name string, count int, task func()) {
	_, acquired := j.Acquire(name, count)
	if acquired {
		task()
		j.Release(name)
	}
}
