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
