package protocol

import (
	"bytes"
	"strconv"
)

// Command defined command type.
type Command int

const (
	// NOOP do nothing
	NOOP Command = 0 // server
	// GRABJOB client ask a job
	GRABJOB Command = 1 // worker
	// SCHEDLATER tell server sched later the job
	SCHEDLATER Command = 2 // worker
	// WORKDONE tell server the work is done
	WORKDONE Command = 3 // worker
	// WORKFAIL tell server work is fail
	WORKFAIL Command = 4 // worker
	// JOBASSIGN assign a job for client
	JOBASSIGN Command = 5 // server
	// NOJOB tell client job is empty
	NOJOB Command = 6 // server
	// CANDO tell server the worker can do some func
	CANDO Command = 7 // worker
	// CANTDO tell server the worker can not do some func
	CANTDO Command = 8 // worker
	// PING test ping
	PING Command = 9 // client or worker
	// PONG reply pong
	PONG Command = 10 // server
	// SLEEP tell the worker to sleep
	SLEEP Command = 11 // worker
	// UNKNOWN command unknow
	UNKNOWN Command = 12 // server
	// SUBMITJOB submit a job for server
	SUBMITJOB Command = 13 // client
	// STATUS ask the server status
	STATUS Command = 14 // client
	// DROPFUNC drop an empty worker func
	DROPFUNC Command = 15 // client
	// SUCCESS reply client success
	SUCCESS Command = 16 // server
	// REMOVEJOB remove a job
	REMOVEJOB Command = 17 // client

	// DUMP dump the data
	DUMP Command = 18 // client
	// LOAD load data to database
	LOAD Command = 19 // client
	// SHUTDOWN shutdown the server
	SHUTDOWN Command = 20 // client
	// BROADCAST broadcast all the worker
	BROADCAST Command = 21 // worker

	// CONFIGGET get the server config
	CONFIGGET Command = 22 // client
	// CONFIGSET set the server config
	CONFIGSET Command = 23 // client
	// CONFIG return config to client
	CONFIG Command = 24 // server
	// RUNJOB run job and got a result
	RUNJOB Command = 25 // client

	// ACQUIRED acquire true or false
	ACQUIRED Command = 26
	// ACQUIRE acquire the lock
	ACQUIRE Command = 27
	// RELEASE release the lock
	RELEASE Command = 28

	// NO_WORKER on run job when no worker return this
	NO_WORKER = 29 // server
	// DATA run job data
	DATA = 30 // server
)

// Bytes convert command to byte
func (c Command) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(c))
	return buf.Bytes()
}

func (c Command) String() string {
	switch c {
	case NOOP:
		return "NOOP"
	case GRABJOB:
		return "GRABJOB"
	case SCHEDLATER:
		return "SCHEDLATER"
	case WORKDONE:
		return "WORKDONE"
	case WORKFAIL:
		return "WORKFAIL"
	case JOBASSIGN:
		return "JOBASSIGN"
	case NOJOB:
		return "NOJOB"
	case CANDO:
		return "CANDO"
	case CANTDO:
		return "CANTDO"
	case PING:
		return "PING"
	case PONG:
		return "PONG"
	case SLEEP:
		return "SLEEP"
	case UNKNOWN:
		return "UNKNOWN"
	case SUBMITJOB:
		return "SUBMITJOB"
	case STATUS:
		return "STATUS"
	case DROPFUNC:
		return "DROPFUNC"
	case SUCCESS:
		return "SUCCESS"
	case REMOVEJOB:
		return "REMOVEJOB"
	case DUMP:
		return "DUMP"
	case SHUTDOWN:
		return "SHUTDOWN"
	case BROADCAST:
		return "BROADCAST"
	case CONFIGGET:
		return "CONFIGGET"
	case CONFIGSET:
		return "CONFIGSET"
	case CONFIG:
		return "CONFIG"
	case RUNJOB:
		return "RUNJOB"
	case ACQUIRED:
		return "ACQUIRED"
	case ACQUIRE:
		return "ACQUIRE"
	case RELEASE:
		return "RELEASE"
	case NO_WORKER:
		return "NO_WORKER"
	case DATA:
		return "DATA"
	}
	panic("Unknow Command " + strconv.Itoa(int(c)))
}
