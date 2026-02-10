package tcpserver

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/cyberinferno/go-utils/idgenerator"
	"github.com/cyberinferno/go-utils/logger"
	"github.com/cyberinferno/go-utils/safemap"
)

// NewSessionFunc is a function that creates a new TCPServerSession for a given
// connection. It receives the assigned session ID and the accepted net.Conn,
// and returns an implementation of TCPServerSession that will handle the connection.
type NewSessionFunc func(id uint32, conn net.Conn) TCPServerSession

// TCPServer is a TCP server that accepts connections and delegates each one to a
// session created by NewSession. Sessions are stored by ID and can be looked up,
// added, or removed. The server runs its accept loop in a goroutine and supports
// graceful stop.
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

// Start starts the TCP server by binding to Addr and beginning the accept loop
// in a goroutine. It is safe to call only when the server is not already running.
//
// Returns:
//   - An error if the server is already running or if listening on Addr fails
func (s *TCPServer) Start() error {
	if s.Running.Load() {
		s.Logger.Error("server already running")
		return fmt.Errorf("server %s already running", s.Name)
	}

	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		s.Logger.Error("server failed to start", logger.Field{Key: "error", Value: err})
		return fmt.Errorf("server %s failed to start: %w", s.Name, err)
	}

	s.Listener = ln
	s.Running.Store(true)

	s.Logger.Info(fmt.Sprintf("%s server started", s.Name), logger.Field{Key: "addr", Value: s.Addr})
	go s.AcceptLoop()

	return nil
}

// Stop stops the TCP server: it sets Running to false, closes the listener, and
// closes all active sessions. Safe to call when the server is not running.
func (s *TCPServer) Stop() {
	if !s.Running.Load() {
		s.Logger.Info(fmt.Sprintf("%s server not running", s.Name))
		return
	}

	s.Running.Store(false)
	if s.Listener != nil {
		_ = s.Listener.Close()
	}

	s.Sessions.Range(func(key uint32, session TCPServerSession) bool {
		if closer, ok := any(session).(interface{ Close() error }); ok {
			_ = closer.Close()
		}

		return true
	})

	s.Logger.Info(fmt.Sprintf("%s server stopped", s.Name))
}

// AddSession stores a session under the given id. It is safe for concurrent use.
//
// Parameters:
//   - id: The session ID to associate with the session
//   - session: The session to store
func (s *TCPServer) AddSession(id uint32, session TCPServerSession) {
	s.Sessions.Store(id, session)
}

// RemoveSession removes the session with the given id from the server. It is
// safe for concurrent use.
//
// Parameters:
//   - id: The session ID to remove
func (s *TCPServer) RemoveSession(id uint32) {
	s.Sessions.Delete(id)
}

// GetSession returns the session for the given id, if present.
//
// Parameters:
//   - id: The session ID to look up
//
// Returns:
//   - The session and true if found, or a zero value and false otherwise
func (s *TCPServer) GetSession(id uint32) (TCPServerSession, bool) {
	return s.Sessions.Get(id)
}

// AcceptLoop runs in a goroutine and accepts incoming connections. For each
// connection it assigns an ID via IdGenerator, creates a session with NewSession,
// stores it with AddSession, and runs session.Handle in a new goroutine. It
// exits when the server is stopped (Running is false).
func (s *TCPServer) AcceptLoop() {
	for s.Running.Load() {
		conn, err := s.Listener.Accept()
		if err != nil {
			if !s.Running.Load() {
				return
			}

			s.Logger.Error(fmt.Sprintf("%s server accept error", s.Name), logger.Field{Key: "error", Value: err})
			continue
		}

		id := s.IdGenerator.Id()
		session := s.NewSession(id, conn)
		s.AddSession(id, session)
		go session.Handle()
	}
}
