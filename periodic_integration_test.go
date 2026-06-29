package periodic

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Lupino/go-periodic/protocol"
)

func integrationConfig(t *testing.T) (string, []protocol.RSAConnParam) {
	t.Helper()
	if os.Getenv("PERIODIC_INTEGRATION") != "1" {
		t.Skip("set PERIODIC_INTEGRATION=1 to run periodic integration tests")
	}

	addr := os.Getenv("PERIODIC_ADDR")
	if addr == "" {
		addr = "unix:///tmp/periodic.sock"
	}

	modeStr := os.Getenv("PERIODIC_RSA_MODE")
	if modeStr == "" {
		return addr, nil
	}

	mode, err := strconv.Atoi(modeStr)
	if err != nil {
		t.Fatalf("invalid PERIODIC_RSA_MODE: %v", err)
	}

	param := protocol.RSAConnParam{
		PrivateKeyPath:      getenvDefault("PERIODIC_CLIENT_PRIVATE_KEY", "private_key.pem"),
		ServerPublicKeyPath: getenvDefault("PERIODIC_SERVER_PUBLIC_KEY", "server_public_key.pem"),
		Mode:                mode,
	}
	return addr, []protocol.RSAConnParam{param}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustConnectClient(t *testing.T, addr string, args ...protocol.RSAConnParam) *Client {
	t.Helper()
	c := NewClient()
	setAuthFromEnv(t, c, "PERIODIC_CLIENT_NAME", "PERIODIC_CLIENT_TOKEN")
	if err := c.Connect(addr, args...); err != nil {
		t.Fatalf("connect client: %v", err)
	}
	return c
}

func mustConnectWorker(t *testing.T, size int, addr string, args ...protocol.RSAConnParam) *Worker {
	t.Helper()
	w := NewWorker(size)
	setAuthFromEnv(t, &w.Client, "PERIODIC_WORKER_NAME", "PERIODIC_WORKER_TOKEN")
	if err := w.Connect(addr, args...); err != nil {
		t.Fatalf("connect worker: %v", err)
	}
	return w
}

func setAuthFromEnv(t *testing.T, c *Client, nameKey, tokenKey string) {
	t.Helper()
	name := os.Getenv(nameKey)
	token := os.Getenv(tokenKey)
	if name == "" && token == "" {
		return
	}
	if err := c.SetAuth(name, token); err != nil {
		t.Fatalf("set auth from %s/%s: %v", nameKey, tokenKey, err)
	}
}

func uniqueName(prefix string) string {
	if os.Getenv("PERIODIC_STATIC_FUNC_NAMES") == "1" {
		return prefix
	}
	return prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

func TestPeriodicPingWorker(t *testing.T) {
	addr, args := integrationConfig(t)
	w := mustConnectWorker(t, 1, addr, args...)
	defer w.Close()

	if ok := w.Ping(); !ok {
		t.Fatal("worker ping failed")
	}
}

func TestPeriodicPingClient(t *testing.T) {
	addr, args := integrationConfig(t)
	c := mustConnectClient(t, addr, args...)
	defer c.Close()

	if ok := c.Ping(); !ok {
		t.Fatal("client ping failed")
	}
}

func TestPeriodicSubmitJob(t *testing.T) {
	addr, args := integrationConfig(t)
	c := mustConnectClient(t, addr, args...)
	defer c.Close()

	funcName := uniqueName("test_submit")
	jobName := uniqueName("job")

	err := c.SubmitJob(funcName, jobName, map[string]interface{}{
		"schedat": time.Now().Unix(),
	})
	if err != nil {
		t.Fatalf("submit job: %v", err)
	}

	_ = c.RemoveJob(funcName, jobName)
	_ = c.DropFunc(funcName)
}

func TestPeriodicRunJobNoWorker(t *testing.T) {
	addr, args := integrationConfig(t)
	c := mustConnectClient(t, addr, args...)
	defer c.Close()

	funcName := uniqueName("test_no_worker")
	err, ret := c.RunJob(funcName, "job", map[string]interface{}{"timeout": int32(2)})
	if err == nil {
		t.Skipf("environment has a worker for %q, got result=%q", funcName, string(ret))
	}
	if !strings.Contains(strings.ToLower(err.Error()), "no worker") {
		t.Fatalf("expected no worker error, got: %v", err)
	}
}

func TestPeriodicWorkerRunJob(t *testing.T) {
	addr, args := integrationConfig(t)
	w := mustConnectWorker(t, 1, addr, args...)
	defer w.Close()

	c := mustConnectClient(t, addr, args...)
	defer c.Close()

	funcName := uniqueName("test_worker")
	jobName := "haha"
	doneCh := make(chan string, 1)

	if err := w.AddFunc(funcName, func(job Job) {
		doneCh <- job.Name
		_ = job.Done([]byte(job.Name))
	}); err != nil {
		t.Fatalf("add func: %v", err)
	}
	defer c.DropFunc(funcName)

	go w.Work()
	time.Sleep(200 * time.Millisecond)

	retErr, ret := c.RunJob(funcName, jobName, map[string]interface{}{"timeout": int32(5)})
	if retErr != nil {
		t.Fatalf("run job: %v", retErr)
	}
	if string(ret) != jobName {
		t.Fatalf("unexpected result: got=%q want=%q", string(ret), jobName)
	}

	select {
	case name := <-doneCh:
		if name != jobName {
			t.Fatalf("unexpected handled job name: got=%q want=%q", name, jobName)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not process job in time")
	}
}

func TestPeriodicWorkerReconnectAutoReaddFuncReference(t *testing.T) {
	addr, args := integrationConfig(t)
	if os.Getenv("PERIODIC_RECONNECT_TEST") != "1" {
		t.Skip("set PERIODIC_RECONNECT_TEST=1 to run reconnect reference test")
	}

	w := mustConnectWorker(t, 1, addr, args...)
	defer w.Close()
	c := mustConnectClient(t, addr, args...)
	defer c.Close()

	funcName := uniqueName("test_reconnect")
	if err := w.AddFunc(funcName, func(job Job) {
		_ = job.Done([]byte("reconnect-ok"))
	}); err != nil {
		t.Fatalf("add func: %v", err)
	}
	defer c.DropFunc(funcName)
	go w.Work()

	// Reference flow:
	// 1) restart periodic server while this test is running
	// 2) worker auto reconnects and re-registers funcs
	// 3) runJob succeeds without calling AddFunc again
	t.Log("restart periodic server now, then wait a moment for auto reconnect")
	time.Sleep(5 * time.Second)

	retErr, ret := c.RunJob(funcName, "job", map[string]interface{}{"timeout": int32(5)})
	if retErr != nil {
		t.Fatalf("run job after reconnect: %v", retErr)
	}
	if string(ret) != "reconnect-ok" {
		t.Fatalf("unexpected reconnect result: %q", string(ret))
	}
}
