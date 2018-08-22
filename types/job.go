package types

import (
	"bytes"
	"encoding/binary"
)

// Job workload.
type Job struct {
	Name    string // The job name, this is unique.
	Func    string // The job function reffer on worker function
	Args    string // Job args
	SchedAt int64  // When to sched the job.
	Counter int32  // The job run counter
	Timeout int32  // The job run timeout
}

// NewJob create a job from json bytes
func NewJob(payload []byte) (job Job, err error) {
	var h byte
	h = payload[0]
	payload = payload[1:]
	job.Func = string(payload[0:h])
	payload = payload[h:]

	h = payload[0]
	payload = payload[1:]
	job.Name = string(payload[0:h])
	payload = payload[h:]

	h32 := payload[0:4]
	payload = payload[4:]
	length := binary.BigEndian.Uint32(h32)
	if length > 0 {
		job.Args = string(payload[0:length])
		payload = payload[length:]
	}

	h64 := payload[0:8]
	payload = payload[8:]
	job.SchedAt = int64(binary.BigEndian.Uint64(h64))

	var ver byte

	ver = payload[0]

	payload = payload[1:]

	if ver == 1 {
		h32 = payload[0:4]
		job.Counter = int32(binary.BigEndian.Uint32(h32))
	} else if ver == 2 {
		h32 = payload[0:4]
		job.Timeout = int32(binary.BigEndian.Uint32(h32))
	} else if ver == 3 {
		h32 = payload[0:4]
		job.Counter = int32(binary.BigEndian.Uint32(h32))
		h32 = payload[4:8]
		job.Timeout = int32(binary.BigEndian.Uint32(h32))
	}

	return
}

// Bytes encode job to json bytes
func (job Job) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(job.Func)))
	buf.WriteString(job.Func)
	buf.WriteByte(byte(len(job.Name)))
	buf.WriteString(job.Name)

	h32 := make([]byte, 4)
	binary.BigEndian.PutUint32(h32, uint32(len(job.Args)))
	buf.Write(h32)
	buf.WriteString(job.Args)

	h64 := make([]byte, 8)
	binary.BigEndian.PutUint64(h64, uint64(job.SchedAt))
	buf.Write(h64)

	var ver = 0
	if job.Counter > 0 && job.Timeout > 0 {
		ver = 3
	} else if job.Timeout > 0 {
		ver = 2
	} else if job.Counter > 0 {
		ver = 1
	}

	buf.WriteByte(byte(ver))

	if ver == 1 {
		binary.BigEndian.PutUint32(h32, uint32(job.Counter))
		buf.Write(h32)
	} else if ver == 2 {
		binary.BigEndian.PutUint32(h32, uint32(job.Timeout))
		buf.Write(h32)
	} else if ver == 3 {
		binary.BigEndian.PutUint32(h32, uint32(job.Counter))
		buf.Write(h32)
		binary.BigEndian.PutUint32(h32, uint32(job.Timeout))
		buf.Write(h32)
	}

	return buf.Bytes()
}
