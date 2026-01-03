package protocol

import (
	"strconv"
)

// Command defined command type.
type Command int

const (
	// NOOP do nothing
	NOOP Command = iota // server (0)
	// GRABJOB client ask a job
	GRABJOB // worker (1)
	// SCHEDLATER tell server sched later the job
	SCHEDLATER // worker (2)
	// WORKDONE tell server the work is done
	WORKDONE // worker (3)
	// WORKFAIL tell server work is fail
	WORKFAIL // worker (4)
	// JOBASSIGN assign a job for client
	JOBASSIGN // server (5)
	// NOJOB tell client job is empty
	NOJOB // server (6)
	// CANDO tell server the worker can do some func
	CANDO // worker (7)
	// CANTDO tell server the worker can not do some func
	CANTDO // worker (8)
	// PING test ping
	PING // client or worker (9)
	// PONG reply pong
	PONG // server (10)
	// SLEEP tell the worker to sleep
	SLEEP // worker (11)
	// UNKNOWN command unknow
	UNKNOWN // server (12)
	// SUBMITJOB submit a job for server
	SUBMITJOB // client (13)
	// STATUS ask the server status
	STATUS // client (14)
	// DROPFUNC drop an empty worker func
	DROPFUNC // client (15)
	// SUCCESS reply client success
	SUCCESS // server (16)
	// REMOVEJOB remove a job
	REMOVEJOB // client (17)
	// DUMP dump the data
	DUMP // client (18)
	// LOAD load data to database
	LOAD // client (19)
	// SHUTDOWN shutdown the server
	SHUTDOWN // client (20)
	// BROADCAST broadcast all the worker
	BROADCAST // worker (21)
	// CONFIGGET get the server config
	CONFIGGET // client (22)
	// CONFIGSET set the server config
	CONFIGSET // client (23)
	// CONFIG return config to client
	CONFIG // server (24)
	// RUNJOB run job and got a result
	RUNJOB // client (25)
	// ACQUIRED acquire true or false
	ACQUIRED // (26)
	// ACQUIRE acquire the lock
	ACQUIRE // (27)
	// RELEASE release the lock
	RELEASE // (28)
	// NO_WORKER on run job when no worker return this
	NO_WORKER // server (29)
	// DATA run job data
	DATA // server (30)
	// RECVDATA receive data
	RECVDATA // client (31)
	// WORKDATA worker data
	WORKDATA // client (32)
	// JOBASSIGNED job already assigned
	JOBASSIGNED // worker (33)
)

// Bytes convert command to byte slice
func (c Command) Bytes() []byte {
	return []byte{byte(c)}
}

// String convert command to string for logging and debugging
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
	case LOAD:
		return "LOAD"
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
	case RECVDATA:
		return "RECVDATA"
	case WORKDATA:
		return "WORKDATA"
	case JOBASSIGNED:
		return "JOBASSIGNED"
	default:
		return "UNKNOWN_COMMAND_" + strconv.Itoa(int(c))
	}
}

// FromByte convert byte to Command with basic validation
func FromByte(b byte) Command {
	cmd := Command(b)
	if cmd > JOBASSIGNED || cmd < NOOP {
		return UNKNOWN
	}
	return cmd
}
