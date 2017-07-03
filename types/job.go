package types

import (
	"bytes"
	"fmt"
	"github.com/Lupino/go-periodic/protocol"
	"strconv"
)

// Job workload.
type Job struct {
	ID      int64
	Name    string // The job name, this is unique.
	Func    string // The job function reffer on worker function
	Args    string // Job args
	SchedAt int64  // When to sched the job.
	Counter int64  // The job run counter
}

// NewJob create a job from json bytes
func NewJob(payload []byte) (job Job, err error) {
	parts := bytes.SplitN(payload, protocol.NullChar, 5)
	partSize := len(parts)
	if partSize < 2 {
		err = fmt.Errorf("InvalID %v\n", payload)
		return
	}
	job.Func = string(parts[0])
	job.Name = string(parts[1])
	if partSize > 2 {
		job.SchedAt, _ = strconv.ParseInt(string(parts[2]), 10, 0)
	}
	if partSize > 3 {
		job.Counter, _ = strconv.ParseInt(string(parts[3]), 10, 0)
	}
	if partSize > 4 {
		job.Args = string(parts[4])
	}
	return
}

// Bytes encode job to json bytes
func (job Job) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteString(job.Func)
	buf.Write(protocol.NullChar)
	buf.WriteString(job.Name)
	argc := len(job.Args)
	if argc > 0 || job.SchedAt > 0 || job.Counter > 0 {
		buf.Write(protocol.NullChar)
		buf.WriteString(strconv.FormatInt(job.SchedAt, 10))
	}
	if argc > 0 || job.Counter > 0 {
		buf.Write(protocol.NullChar)
		buf.WriteString(strconv.FormatInt(job.Counter, 10))
	}
	if argc > 0 {
		buf.Write(protocol.NullChar)
		buf.WriteString(job.Args)
	}
	return buf.Bytes()
}
