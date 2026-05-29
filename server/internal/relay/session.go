package relay

import (
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
)

// SessionStats holds byte transfer statistics for a session.
type SessionStats struct {
	BytesIn  int64 // bytes from external to work (upload to tunnel)
	BytesOut int64 // bytes from work to external (download from tunnel)
}

// StatsCallback is called when a session completes to report bytes transferred.
type StatsCallback func(bytesIn, bytesOut int64)

// ProxySession bridges two TCP connections for a single proxied session.
type ProxySession struct {
	sessionID    string
	externalConn io.ReadWriteCloser
	workConn     io.ReadWriteCloser
	logger       *slog.Logger
	onComplete   StatsCallback
}

// NewProxySession creates a new session bridging external and work connections.
func NewProxySession(sessionID string, externalConn, workConn io.ReadWriteCloser, logger *slog.Logger, onComplete StatsCallback) *ProxySession {
	return &ProxySession{
		sessionID:    sessionID,
		externalConn: externalConn,
		workConn:     workConn,
		logger:       logger,
		onComplete:   onComplete,
	}
}

// Bridge starts bidirectional data forwarding between the two connections.
// It blocks until both directions are done.
func (s *ProxySession) Bridge() {
	var wg sync.WaitGroup
	var closeOnce sync.Once
	var bytesIn, bytesOut atomic.Int64

	closeBoth := func() {
		closeOnce.Do(func() {
			s.externalConn.Close()
			s.workConn.Close()
		})
	}

	wg.Add(2)

	// external -> work (bytes in from external user's perspective)
	go func() {
		defer wg.Done()
		n, err := io.Copy(s.workConn, s.externalConn)
		bytesIn.Store(n)
		s.logger.Debug("external->work done", "bytes", n, "error", err)
		closeBoth()
	}()

	// work -> external (bytes out to external user)
	go func() {
		defer wg.Done()
		n, err := io.Copy(s.externalConn, s.workConn)
		bytesOut.Store(n)
		s.logger.Debug("work->external done", "bytes", n, "error", err)
		closeBoth()
	}()

	wg.Wait()

	if s.onComplete != nil {
		s.onComplete(bytesIn.Load(), bytesOut.Load())
	}
}
