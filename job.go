package periodic

import (
	"encoding/binary"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"github.com/Lupino/go-periodic/types"
)

// Job defines a job type and its associated worker and raw data.
type Job struct {
	Worker   *Worker
	Raw      types.Job
	FuncName string
	Name     string
	Args     string
	Handle   []byte
}

// NewJob creates a job from raw byte data assigned by the server.
func NewJob(bc *Worker, data []byte) (job Job, err error) {
	var raw types.Job
	raw, err = types.NewJob(data)
	if err != nil {
		return
	}

	// Optimization: Pre-allocate handle buffer instead of using bytes.Buffer.
	handle := make([]byte, 1+len(raw.Func)+1+len(raw.Name))
	handle[0] = byte(len(raw.Func))
	copy(handle[1:], raw.Func)
	handle[1+len(raw.Func)] = byte(len(raw.Name))
	copy(handle[2+len(raw.Func):], raw.Name)

	job = Job{
		Worker:   bc,
		Raw:      raw,
		FuncName: raw.Func,
		Name:     raw.Name,
		Args:     raw.Args,
		Handle:   handle,
	}
	return
}

// Done notifies the periodic server that the job is complete.
func (j *Job) Done(data ...[]byte) error {
	totalSize := len(j.Handle)
	if len(data) == 1 {
		totalSize += len(data[0])
	}

	buf := make([]byte, totalSize)
	copy(buf, j.Handle)
	if len(data) == 1 {
		copy(buf[len(j.Handle):], data[0])
	}

	// Fix: Capture and check the error from the underlying connection.
	ret, vv, err := j.Worker.sendCommandAndReceive(protocol.WORKDONE, buf)
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Done error: %s", vv)
}

// Data sends intermediate job data to the periodic server.
func (j *Job) Data(data ...[]byte) error {
	totalSize := len(j.Handle)
	if len(data) == 1 {
		totalSize += len(data[0])
	}

	buf := make([]byte, totalSize)
	copy(buf, j.Handle)
	if len(data) == 1 {
		copy(buf[len(j.Handle):], data[0])
	}

	ret, vv, err := j.Worker.sendCommandAndReceive(protocol.WORKDATA, buf)
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Data error: %s", vv)
}

// Fail notifies the periodic server that the job has failed.
func (j *Job) Fail() error {
	ret, data, err := j.Worker.sendCommandAndReceive(protocol.WORKFAIL, j.Handle)
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Fail error: %s", data)
}

// SchedLater tells the periodic server to reschedule the job for later.
// SchedLater(delay int)
// SchedLater(delay, counter int) sched with a incr the counter
func (j *Job) SchedLater(opts ...int) error {
	if len(opts) < 1 {
		return fmt.Errorf("SchedLater requires at least a delay parameter")
	}

	delay := opts[0]
	handleLen := len(j.Handle)
	buf := make([]byte, handleLen+8+2) // Handle + 8 bytes (delay) + 2 bytes (counter)

	copy(buf, j.Handle)
	binary.BigEndian.PutUint64(buf[handleLen:], uint64(delay))

	if len(opts) == 2 {
		binary.BigEndian.PutUint16(buf[handleLen+8:], uint16(opts[1]))
	} else {
		binary.BigEndian.PutUint16(buf[handleLen+8:], 0)
	}

	ret, data, err := j.Worker.sendCommandAndReceive(protocol.SCHEDLATER, buf)
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("SchedLater error: %s", data)
}

// Acquire requests a lock from the periodic server.
func (j *Job) Acquire(name string, count int) (error, bool) {
	nameLen := len(name)
	buf := make([]byte, 1+nameLen+2+len(j.Handle))
	buf[0] = byte(nameLen)
	copy(buf[1:], name)
	binary.BigEndian.PutUint16(buf[1+nameLen:], uint16(count))
	copy(buf[1+nameLen+2:], j.Handle)

	ret, data, err := j.Worker.sendCommandAndReceive(protocol.ACQUIRE, buf)
	if err != nil {
		return err, false
	}

	// Safety: Check data length before accessing index.
	if ret == protocol.ACQUIRED && len(data) > 0 && data[0] == 1 {
		return nil, true
	}
	return nil, false
}

// Release releases a previously acquired lock.
func (j *Job) Release(name string) error {
	nameLen := len(name)
	buf := make([]byte, 1+nameLen+len(j.Handle))
	buf[0] = byte(nameLen)
	copy(buf[1:], name)
	copy(buf[1+nameLen:], j.Handle)

	ret, data, err := j.Worker.sendCommandAndReceive(protocol.RELEASE, buf)
	if err != nil {
		return err
	}
	if ret == protocol.SUCCESS {
		return nil
	}
	return fmt.Errorf("Release error: %s", data)
}

// WithLock executes a task while holding a named lock.
func (j *Job) WithLock(name string, count int, task func()) {
	err, acquired := j.Acquire(name, count)
	if err == nil && acquired {
		defer j.Release(name)
		task()
	}
}
