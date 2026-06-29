The periodic task system client for go.

Install
-------

    go get -v github.com/Lupino/go-preiodic

Usage
-----

worker
```go
import "github.com/Lupino/go-periodic"

var periodicServer = "unix:///tmp/periodic.sock"
var worker = periodic.NewWorker()
worker.Connect(periodicServer)

func handle(job periodic.Job) {
    job.Done()
    // job.Fail()
    // job.SchedLater(3)
}

worker.AddFunc("funcName", handle)

worker.Work()

```
client

```go
import "github.com/Lupino/go-periodic"

var periodicServer = "unix:///tmp/periodic.sock"
var client = periodic.NewClient()
client.Connect(periodicServer)
client.SubmitJob(...)
```

Authenticated clients
---------------------

When `periodicd` runs with an auth file, configure the identity before
connecting:

```go
var periodicServer = "unix:///tmp/periodic.sock"

var client = periodic.NewClient()
client.SetAuth("client-a", "token-a")
client.Connect(periodicServer)

var worker = periodic.NewWorker(1)
worker.SetAuth("client-a", "token-a")
worker.Connect(periodicServer)
worker.AddFunc("func1", handle)
```

Example server auth file line:

```text
client client-a token-a func1,func2
worker worker-a token-worker-a func1,func2
```

example see [here](https://github.com/Lupino/periodic/tree/master/cmd/periodic/subcmd)
