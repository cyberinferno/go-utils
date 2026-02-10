// Package eventdriventcpclient provides an event-driven TCP client that notifies
// callers of connection state changes, received data, and errors via registered
// handlers. It supports optional auto-reconnect and configurable timeouts.
package eventdriventcpclient

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// ConnectionState represents the current state of the TCP connection.
type ConnectionState int

const (
	Disconnected ConnectionState = iota // Not connected and not attempting to connect
	Connecting                          // Connection attempt in progress
	Connected                           // Successfully connected
	Reconnecting                        // Disconnected and attempting to reconnect (when AutoReconnect is enabled)
	Closed                              // Client has been closed and will not reconnect
)

// String returns a human-readable name for the connection state.
func (cs ConnectionState) String() string {
	switch cs {
	case Disconnected:
		return "Disconnected"
	case Connecting:
		return "Connecting"
	case Connected:
		return "Connected"
	case Reconnecting:
		return "Reconnecting"
	case Closed:
		return "Closed"
	default:
		return "Unknown"
	}
}

// ConnectionStateEvent is emitted when the connection state changes.
// It is passed to the handler registered with OnConnectionState.
type ConnectionStateEvent struct {
	State     ConnectionState // The new connection state
	Address   string          // The remote address (e.g. "host:port")
	Timestamp time.Time       // When the state change occurred
	Error     error           // Non-nil if the state change was due to an error
}

// DataReceivedEvent is emitted when data is read from the connection.
// It is passed to the handler registered with OnDataReceived.
type DataReceivedEvent struct {
	Data      []byte    // The received bytes (do not modify; copy if needed)
	Length    int       // Length of Data (same as len(Data))
	Timestamp time.Time // When the data was received
}

// ErrorEvent is emitted when a read, write, or connection error occurs.
// It is passed to the handler registered with OnError.
type ErrorEvent struct {
	Error     error     // The error that occurred
	Timestamp time.Time // When the error occurred
}

// ConnectionStateHandler is called when the connection state changes.
// Handlers are invoked from goroutines; implementations must be safe for concurrent use.
type ConnectionStateHandler func(event ConnectionStateEvent)

// DataReceivedHandler is called when data is received from the connection.
// Handlers are invoked from goroutines; implementations must be safe for concurrent use.
type DataReceivedHandler func(event DataReceivedEvent)

// ErrorHandler is called when a read, write, or connection error occurs.
// Handlers are invoked from goroutines; implementations must be safe for concurrent use.
type ErrorHandler func(event ErrorEvent)

// Config holds configuration for the event-driven TCP client.
type Config struct {
	// Address is the "host:port" to connect to (e.g. "localhost:8080").
	Address string
	// AutoReconnect enables automatic reconnection when the connection is lost.
	AutoReconnect bool
	// ReconnectInterval is the delay between reconnection attempts when AutoReconnect is true.
	ReconnectInterval time.Duration
	// ReadBufferSize is the size of the read buffer when DataLengthBasedRead is false.
	ReadBufferSize int
	// WriteTimeout is the max duration for a single write; 0 means no timeout.
	WriteTimeout time.Duration
	// ReadTimeout is the max duration to wait for read data; 0 means no timeout.
	ReadTimeout time.Duration
	// ConnectionTimeout is the max duration for establishing a new connection.
	ConnectionTimeout time.Duration
	// DataLengthBasedRead, when true, reads a 4-byte little-endian length prefix
	// and then that many bytes per message instead of streaming into fixed-size chunks.
	DataLengthBasedRead bool
}

// DefaultEventDrivenTCPClientConfig returns a Config with default values for the given address.
// AutoReconnect is false; override fields as needed before passing to NewEventDrivenTCPClient.
//
// Parameters:
//   - address: The "host:port" to connect to
//
// Returns:
//   - A Config with defaults: ReconnectInterval 5s, ReadBufferSize 4096,
//     WriteTimeout 10s, ConnectionTimeout 10s, ReadTimeout 0, DataLengthBasedRead false.
func DefaultEventDrivenTCPClientConfig(address string) Config {
	return Config{
		Address:             address,
		AutoReconnect:       false,
		ReconnectInterval:   5 * time.Second,
		ReadBufferSize:      4096,
		WriteTimeout:        10 * time.Second,
		ReadTimeout:         0,
		ConnectionTimeout:   10 * time.Second,
		DataLengthBasedRead: false,
	}
}

// EventDrivenTCPClient is a TCP client that drives I/O and connection lifecycle
// via events. Register handlers with OnConnectionState, OnDataReceived, and OnError,
// then call Connect to start. It is safe for concurrent use.
type EventDrivenTCPClient struct {
	config Config
	conn   net.Conn
	state  ConnectionState

	onConnectionState ConnectionStateHandler
	onDataReceived    DataReceivedHandler
	onError           ErrorHandler

	mu            sync.RWMutex
	stopChan      chan struct{}
	reconnectChan chan struct{}
	wg            sync.WaitGroup
	closed        bool
	reconnecting  bool
}

// NewEventDrivenTCPClient creates a new event-driven TCP client with the given config.
// The client starts in Disconnected state; call Connect to establish a connection.
//
// Parameters:
//   - config: Connection and behavior settings (e.g. from DefaultEventDrivenTCPClientConfig)
//
// Returns:
//   - A new *EventDrivenTCPClient ready to use; call Close when done to release resources.
func NewEventDrivenTCPClient(config Config) *EventDrivenTCPClient {
	return &EventDrivenTCPClient{
		config:        config,
		state:         Disconnected,
		stopChan:      make(chan struct{}),
		reconnectChan: make(chan struct{}, 1),
	}
}

// OnConnectionState registers the handler for connection state changes.
// Only one handler is active; repeated calls replace the previous handler.
// Pass nil to clear the handler.
//
// Parameters:
//   - handler: Function called on state changes (Connecting, Connected, Disconnected, etc.)
func (c *EventDrivenTCPClient) OnConnectionState(handler ConnectionStateHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onConnectionState = handler
}

// OnDataReceived registers the handler for incoming data.
// Only one handler is active; repeated calls replace the previous handler.
// Pass nil to clear the handler.
//
// Parameters:
//   - handler: Function called with each chunk or message of received data
func (c *EventDrivenTCPClient) OnDataReceived(handler DataReceivedHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onDataReceived = handler
}

// OnError registers the handler for read, write, and connection errors.
// Only one handler is active; repeated calls replace the previous handler.
// Pass nil to clear the handler.
//
// Parameters:
//   - handler: Function called when an error occurs
func (c *EventDrivenTCPClient) OnError(handler ErrorHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onError = handler
}

// Connect establishes a TCP connection to the configured address.
// It returns an error if the client is closed, already connected/connecting, or if the dial fails.
// When AutoReconnect is enabled, a read goroutine and reconnect goroutine are started.
//
// Returns:
//   - nil on success; otherwise an error (e.g. "client is closed", "already connected or connecting", or dial error).
func (c *EventDrivenTCPClient) Connect() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("client is closed")
	}
	if c.state == Connected || c.state == Connecting {
		c.mu.Unlock()
		return fmt.Errorf("already connected or connecting")
	}
	c.mu.Unlock()

	return c.connect()
}

// Disconnect closes the current connection and moves to Disconnected state.
// It does not set the client to Closed; Connect may be called again.
// Safe to call when already disconnected or closed; returns nil in those cases.
//
// Returns:
//   - nil if already disconnected/closed, or the error from closing the connection.
func (c *EventDrivenTCPClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == Disconnected || c.state == Closed {
		return nil
	}

	return c.disconnect()
}

func (c *EventDrivenTCPClient) disconnect() error {
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.setState(Disconnected, nil)
		return err
	}

	return nil
}

// Close shuts down the client, closes the connection, and stops all goroutines.
// After Close, the client is in Closed state and must not be used further.
// Idempotent; calling Close multiple times is safe and returns nil.
//
// Returns:
//   - nil
func (c *EventDrivenTCPClient) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}

	c.closed = true

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
	c.mu.Unlock()

	close(c.stopChan)
	c.wg.Wait()

	c.setState(Closed, nil)

	return nil
}

// Send writes data to the connection. It returns an error if not connected or if the write fails.
// When WriteTimeout is set in config, each write is limited to that duration.
// On write error, the error handler is invoked and reconnect may be triggered if AutoReconnect is enabled.
//
// Parameters:
//   - data: Bytes to send; not modified
//
// Returns:
//   - nil on success; an error if not connected, connection is nil, or the write fails.
func (c *EventDrivenTCPClient) Send(data []byte) error {
	c.mu.RLock()
	conn := c.conn
	state := c.state
	c.mu.RUnlock()

	if state != Connected {
		return fmt.Errorf("not connected")
	}

	if conn == nil {
		return fmt.Errorf("connection is nil")
	}

	if c.config.WriteTimeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout)); err != nil {
			return err
		}

		defer func() {
			_ = conn.SetWriteDeadline(time.Time{}) // Best effort to clear deadline
		}()
	}

	_, err := conn.Write(data)
	if err != nil {
		c.emitError(err)
		c.triggerReconnect()
	}

	return err
}

// GetState returns the current connection state.
//
// Returns:
//   - The current ConnectionState (Disconnected, Connecting, Connected, Reconnecting, or Closed).
func (c *EventDrivenTCPClient) GetState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// IsConnected returns true if the client is in Connected state.
func (c *EventDrivenTCPClient) IsConnected() bool {
	return c.GetState() == Connected
}

func (c *EventDrivenTCPClient) connect() error {
	c.setState(Connecting, nil)

	dialer := net.Dialer{
		Timeout: c.config.ConnectionTimeout,
	}

	conn, err := dialer.Dial("tcp", c.config.Address)
	if err != nil {
		c.setState(Disconnected, err)
		c.emitError(err)
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	c.setState(Connected, nil)

	c.wg.Add(1)
	go c.readLoop()

	if c.config.AutoReconnect {
		c.wg.Add(1)
		go c.reconnectHandler()
	}

	return nil
}

func (c *EventDrivenTCPClient) readLoop() {
	defer c.wg.Done()

	if c.config.DataLengthBasedRead {
		for {
			c.mu.RLock()
			conn := c.conn
			closed := c.closed
			c.mu.RUnlock()

			if conn == nil || closed {
				return
			}

			if c.config.ReadTimeout > 0 {
				if err := conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout)); err != nil {
					if !c.isClosed() {
						c.emitError(err)
						c.triggerReconnect()
					}
					break
				}
			} else {
				if err := conn.SetReadDeadline(time.Time{}); err != nil {
					if !c.isClosed() {
						c.emitError(err)
						c.triggerReconnect()
					}
					break
				}
			}

			var buf bytes.Buffer
			if _, err := io.CopyN(&buf, conn, 4); err != nil {
				if !c.isClosed() {
					c.emitError(err)
					c.triggerReconnect()
				}

				break
			}

			if c.isClosed() {
				return
			}

			reader := io.MultiReader(&buf, conn)
			dataLength := binary.LittleEndian.Uint32(buf.Bytes())
			if dataLength == 0 {
				continue
			}

			if dataLength > 16*1024*1024 {
				break
			}

			packet := make([]byte, dataLength)
			if _, err := io.ReadFull(reader, packet); err != nil {
				if !c.isClosed() {
					c.emitError(err)
					c.triggerReconnect()
				}

				break
			}

			if c.isClosed() {
				return
			}

			c.emitDataReceived(packet)
		}

		return
	}

	buffer := make([]byte, c.config.ReadBufferSize)
	for {
		c.mu.RLock()
		conn := c.conn
		closed := c.closed
		c.mu.RUnlock()

		if conn == nil || closed {
			return
		}

		if c.config.ReadTimeout > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(c.config.ReadTimeout)); err != nil {
				if !c.isClosed() {
					c.emitError(err)
					c.triggerReconnect()
				}
				return
			}
		} else {
			if err := conn.SetReadDeadline(time.Time{}); err != nil {
				if !c.isClosed() {
					c.emitError(err)
					c.triggerReconnect()
				}
				return
			}
		}

		n, err := conn.Read(buffer)

		if c.isClosed() {
			return
		}

		if err != nil {
			if !c.isClosed() {
				c.emitError(err)
				c.triggerReconnect()
			}

			return
		}

		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			c.emitDataReceived(data)
		}
	}
}

func (c *EventDrivenTCPClient) reconnectHandler() {
	defer c.wg.Done()

	for {
		select {
		case <-c.stopChan:
			return
		case <-c.reconnectChan:
			c.mu.Lock()
			if c.reconnecting {
				c.mu.Unlock()
				continue
			}
			c.reconnecting = true
			c.mu.Unlock()

			c.mu.Lock()
			if err := c.disconnect(); err != nil {
				c.emitError(err)
			}

			c.mu.Unlock()

			c.setState(Reconnecting, nil)

			select {
			case <-c.stopChan:
				c.mu.Lock()
				c.reconnecting = false
				c.mu.Unlock()
				return
			case <-time.After(c.config.ReconnectInterval):
			}

			if c.isClosed() {
				c.mu.Lock()
				c.reconnecting = false
				c.mu.Unlock()
				return
			}

			err := c.connect()

			c.mu.Lock()
			c.reconnecting = false
			c.mu.Unlock()

			if err != nil {
				select {
				case c.reconnectChan <- struct{}{}:
				default:
				}
			}
		}
	}
}

func (c *EventDrivenTCPClient) triggerReconnect() {
	if !c.config.AutoReconnect || c.isClosed() {
		return
	}

	select {
	case c.reconnectChan <- struct{}{}:
	default:
	}
}

func (c *EventDrivenTCPClient) setState(state ConnectionState, err error) {
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()

	c.emitConnectionState(state, err)
}

func (c *EventDrivenTCPClient) emitConnectionState(state ConnectionState, err error) {
	c.mu.RLock()
	handler := c.onConnectionState
	c.mu.RUnlock()

	if handler != nil {
		event := ConnectionStateEvent{
			State:     state,
			Address:   c.config.Address,
			Timestamp: time.Now(),
			Error:     err,
		}

		go handler(event)
	}
}

func (c *EventDrivenTCPClient) emitDataReceived(data []byte) {
	c.mu.RLock()
	handler := c.onDataReceived
	c.mu.RUnlock()

	if handler != nil {
		event := DataReceivedEvent{
			Data:      data,
			Length:    len(data),
			Timestamp: time.Now(),
		}

		go handler(event)
	}
}

func (c *EventDrivenTCPClient) emitError(err error) {
	c.mu.RLock()
	handler := c.onError
	c.mu.RUnlock()

	if handler != nil {
		event := ErrorEvent{
			Error:     err,
			Timestamp: time.Now(),
		}

		go handler(event)
	}
}

func (c *EventDrivenTCPClient) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}
