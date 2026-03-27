package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	need := func(n int, field string) error {
		if len(payload) < n {
			return fmt.Errorf("invalid job payload: need %d bytes for %s, got %d", n, field, len(payload))
		}
		return nil
	}

	if err := need(1, "func length"); err != nil {
		return job, err
	}
	funcLen := int(payload[0])
	payload = payload[1:]
	if err := need(funcLen, "func"); err != nil {
		return job, err
	}
	job.Func = string(payload[:funcLen])
	payload = payload[funcLen:]

	if err := need(1, "name length"); err != nil {
		return job, err
	}
	nameLen := int(payload[0])
	payload = payload[1:]
	if err := need(nameLen, "name"); err != nil {
		return job, err
	}
	job.Name = string(payload[:nameLen])
	payload = payload[nameLen:]

	if err := need(4, "args length"); err != nil {
		return job, err
	}
	argsLen := int(binary.BigEndian.Uint32(payload[:4]))
	payload = payload[4:]
	if err := need(argsLen, "args"); err != nil {
		return job, err
	}
	if argsLen > 0 {
		job.Args = string(payload[:argsLen])
		payload = payload[argsLen:]
	}

	if err := need(8, "schedAt"); err != nil {
		return job, err
	}
	job.SchedAt = int64(binary.BigEndian.Uint64(payload[:8]))
	payload = payload[8:]

	if err := need(1, "version"); err != nil {
		return job, err
	}
	ver := payload[0]
	payload = payload[1:]

	switch ver {
	case 0:
		return job, nil
	case 1:
		if err := need(4, "counter"); err != nil {
			return job, err
		}
		job.Counter = int32(binary.BigEndian.Uint32(payload[:4]))
	case 2:
		if err := need(4, "timeout"); err != nil {
			return job, err
		}
		job.Timeout = int32(binary.BigEndian.Uint32(payload[:4]))
	case 3:
		if err := need(8, "counter+timeout"); err != nil {
			return job, err
		}
		job.Counter = int32(binary.BigEndian.Uint32(payload[:4]))
		job.Timeout = int32(binary.BigEndian.Uint32(payload[4:8]))
	default:
		return job, fmt.Errorf("invalid job payload: unknown version %d", ver)
	}

	return job, nil
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
