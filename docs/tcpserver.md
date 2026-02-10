# TCPServer Documentation

The `tcpserver` package provides a TCP server that accepts connections and delegates each one to a session. You supply a session factory (`NewSessionFunc`) and an ID generator; the server binds to an address, runs an accept loop in a goroutine, and starts each session's `Handle` in its own goroutine. Sessions are stored by ID and can be looked up, added, or removed. The server supports graceful stop: it closes the listener and all active sessions.

## Features

- **Session-per-connection**: Each accepted connection is wrapped in a `TCPServerSession` created by your `NewSessionFunc`
- **Concurrent safe**: Session map and server state are safe for concurrent access
- **Graceful stop**: `Stop()` closes the listener and closes all sessions that implement `Close() error`
- **Pluggable ID generator**: Use the `idgenerator` package or any `*idgenerator.IdGenerator` for session IDs
- **Pluggable logging**: Set `Logger` to integrate with your logging (e.g. `logger` package)

## Installation

```go
import "github.com/cyberinferno/go-utils/tcpserver"
```

## Dependencies

The server expects:

- **Logger**: Any type that implements `logger.Logger` (e.g. from `github.com/cyberinferno/go-utils/logger`)
- **IdGenerator**: A `*idgenerator.IdGenerator` (e.g. from `github.com/cyberinferno/go-utils/idgenerator`)
- **NewSession**: A function that creates a `TCPServerSession` from a session ID and `net.Conn`

## Creating a Server

### TCPServer struct

Build a `TCPServer` by setting its fields before calling `Start`. You must provide a listener address, a session factory, an ID generator, and a logger. Sessions can be a new `SafeMap` or reused.

| Field | Type | Description |
|-------|------|-------------|
| `Logger` | `logger.Logger` | Used for server start/stop and accept errors. Required. |
| `Name` | `string` | Server name used in log messages (e.g. `"game"`, `"api"`). |
| `Addr` | `string` | Listen address (e.g. `":8080"`, `"localhost:9000"`). |
| `Listener` | `net.Listener` | Set by `Start`; do not set before starting. |
| `Sessions` | `*safemap.SafeMap[uint32, TCPServerSession]` | Session storage. Initialize with `safemap.NewSafeMap[uint32, tcpserver.TCPServerSession]()`. |
| `Running` | `atomic.Bool` | Set by `Start`/`Stop`; optional to set beforehand. |
| `NewSession` | `NewSessionFunc` | Factory that creates a session for each connection. Required. |
| `IdGenerator` | `*idgenerator.IdGenerator` | Assigns unique session IDs. Required. |

Example:

```go
import (
	"github.com/cyberinferno/go-utils/idgenerator"
	"github.com/cyberinferno/go-utils/logger"
	"github.com/cyberinferno/go-utils/safemap"
	"github.com/cyberinferno/go-utils/tcpserver"
)

log := logger.New(...) // your logger

srv := &tcpserver.TCPServer{
	Logger:      log,
	Name:        "myserver",
	Addr:        ":8080",
	Sessions:    safemap.NewSafeMap[uint32, tcpserver.TCPServerSession](),
	NewSession:  myNewSession,
	IdGenerator: idgenerator.NewIdGenerator(0),
}

if err := srv.Start(); err != nil {
	log.Fatal("start failed", err)
}
defer srv.Stop()
```

---

## Implementing TCPServerSession

Your session type must implement the `TCPServerSession` interface:

```go
type TCPServerSession interface {
	ID() uint32
	Handle()
	Close() error
	Send(data []byte) error
}
```

- **ID**: Return the session ID passed to your `NewSessionFunc`.
- **Handle**: Run the read loop (or other logic). The server calls `go session.Handle()` for each connection; when `Handle` returns, the session is typically removed from the server.
- **Close**: Close the connection and release resources. Should be safe to call more than once. The server calls `Close` on each session when `Stop()` is called if the session implements `Close() error`.
- **Send**: Write data to the connection. Should be safe for concurrent use if multiple goroutines may call it.

Example skeleton:

```go
type MySession struct {
	id     uint32
	conn   net.Conn
	server *tcpserver.TCPServer
	done   chan struct{}
}

func (s *MySession) ID() uint32   { return s.id }
func (s *MySession) Handle()      { /* read loop; call s.server.RemoveSession(s.id) when done */ }
func (s *MySession) Close() error { close(s.done); return s.conn.Close() }
func (s *MySession) Send(data []byte) error { _, err := s.conn.Write(data); return err }
```

---

## Server Methods

### Start

Starts the TCP server by binding to `Addr` and beginning the accept loop in a goroutine. Call only when the server is not already running.

```go
if err := srv.Start(); err != nil {
	// server already running or listen failed
}
```

**Returns:**

- An error if the server is already running or if listening on `Addr` fails.

---

### Stop

Stops the server: sets `Running` to false, closes the listener, and closes all active sessions (any session that implements `Close() error` has `Close()` called). Safe to call when the server is not running.

```go
srv.Stop()
```

---

### AddSession

Stores a session under the given ID. Safe for concurrent use. Useful if you create sessions outside the accept loop (e.g. for testing or re-attach).

**Parameters:**

- **id**: The session ID to associate with the session.
- **session**: The session to store.

```go
srv.AddSession(123, mySession)
```

---

### RemoveSession

Removes the session with the given ID from the server. Safe for concurrent use. Typically called from the session’s `Handle` when the connection ends.

**Parameters:**

- **id**: The session ID to remove.

```go
srv.RemoveSession(sessionID)
```

---

### GetSession

Returns the session for the given ID, if present.

**Parameters:**

- **id**: The session ID to look up.

**Returns:**

- The session and `true` if found, or a zero value and `false` otherwise.

```go
if sess, ok := srv.GetSession(42); ok {
	_ = sess.Send([]byte("hello"))
}
```

---

### AcceptLoop

Runs in a goroutine started by `Start`. Accepts connections in a loop; for each connection it assigns an ID via `IdGenerator`, creates a session with `NewSession`, stores it with `AddSession`, and runs `session.Handle()` in a new goroutine. Exits when the server is stopped (`Running` is false). You do not normally call `AcceptLoop` directly.

---

## NewSessionFunc

`NewSessionFunc` is the type of the function that creates a new session for each accepted connection:

```go
type NewSessionFunc func(id uint32, conn net.Conn) TCPServerSession
```

- **id**: Unique session ID assigned by the server (from `IdGenerator`).
- **conn**: The accepted `net.Conn` (e.g. `*net.TCPConn`).

Return an implementation of `TCPServerSession` that will handle this connection. The server will call `AddSession(id, session)` and then `go session.Handle()`.

---

## Complete Example

```go
package main

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/cyberinferno/go-utils/idgenerator"
	"github.com/cyberinferno/go-utils/logger"
	"github.com/cyberinferno/go-utils/safemap"
	"github.com/cyberinferno/go-utils/tcpserver"
)

type EchoSession struct {
	id     uint32
	conn   net.Conn
	server *tcpserver.TCPServer
	closed sync.Once
}

func (e *EchoSession) ID() uint32 { return e.id }

func (e *EchoSession) Handle() {
	defer e.server.RemoveSession(e.id)
	buf := make([]byte, 4096)
	for {
		n, err := e.conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				// log read error
			}
			return
		}
		if _, err := e.conn.Write(buf[:n]); err != nil {
			return
		}
	}
}

func (e *EchoSession) Close() error {
	var err error
	e.closed.Do(func() { err = e.conn.Close() })
	return err
}

func (e *EchoSession) Send(data []byte) error {
	_, err := e.conn.Write(data)
	return err
}

func main() {
	log := logger.New(...)
	srv := &tcpserver.TCPServer{
		Logger:      log,
		Name:        "echo",
		Addr:        ":7000",
		Sessions:    safemap.NewSafeMap[uint32, tcpserver.TCPServerSession](),
		IdGenerator: idgenerator.NewIdGenerator(0),
		NewSession: func(id uint32, conn net.Conn) tcpserver.TCPServerSession {
			return &EchoSession{id: id, conn: conn, server: srv}
		},
	}
	if err := srv.Start(); err != nil {
		log.Fatal("start:", err)
	}
	defer srv.Stop()
	fmt.Println("Echo server on :7000")
	select {}
}
```

---

## Type Reference

### TCPServer

```go
type TCPServer struct {
	Logger      logger.Logger
	Name        string
	Addr        string
	Listener    net.Listener
	Sessions    *safemap.SafeMap[uint32, TCPServerSession]
	Running     atomic.Bool
	NewSession  NewSessionFunc
	IdGenerator *idgenerator.IdGenerator
}
```

TCP server that accepts connections and runs one `TCPServerSession` per connection. Set `Logger`, `Name`, `Addr`, `Sessions`, `NewSession`, and `IdGenerator` before `Start`.

### NewSessionFunc

```go
type NewSessionFunc func(id uint32, conn net.Conn) TCPServerSession
```

Creates a new session for an accepted connection. Receives the assigned session ID and the `net.Conn`; returns a `TCPServerSession`.

### TCPServerSession

```go
type TCPServerSession interface {
	ID() uint32
	Handle()
	Close() error
	Send(data []byte) error
}
```

Interface that each connection session must implement. The server runs `Handle` in a goroutine and calls `Close` on all sessions when `Stop()` is invoked (if the session implements `Close() error`).

### Methods summary

| Method | Description |
|--------|-------------|
| `Start() error` | Bind to `Addr` and start accept loop in a goroutine. |
| `Stop()` | Stop server, close listener and all sessions. |
| `AddSession(id uint32, session TCPServerSession)` | Store a session by ID. |
| `RemoveSession(id uint32)` | Remove session by ID. |
| `GetSession(id uint32) (TCPServerSession, bool)` | Look up session by ID. |
| `AcceptLoop()` | Accept loop (called internally by `Start`). |

---

## Best Practices

1. **Remove session in Handle**: When the connection ends (read error or EOF), call `server.RemoveSession(session.ID())` from inside `Handle` so the server does not retain closed sessions.

2. **Make Close idempotent**: Use a `sync.Once` or a mutex so `Close()` can be called multiple times without double-closing the connection.

3. **Synchronize Send**: If multiple goroutines can call `Send`, protect writes with a mutex or use a single writer goroutine and a channel.

4. **Log in session**: Use the server’s `Logger` (or a logger passed into your session) for per-connection errors so you can correlate with session ID.

---

## Limitations

- **No built-in TLS**: Wrap `net.Conn` in `tls.Server(conn, tlsConfig)` in your `NewSessionFunc` if you need TLS.
- **AcceptLoop blocks on Accept**: While the server is running, one goroutine is blocked in `Accept`. Ensure `Stop()` is called on shutdown so the listener is closed and the loop can exit.
- **Session map growth**: Sessions are only removed when you call `RemoveSession`; ensure `Handle` (or your cleanup logic) calls it when the connection is done.
