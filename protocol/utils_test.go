package protocol

import (
	"fmt"
	"testing"
)

func testPanic() {
	x := recover()
	if x == nil {
		panic("there no panic")
	}
	fmt.Printf("panic %s\n", x)
}

func TestParseCommand(t *testing.T) {
	var pack = []byte("100\x00\x01\x01\x00\x01hhcc")
	var msgID, cmd, data = ParseCommand(pack)
	fmt.Printf("%d, %d, %s\n", msgID, cmd, data)
}

func TestParseCommandPanic1(t *testing.T) {
	var pack = []byte("100\x00\x01")
	ParseCommand(pack)
}

func TestParseCommandPanic2(t *testing.T) {
	defer testPanic()
	var pack = []byte("100")
	ParseCommand(pack)
}
