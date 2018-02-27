package protocol

// ParseCommand payload to extract msgID cmd and data
func ParseCommand(payload []byte) (msgID []byte, cmd Command, data []byte) {
	msgID = payload[0:4]
	cmd = UNKNOWN
	if len(payload) > 4 {
		cmd = Command(payload[4])
	}
	if len(payload) > 5 {
		data = payload[5:]
	}
	return
}
