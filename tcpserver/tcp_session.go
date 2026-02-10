package tcpserver

// TCPServerSession is the interface that must be implemented by each connection
// session. The server creates a session per connection and runs Handle in a
// goroutine; the session is responsible for reading, processing, and optionally
// sending data until Close is called.
type TCPServerSession interface {
	// ID returns the session's unique identifier assigned by the server.
	//
	// Returns:
	//   - The session ID (uint32)
	ID() uint32

	// Handle runs the session's main loop (e.g., read loop). It is typically
	// started in a goroutine by the server and runs until the connection is
	// closed or the session decides to exit.
	Handle()

	// Close closes the session and releases resources. It should be safe to call
	// multiple times.
	//
	// Returns:
	//   - An error if closing failed
	Close() error

	// Send writes data to the connection. Implementations should be safe for
	// concurrent use if multiple goroutines may call Send.
	//
	// Parameters:
	//   - data: The bytes to send
	//
	// Returns:
	//   - An error if the write failed
	Send(data []byte) error
}
