package types

import "testing"

func TestNewJobRejectsMalformedPayload(t *testing.T) {
	if _, err := NewJob([]byte{}); err == nil {
		t.Fatal("expected error for empty payload")
	}

	// funcLen=1 but no func bytes.
	if _, err := NewJob([]byte{1}); err == nil {
		t.Fatal("expected error for truncated func")
	}

	// Valid func/name, but args length says 10 and only 1 byte present.
	payload := []byte{
		1, 'f',
		1, 'n',
		0, 0, 0, 10, 'x',
	}
	if _, err := NewJob(payload); err == nil {
		t.Fatal("expected error for truncated args")
	}
}

func TestNewJobRoundTrip(t *testing.T) {
	src := Job{
		Name:    "n",
		Func:    "f",
		Args:    "a",
		SchedAt: 1234,
		Counter: 2,
		Timeout: 3,
	}
	got, err := NewJob(src.Bytes())
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if got.Name != src.Name || got.Func != src.Func || got.Args != src.Args || got.SchedAt != src.SchedAt || got.Counter != src.Counter || got.Timeout != src.Timeout {
		t.Fatalf("round-trip mismatch: got=%+v want=%+v", got, src)
	}
}
