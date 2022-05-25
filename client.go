package periodic

// Client defined a client.
type Client struct {
	BaseClient
}

// NewClient create a client.
func NewClient() *Client {
	return new(Client)
}
