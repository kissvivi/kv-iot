// Package tcpserver provides small, reusable TCP test servers for protocol
// integration tests. It intentionally uses only the standard library.
package tcpserver

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
)

// ErrClosed is returned when Start is called on a closed server.
var ErrClosed = errors.New("tcpserver: server is closed")

// Handler serves a single accepted connection. Returning nil ends the
// connection cleanly; returning an error is treated as a connection failure.
type Handler func(ctx context.Context, conn net.Conn) error

// Config controls the listening endpoint used by a Server.
type Config struct {
	Addr string
}

// Server is a startable TCP listener backed by a test handler.
type Server struct {
	cfg     Config
	handler Handler

	ctx    context.Context
	cancel context.CancelFunc

	mu         sync.Mutex
	listener   net.Listener
	conns      map[net.Conn]struct{}
	started    bool
	closed     bool
	acceptDone chan struct{}
	handlers   sync.WaitGroup
	closeOnce  sync.Once
}

// New creates a server using the default loopback address.
func New(handler Handler) *Server {
	return NewWithConfig(Config{}, handler)
}

// NewWithConfig creates a server with an explicit address.
func NewWithConfig(cfg Config, handler Handler) *Server {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:0"
	}
	if handler == nil {
		handler = func(_ context.Context, conn net.Conn) error {
			_, err := io.Copy(io.Discard, conn)
			return err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		cfg:        cfg,
		handler:    handler,
		ctx:        ctx,
		cancel:     cancel,
		conns:      make(map[net.Conn]struct{}),
		acceptDone: make(chan struct{}),
	}
}

// Start begins accepting connections. It is not safe to call Start more than
// once for the same Server.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClosed
	}
	if s.started {
		s.mu.Unlock()
		return errors.New("tcpserver: server already started")
	}

	listener, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.listener = listener
	s.started = true
	s.mu.Unlock()

	go s.acceptLoop(listener)
	return nil
}

// Addr returns the actual listening address after Start.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Close stops the listener, closes active connections, and waits for handlers.
func (s *Server) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		listener := s.listener
		started := s.started
		conns := make([]net.Conn, 0, len(s.conns))
		for conn := range s.conns {
			conns = append(conns, conn)
		}
		s.mu.Unlock()

		s.cancel()
		if listener != nil {
			closeErr = listener.Close()
			if errors.Is(closeErr, net.ErrClosed) {
				closeErr = nil
			}
		}
		for _, conn := range conns {
			_ = conn.Close()
		}
		if started {
			<-s.acceptDone
		}
		s.handlers.Wait()
	})
	return closeErr
}

func (s *Server) acceptLoop(listener net.Listener) {
	defer close(s.acceptDone)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		s.conns[conn] = struct{}{}
		s.handlers.Add(1)
		s.mu.Unlock()

		go s.serveConnection(conn)
	}
}

func (s *Server) serveConnection(conn net.Conn) {
	defer s.handlers.Done()
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()
		_ = conn.Close()
	}()

	_ = s.handler(s.ctx, conn)
}
