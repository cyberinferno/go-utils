# Event-Driven TCP Client Documentation

The `eventdriventcpclient` package provides an event-driven TCP client that notifies callers of connection state changes, received data, and errors via registered handlers. It supports optional auto-reconnect, configurable timeouts, and two read modes: stream-based (fixed buffer) or length-prefixed messages. The client is safe for concurrent use.

## Features

- **Event-Driven**: Register handlers for connection state, received data, and errors; no blocking read loops in your code
- **Concurrent Safe**: All exported methods are safe for use from multiple goroutines
- **Optional Auto-Reconnect**: When enabled, the client automatically reconnects after connection loss with configurable interval
- **Configurable Timeouts**: Connection, read, and write timeouts; use zero for no timeout
- **Two Read Modes**: Stream reads (fixed buffer size) or length-prefixed messages (4-byte little-endian length + payload)
- **Clear Lifecycle**: Disconnected → Connecting → Connected; optional Reconnecting; Close for shutdown

## Installation

```go
import "github.com/cyberinferno/go-utils/eventdriventcpclient"
```

## Configuration

### Config

Configuration is provided via the `Config` struct. Use `DefaultEventDrivenTCPClientConfig` and override fields as needed.

| Field | Type | Description |
|-------|------|-------------|
| `Address` | `string` | The `"host:port"` to connect to (e.g. `"localhost:8080"`). |
| `AutoReconnect` | `bool` | When true, the client automatically reconnects after disconnect or read/write errors. |
| `ReconnectInterval` | `time.Duration` | Delay between reconnection attempts when AutoReconnect is true. |
| `ReadBufferSize` | `int` | Size of the read buffer when `DataLengthBasedRead` is false. |
| `WriteTimeout` | `time.Duration` | Max duration for a single write; 0 means no timeout. |
| `ReadTimeout` | `time.Duration` | Max duration to wait for read data; 0 means no timeout. |
| `ConnectionTimeout` | `time.Duration` | Max duration for establishing a new connection. |
| `DataLengthBasedRead` | `bool` | When true, each message is read as 4-byte little-endian length + that many bytes. |

### DefaultEventDrivenTCPClientConfig

Returns a `Config` with default values for the given address. AutoReconnect is false by default.

```go
import (
    "github.com/cyberinferno/go-utils/eventdriventcpclient"
    "time"
)

cfg := eventdriventcpclient.DefaultEventDrivenTCPClientConfig("localhost:8080")

// Override as needed
cfg.AutoReconnect = true
cfg.ReconnectInterval = 3 * time.Second
cfg.ReadBufferSize = 8192
cfg.WriteTimeout = 5 * time.Second
cfg.ReadTimeout = 30 * time.Second
cfg.ConnectionTimeout = 10 * time.Second
cfg.DataLengthBasedRead = true
```

**Parameters:**

- **address**: The `"host:port"` to connect to.

**Returns:**

- A `Config` with defaults: ReconnectInterval 5s, ReadBufferSize 4096, WriteTimeout 10s, ConnectionTimeout 10s, ReadTimeout 0, DataLengthBasedRead false.

---

## Creating a Client

### NewEventDrivenTCPClient

Creates a new event-driven TCP client. The client starts in `Disconnected` state; call `Connect` to establish a connection. Call `Close` when done to release resources.

```go
cfg := eventdriventcpclient.DefaultEventDrivenTCPClientConfig("localhost:9000")
client := eventdriventcpclient.NewEventDrivenTCPClient(cfg)

defer client.Close()
```

**Parameters:**

- **config**: Connection and behavior settings (e.g. from `DefaultEventDrivenTCPClientConfig`).

**Returns:**

- A new `*EventDrivenTCPClient` ready to use.

---

## Event Types and Handlers

### Connection State

`ConnectionState` is an enum: `Disconnected`, `Connecting`, `Connected`, `Reconnecting`, `Closed`. Use `ConnectionState.String()` for a human-readable name.

**ConnectionStateEvent** is passed to the connection state handler when the state changes:

```go
type ConnectionStateEvent struct {
    State     ConnectionState // The new state
    Address   string          // Remote address (e.g. "host:port")
    Timestamp time.Time       // When the change occurred
    Error     error           // Non-nil if the change was due to an error
}
```

**ConnectionStateHandler** is a function type; register with `OnConnectionState`. Handlers are invoked from goroutines and must be safe for concurrent use.

### Data Received

**DataReceivedEvent** is passed to the data handler when bytes are read from the connection:

```go
type DataReceivedEvent struct {
    Data      []byte    // Received bytes (do not modify; copy if needed)
    Length    int       // Same as len(Data)
    Timestamp time.Time // When the data was received
}
```

**DataReceivedHandler** is a function type; register with `OnDataReceived`. Handlers are invoked from goroutines.

### Errors

**ErrorEvent** is passed to the error handler when a read, write, or connection error occurs:

```go
type ErrorEvent struct {
    Error     error     // The error that occurred
    Timestamp time.Time // When it occurred
}
```

**ErrorHandler** is a function type; register with `OnError`. Handlers are invoked from goroutines.

---

## Basic Usage

### Register Handlers

Register handlers before calling `Connect`, or from another goroutine while the client is running. Only one handler per type is active; repeated calls replace the previous handler. Pass `nil` to clear a handler.

```go
client.OnConnectionState(func(ev eventdriventcpclient.ConnectionStateEvent) {
    log.Printf("state: %s @ %s (err: %v)", ev.State, ev.Address, ev.Error)
})

client.OnDataReceived(func(ev eventdriventcpclient.DataReceivedEvent) {
    log.Printf("received %d bytes", ev.Length)
    // Copy ev.Data if you need to keep it; do not modify ev.Data
})

client.OnError(func(ev eventdriventcpclient.ErrorEvent) {
    log.Printf("error: %v", ev.Error)
})
```

### Connect

Establishes a TCP connection to the configured address. Returns an error if the client is closed, already connected or connecting, or if the dial fails. When AutoReconnect is enabled, a read goroutine and reconnect goroutine are started.

```go
err := client.Connect()
if err != nil {
    log.Fatal(err)
}
```

**Returns:**

- `nil` on success; otherwise an error (e.g. "client is closed", "already connected or connecting", or dial error).

### Send

Writes data to the connection. Returns an error if not connected or if the write fails. When `WriteTimeout` is set, each write is limited to that duration. On write error, the error handler is invoked and reconnect may be triggered if AutoReconnect is enabled.

```go
data := []byte("hello\n")
err := client.Send(data)
if err != nil {
    log.Printf("send failed: %v", err)
}
```

**Parameters:**

- **data**: Bytes to send; not modified.

**Returns:**

- `nil` on success; an error if not connected or the write fails.

### GetState and IsConnected

```go
state := client.GetState()       // ConnectionState
if client.IsConnected() {
    // ready to Send
}
```

### Disconnect

Closes the current connection and moves to `Disconnected` state. Does not set the client to `Closed`; you may call `Connect` again. Safe to call when already disconnected or closed; returns nil in those cases.

```go
err := client.Disconnect()
```

### Close

Shuts down the client, closes the connection, and stops all goroutines. After `Close`, the client is in `Closed` state and must not be used further. Idempotent; calling `Close` multiple times is safe and returns nil.

```go
client.Close()
```

---

## Connection States

| State | Description |
|-------|-------------|
| `Disconnected` | Not connected and not attempting to connect. |
| `Connecting` | Connection attempt in progress. |
| `Connected` | Successfully connected; you can call `Send`. |
| `Reconnecting` | Disconnected and attempting to reconnect (when AutoReconnect is enabled). |
| `Closed` | Client has been closed and will not reconnect. |

State transitions are reported to the handler registered with `OnConnectionState`.

---

## Read Modes

### Stream Mode (DataLengthBasedRead = false)

Data is read into a buffer of size `ReadBufferSize`. Each time the buffer is filled (or the connection returns data), a `DataReceivedEvent` is emitted with up to `ReadBufferSize` bytes. Message boundaries are not preserved; you must implement framing in your handler if needed.

### Length-Prefixed Mode (DataLengthBasedRead = true)

Each message is:

1. 4 bytes: little-endian uint32 length (excluding these 4 bytes).
2. N bytes: payload of that length.

Messages larger than 16 MiB are rejected (read loop exits). Length 0 is allowed and results in no data event. This mode is useful for binary protocols where the server sends length-prefixed frames.

When sending from your side, you can use the same format:

```go
// Example: send length-prefixed message
func sendMessage(conn *eventdriventcpclient.EventDrivenTCPClient, payload []byte) error {
    buf := make([]byte, 4+len(payload))
    binary.LittleEndian.PutUint32(buf[:4], uint32(len(payload)))
    copy(buf[4:], payload)
    return conn.Send(buf)
}
```

---

## Concurrency

- **Client methods**: All exported methods (`Connect`, `Disconnect`, `Close`, `Send`, `GetState`, `IsConnected`, `OnConnectionState`, `OnDataReceived`, `OnError`) are safe for concurrent use.
- **Handlers**: Handlers are invoked from the client’s goroutines. Your handler code must be safe for concurrent use (e.g. avoid race conditions if you share state with other goroutines).
- **DataReceivedEvent.Data**: Do not modify the slice; copy it if you need to keep the data after the handler returns.

---

## Complete Examples

### Example 1: Simple Echo Client

```go
package main

import (
    "log"
    "time"

    "github.com/cyberinferno/go-utils/eventdriventcpclient"
)

func main() {
    cfg := eventdriventcpclient.DefaultEventDrivenTCPClientConfig("localhost:8080")
    client := eventdriventcpclient.NewEventDrivenTCPClient(cfg)
    defer client.Close()

    client.OnConnectionState(func(ev eventdriventcpclient.ConnectionStateEvent) {
        log.Printf("state: %s", ev.State)
    })
    client.OnDataReceived(func(ev eventdriventcpclient.DataReceivedEvent) {
        log.Printf("received %d bytes: %s", ev.Length, string(ev.Data))
    })
    client.OnError(func(ev eventdriventcpclient.ErrorEvent) {
        log.Printf("error: %v", ev.Error)
    })

    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }

    _ = client.Send([]byte("hello\n"))
    time.Sleep(time.Second)
    _ = client.Disconnect()
}
```

### Example 2: Auto-Reconnect

```go
package main

import (
    "log"
    "time"

    "github.com/cyberinferno/go-utils/eventdriventcpclient"
)

func main() {
    cfg := eventdriventcpclient.DefaultEventDrivenTCPClientConfig("localhost:8080")
    cfg.AutoReconnect = true
    cfg.ReconnectInterval = 2 * time.Second

    client := eventdriventcpclient.NewEventDrivenTCPClient(cfg)
    defer client.Close()

    client.OnConnectionState(func(ev eventdriventcpclient.ConnectionStateEvent) {
        log.Printf("state: %s (err: %v)", ev.State, ev.Error)
    })
    client.OnDataReceived(func(ev eventdriventcpclient.DataReceivedEvent) {
        log.Printf("received %d bytes", ev.Length)
    })
    client.OnError(func(ev eventdriventcpclient.ErrorEvent) {
        log.Printf("error: %v", ev.Error)
    })

    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }

    // If the connection drops, the client will reconnect automatically.
    time.Sleep(60 * time.Second)
}
```

### Example 3: Length-Prefixed Messages

```go
package main

import (
    "encoding/binary"
    "log"
    "time"

    "github.com/cyberinferno/go-utils/eventdriventcpclient"
)

func main() {
    cfg := eventdriventcpclient.DefaultEventDrivenTCPClientConfig("localhost:9000")
    cfg.DataLengthBasedRead = true

    client := eventdriventcpclient.NewEventDrivenTCPClient(cfg)
    defer client.Close()

    client.OnDataReceived(func(ev eventdriventcpclient.DataReceivedEvent) {
        log.Printf("message: %q", string(ev.Data))
    })

    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }

    // Send a length-prefixed message
    msg := []byte("hello")
    buf := make([]byte, 4+len(msg))
    binary.LittleEndian.PutUint32(buf[:4], uint32(len(msg)))
    copy(buf[4:], msg)
    _ = client.Send(buf)

    time.Sleep(time.Second)
}
```

### Example 4: Check State Before Send

```go
if !client.IsConnected() {
    log.Println("not connected, skipping send")
    return
}
if err := client.Send(data); err != nil {
    log.Printf("send failed: %v", err)
}
```

---

## Type Reference

### EventDrivenTCPClient

```go
type EventDrivenTCPClient struct {
    // config, connection, state, handlers (unexported)
}
```

TCP client that drives I/O and connection lifecycle via events. Register handlers, then call `Connect`. Safe for concurrent use. Call `Close` when done.

### NewEventDrivenTCPClient

```go
func NewEventDrivenTCPClient(config Config) *EventDrivenTCPClient
```

Creates a new client with the given config.

### Config

```go
type Config struct {
    Address             string
    AutoReconnect       bool
    ReconnectInterval   time.Duration
    ReadBufferSize      int
    WriteTimeout        time.Duration
    ReadTimeout         time.Duration
    ConnectionTimeout   time.Duration
    DataLengthBasedRead bool
}
```

Configuration for the client. Use `DefaultEventDrivenTCPClientConfig(address)` and override fields.

### Methods

| Method | Description |
|--------|-------------|
| `OnConnectionState(handler ConnectionStateHandler)` | Registers handler for connection state changes; pass nil to clear. |
| `OnDataReceived(handler DataReceivedHandler)` | Registers handler for received data; pass nil to clear. |
| `OnError(handler ErrorHandler)` | Registers handler for errors; pass nil to clear. |
| `Connect() error` | Establishes TCP connection; starts read/reconnect goroutines when enabled. |
| `Disconnect() error` | Closes connection and moves to Disconnected; Connect may be called again. |
| `Close() error` | Shuts down client and all goroutines; idempotent. |
| `Send(data []byte) error` | Writes data; returns error if not connected or write fails. |
| `GetState() ConnectionState` | Returns current connection state. |
| `IsConnected() bool` | Returns true if state is Connected. |

### Event and Handler Types

| Type | Description |
|------|-------------|
| `ConnectionState` | Enum: Disconnected, Connecting, Connected, Reconnecting, Closed. |
| `ConnectionStateEvent` | State, Address, Timestamp, Error. |
| `DataReceivedEvent` | Data, Length, Timestamp. |
| `ErrorEvent` | Error, Timestamp. |
| `ConnectionStateHandler func(ConnectionStateEvent)` | Called on state change. |
| `DataReceivedHandler func(DataReceivedEvent)` | Called when data is received. |
| `ErrorHandler func(ErrorEvent)` | Called on read/write/connection error. |

---

## Best Practices

1. **Register handlers before Connect**: Set `OnConnectionState`, `OnDataReceived`, and `OnError` before calling `Connect` so you don’t miss early events.

2. **Copy received data if you need it later**: `DataReceivedEvent.Data` is a slice that may be reused; copy it if you need to keep the bytes after the handler returns.

3. **Don’t block handlers**: Handlers run in the client’s read/event goroutines; long-running work in a handler can delay processing. Offload work to another goroutine (e.g. send to a channel).

4. **Use Close when done**: Always call `Close()` (e.g. with `defer`) so goroutines and the connection are released.

5. **Check IsConnected before Send**: `Send` returns an error when not connected; checking `IsConnected()` first can avoid unnecessary errors and trigger reconnect logic if you use AutoReconnect.

6. **Choose read mode appropriately**: Use stream mode for raw streams or when you implement your own framing; use `DataLengthBasedRead = true` when the protocol is length-prefixed (4-byte little-endian + payload).

7. **Set timeouts in production**: Use `ConnectionTimeout`, `ReadTimeout`, and `WriteTimeout` to avoid hanging on dead connections.

---

## Limitations

- **Single connection**: One TCP connection per client; no connection pooling or multiple endpoints.
- **No TLS**: Plain TCP only; wrap with TLS at a higher layer if needed.
- **Length-prefixed max size**: In `DataLengthBasedRead` mode, messages larger than 16 MiB cause the read loop to exit.
- **One handler per type**: Registering a new handler replaces the previous one; for multiple listeners, fan out from a single handler.
- **Do not copy client**: The client must not be copied after first use (same as types containing mutexes).
