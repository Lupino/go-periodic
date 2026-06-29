package periodic

import (
	"github.com/Lupino/go-periodic/protocol"
	"testing"
)

func TestClientCloseWithZeroValueDoesNotPanic(t *testing.T) {
	var c Client
	c.Close()
}

func TestClientCloneWithZeroValueDoesNotPanic(t *testing.T) {
	var c Client
	_ = c.Clone()
}

func TestNewWorkerUsesWorkerClientType(t *testing.T) {
	w := NewWorker(1)
	if w.clientType != protocol.TYPEWORKER {
		t.Fatalf("unexpected worker client type: %v", w.clientType)
	}
}

func TestNewWorkerWithZeroSizeDefaultsToOne(t *testing.T) {
	w := NewWorker(0)
	if w.wp.Size() != 1 {
		t.Fatalf("unexpected worker pool size: %d", w.wp.Size())
	}
}

func TestSetAuth(t *testing.T) {
	c := NewClient()
	if err := c.SetAuth("client-a", "token-a"); err != nil {
		t.Fatalf("SetAuth returned error: %v", err)
	}
	if c.clientAuth == nil || c.clientAuth.Name != "client-a" || c.clientAuth.Token != "token-a" {
		t.Fatalf("unexpected client auth: %#v", c.clientAuth)
	}
	if err := c.SetAuth("client-a", ""); err == nil {
		t.Fatal("SetAuth accepted partial auth")
	}
	if err := c.SetAuth("", ""); err != nil {
		t.Fatalf("SetAuth clear returned error: %v", err)
	}
	if c.clientAuth != nil {
		t.Fatalf("client auth was not cleared: %#v", c.clientAuth)
	}
}
