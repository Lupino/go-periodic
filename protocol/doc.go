/*
Package protocol defined periodic protocol

The Periodic protocol operates over TCP on port 5000 by default,
or unix socket on unix:///tmp/periodic.sock.
Communication happens between either a client and periodic server,
or between a worker and periodic server.
In either case, the protocol consists of packets
containing requests and responses. All packets sent to a periodic server
are considered requests, and all packets sent from a periodic server are
considered responses. A simple configuration may look like:

	----------     ----------     ----------     ----------
	| Client |     | Client |     | Client |     | Client |
	----------     ----------     ----------     ----------
	     \             /              \             /
	      \           /                \           /
	   -------------------          -------------------
	   | Periodic Server |          | Periodic Server |
	   -------------------          -------------------
	            |                            |
	    ----------------------------------------------
	    |              |              |              |
	----------     ----------     ----------     ----------
	| Worker |     | Worker |     | Worker |     | Worker |
	----------     ----------     ----------     ----------

Initially, the workers register functions they can perform with each
job server. Clients will then connect to a job server and issue a
request to a job to be run. The job server then notifies each worker
that can perform that job (based on the function it registered) that
a new job is ready. The first worker to wake up and retrieve the job
will then execute it.
All communication between workers or clients and the periodic server
are binary.

## Client type

Once connected the client send it's type, and server will respond the connction id.
Request:

	4 byte magic code   - This is either "\0REQ" for requests or "\0RES"
	                      for responses.
	4 byte size         - A big-endian (network-order) integer containing
	                      the size of the data being sent.
	4 byte crc32        - A big-endian (network-order) integer containing
	                      the crc32 of the data being sent.
	1 byte command      - A big-endian (network-order) integer containing
	                      an enumerated packet command. Possible values are:
	                    #   Name             Type
	                    1   TYPE_CLIENT      Client
	                    2   TYPE_WORKER      Worker

Response:

	4 byte magic code       - This is either "\0REQ" for requests or "\0RES"
	                          for responses.
	4 byte size             - A big-endian (network-order) integer containing
	                          the size of the data being sent.
	4 byte crc32        - A big-endian (network-order) integer containing
	                      the crc32 of the data being sent.
	? byte connection id    - A binary connection id

## Binary Packet

Requests and responses are encapsulated by a binary packet. A binary
packet consists of a header which is optionally followed by data. The
header is:

	4 byte magic code   - This is either "\0REQ" for requests or "\0RES"
	                      for responses.
	4 byte size         - A big-endian (network-order) integer containing
	                      the size of the data being sent.
	4 byte crc32        - A big-endian (network-order) integer containing
	                      the crc32 of the data being sent.
	4 byte message id   - A client unique message id.
	1 byte command      - A big-endian (network-order) integer containing
	                      an enumerated packet command. Possible values are:
	                    #   Name          Type
	                    0   NOOP          Client/Worker
	                    1   GRAB_JOB      Worker
	                    2   SCHED_LATER   Worker
	                    3   WORK_DONE     Worker
	                    4   WORK_FAIL     Worker
	                    5   JOB_ASSIGN    Worker
	                    6   NO_JOB        Worker
	                    7   CAN_DO        Worker
	                    8   CANT_DO       Worker
	                    9   PING          Client/Worker
	                    10  PONG          Client/Worker
	                    11  SLEEP         Worker
	                    12  UNKNOWN       Client/Worker
	                    13  SUBMIT_JOB    Client
	                    14  STATUS        Client
	                    15  DROP_FUNC     Client
	                    16  SUCCESS       Client/Worker
	                    17  REMOVE_JOB    Client
	                    18  DUMP          Client
	                    19  LOAD          Client
	                    20  SHUTDOWN      Client
	                    21  BROADCAST     Worker
	                    22  CONFIG_GET    Client
	                    23  CONFIG_SET    Client
	                    24  CONFIG        Client
	                    25  RUN_JOB       Client
	                    26  ACQUIRED      Worker
	                    27  ACQUIRE       Worker
	                    28  RELEASE       Worker
	                    29  NO_WORKER     Client
	                    30  DATA          Client

## Client/Worker Requests
These request types may be sent by either a client or a worker:

	PING
	    When a periodic server receives this request, it simply generates a
	    PONG packet. This is primarily used for testing
	    or debugging.
	    Arguments:
	    - None.

## Client/Worker Responses
These response types may be sent to either a client or a worker:

	PONG
	    This is sent in response to a PING request.
	    Arguments:
	    - None.

## Client Requests
These request types may only be sent by a client:

	 SUBMIT_JOB
	     A client issues one of these when a job needs to be run. The
	     server will respond with a SUCCESS packet.
	     Arguments:
	     - Job binary packet
	RUN_JOB
	     A client issues one of these when a job needs to be run. The
	     server will respond with a DATA packet or NO_WORKER packet.
	     Arguments:
	     - Job binary packet
	 STATUS
	     This sends back a list of all registered functions.  Next to
	     each function is the number of jobs in the queue, the number of
	     running jobs, and the number of capable workers. The format is:
	     FUNCTIONS  WORKERS  JOBS  PROCESSING  LOCKED  SCHEDAT
	     Arguments:
	     - None.
	 DROP_FUNC
	     Drop the function when there is not worker registered, and respond with
	     a SUCCESS packet.
	     Arguments:
	     - 1 byte func size
	     - ? byte func name
	 REMOVE_JOB
	     Remove a job, and respond with a SUCCESS packet.
	     Arguments:
	     - 1 byte func size
	     - ? byte func name
	     - 1 byte name size
	     - ? byte name
	 DUMP
	     Dump data from server. The server will respond with a binary packet.
	     Arguments:
	     - None.
	 LOAD
	     Load data to server. The server will respond with a SUCCESS packet.
	     Arguments:
	     - None.
	 CONFIG_GET
	     Get config from server. The server will respond with a CONFIG packet.
	     Arguments:
	     - 1 byte key size.
	     - ? byte key. the key is one of
	         - poll-interval
	         - revert-interval
	         - timeout
	         - keepalive
	         - max-batch-size
	 CONFIG_GET
	     Get config from server. The server will respond with a CONFIG packet.
	     Arguments:
	     - 1 byte key size.
	     - ? byte key. the key is one of
	         - poll-interval
	         - revert-interval
	         - timeout
	         - keepalive
	         - max-batch-size
	     - 4 byte value (int32).
	 SHUTDOWN
	     Shutdown server.
	     Arguments:
	     - None.

## Client Responses
These response types may only be sent to a client:

	SUCCESS
	    This is sent in response to one of the SUBMIT_JOB/DROP_FUNC/REMOVE_JOB/CONFIG_SET
	    packets. It signifies to the client that a the server successfully
	    received the job and queued it to be run by a worker.
	    Arguments:
	    - None.
	CONFIG
	    This is sent in response to one of the CONFIG_GET packets.
	    Arguments:
	    - 4 byte value (int32).
	DATA
	    This is sent in response to one of the STATUS/RUN_JOB/DUMP packets.
	    Arguments:
	    - 4 byte value (int32).
	    - ? byte binary
	NO_WORKER
	    This is sent in response to one of the RUN_JOB packets.
	    Arguments:
	    - None

## Worker Requests
These request types may only be sent by a worker:

	BROADCAST
	    This is sent to notify the server that the worker is able to
	    perform the given function. The worker is then put on a list to be
	    woken up whenever the job server receives a job for that function.
	    The server will respond with a SUCCESS packet.
	    Arguments:
	    - 1 byte func size
	    - ? byte func name
	CAN_DO
	    This is sent to notify the server that the worker is able to
	    perform the given function. The worker is then put on a list to be
	    woken up whenever the job server receives a job for that function.
	    The server will respond with a SUCCESS packet.
	    Arguments:
	    - 1 byte func size
	    - ? byte func name
	CANT_DO
	    This is sent to notify the server that the worker is no longer
	    able to perform the given function.
	    The server will respond with a SUCCESS packet.
	    Arguments:
	    - 1 byte func size
	    - ? byte func name
	SLEEP
	    This is sent to notify the server that the worker is about to
	    sleep, and that it should be woken up with a NOOP packet if a
	    job comes in for a function the worker is able to perform.
	    Arguments:
	    - None.
	GRAB_JOB
	    This is sent to the server to request any available jobs on the
	    queue. The server will respond with either NO_JOB or JOB_ASSIGN,
	    depending on whether a job is available.
	    Arguments:
	    - None.
	WORK_DONE
	    This is to notify the server that the job completed successfully.
	    The server will respond with a SUCCESS packet.
	    Arguments:
	    - ? byte handle
	    - ? byte data
	WORK_FAIL
	    This is to notify the server that the job failed.
	    The server will respond with a SUCCESS packet.
	    Arguments:
	    - ? byte handle
	SCHED_LATER
	    This is to notify the server to do the job on next time.
	    The server will respond with a SUCCESS packet.
	    Arguments:
	    - ? byte handle
	    - 8 byte time delay
	    - 2 byte step counter
	ACQUIRE
	    This is to acquire a lock from the server.
	    The server will respond with a ACQUIRED packet.
	    Arguments:
	    - 1 byte lock name size
	    - ? byte lock name
	    - 2 byte lock count
	    - ? byte handle
	RELEASE
	    This is to release a lock to the server.
	    The server will respond with a SUCCESS packet.
	    Arguments:
	    - 1 byte lock name size
	    - ? byte lock name
	    - ? byte handle

## Worker Responses
These response types may only be sent to a worker:

	NOOP
	    This is used to wake up a sleeping worker so that it may grab a
	    pending job.
	    Arguments:
	    - None.
	NO_JOB
	    This is given in response to a GRAB_JOB request to notify the
	    worker there are no pending jobs that need to run.
	    Arguments:
	    - None.
	JOB_ASSIGN
	    This is given in response to a GRAB_JOB request to give the worker
	    information needed to run the job. All communication about the
	    job (such as status updates and completion response) should use
	    the handle, and the worker should run the given function with
	    the argument.
	    Arguments:
	    - Job binary packet
	ACQUIRED
	    This is to response to a ACQUIRE request. if true run the job, else skip.
	    Arguments:
	    - 1 byte 1 or 0

## Job binary packets
Job is encode with a binary packet:

  - 1 byte func size

  - ? byte func name

  - 1 byte name size

  - ? byte name

  - 4 byte workload size

  - ? byte workload

  - 8 byte sched time (unix time, int64)

  - 1 byte job version

  - #  version

  - 0  Ver0

  - 1  Ver1

  - 2  Ver2

  - 3  Ver3

    Version Spec:
    Ver0:

  - None
    Ver1:

  - 4 byte run count           this assign by worker
    Ver2:

  - 4 byte timeout
    Ver3:

  - 4 byte run count           this assign by worker

  - 4 byte timeout

## Job handle packets
Job handle is encode with a binary packet:
  - 1 byte func size
  - ? byte func name
  - 1 byte name size
  - ? byte name
*/
package protocol
