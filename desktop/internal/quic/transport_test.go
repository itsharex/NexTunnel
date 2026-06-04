package quic

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"sync"
	"testing"
	"time"
)

func testClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true, // 仅测试自签证书使用，生产路径必须传入可信 TLS 配置。
		NextProtos:         []string{DefaultQUICConfig().ALPN},
	}
}

// TestQUICTransport_LocalLoopback verifies bidirectional data transfer on localhost.
func TestQUICTransport_LocalLoopback(t *testing.T) {
	cfg := DefaultQUICConfig()
	cfg.ListenAddr = "127.0.0.1:0"

	ln, err := NewListener(cfg)
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()
	t.Logf("QUIC listener at %s", addr)

	serverTransport := NewQUICTransport()
	var serverErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		serverErr = serverTransport.AcceptFromListener(ctx, ln)
	}()

	clientTransport := NewQUICTransport(WithTLSConfig(testClientTLSConfig()))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := clientTransport.Dial(ctx, addr); err != nil {
		t.Fatalf("Dial: %v", err)
	}

	wg.Wait()
	if serverErr != nil {
		t.Fatalf("Accept: %v", serverErr)
	}

	t.Logf("Client: local=%v remote=%v", clientTransport.LocalAddr(), clientTransport.RemoteAddr())
	t.Logf("Server: local=%v remote=%v", serverTransport.LocalAddr(), serverTransport.RemoteAddr())

	// Client -> Server
	testData := []byte("hello from QUIC client")
	if _, err := clientTransport.Write(testData); err != nil {
		t.Fatalf("client write: %v", err)
	}

	buf := make([]byte, 1024)
	n, err := serverTransport.Read(buf)
	if err != nil {
		t.Fatalf("server read: %v", err)
	}
	if !bytes.Equal(buf[:n], testData) {
		t.Fatalf("data mismatch: got %q, want %q", buf[:n], testData)
	}

	// Server -> Client
	reply := []byte("hello from QUIC server")
	if _, err := serverTransport.Write(reply); err != nil {
		t.Fatalf("server write: %v", err)
	}

	n, err = clientTransport.Read(buf)
	if err != nil {
		t.Fatalf("client read: %v", err)
	}
	if !bytes.Equal(buf[:n], reply) {
		t.Fatalf("reply mismatch: got %q, want %q", buf[:n], reply)
	}

	if clientTransport.State() != QUICStateConnected {
		t.Errorf("client state = %v, want %v", clientTransport.State(), QUICStateConnected)
	}
	if serverTransport.State() != QUICStateConnected {
		t.Errorf("server state = %v, want %v", serverTransport.State(), QUICStateConnected)
	}

	clientTransport.Close()
	serverTransport.Close()

	if clientTransport.State() != QUICStateClosed {
		t.Errorf("client state after close = %v, want %v", clientTransport.State(), QUICStateClosed)
	}

	t.Log("QUIC local loopback test PASSED")
}

// TestQUICStream_Multiplexing verifies multiple streams on one connection.
func TestQUICStream_Multiplexing(t *testing.T) {
	cfg := DefaultQUICConfig()
	cfg.ListenAddr = "127.0.0.1:0"

	ln, err := NewListener(cfg)
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	serverTransport := NewQUICTransport()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		serverTransport.AcceptFromListener(ctx, ln)
	}()

	clientTransport := NewQUICTransport(WithTLSConfig(testClientTLSConfig()))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := clientTransport.Dial(ctx, addr); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	wg.Wait()

	const numStreams = 5
	var streamWg sync.WaitGroup
	received := make([]string, numStreams)

	// Client opens streams and sends data
	for i := 0; i < numStreams; i++ {
		streamWg.Add(1)
		go func(idx int) {
			defer streamWg.Done()
			s, err := clientTransport.OpenStream()
			if err != nil {
				t.Errorf("stream %d: open: %v", idx, err)
				return
			}
			msg := []byte("stream-" + string(rune('A'+idx)))
			if _, err := s.Write(msg); err != nil {
				t.Errorf("stream %d: write: %v", idx, err)
			}
			s.Close()
		}(i)
	}

	// Server accepts streams and reads data
	for i := 0; i < numStreams; i++ {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		s, err := serverTransport.AcceptStream(ctx2)
		cancel2()
		if err != nil {
			t.Fatalf("accept stream %d: %v", i, err)
		}

		buf := make([]byte, 1024)
		n, err := s.Read(buf)
		if err != nil && err != io.EOF {
			t.Errorf("read stream %d: %v", i, err)
		}
		received[i] = string(buf[:n])
		t.Logf("Server received: %q", received[i])
		s.Close()
	}

	streamWg.Wait()

	// Verify all messages received (order may differ)
	for i, msg := range received {
		if msg == "" {
			t.Errorf("stream %d: no data received", i)
		}
	}

	clientTransport.Close()
	serverTransport.Close()

	t.Log("QUIC multiplexing test PASSED")
}

// TestQUICConfig_Options verifies all functional options apply correctly.
func TestQUICConfig_Options(t *testing.T) {
	t1 := NewQUICTransport(
		With0RTT(false),
		WithConnectionMigration(false),
		WithMaxStreams(50),
		WithKeepAlive(30*time.Second),
		WithListenAddr("127.0.0.1:0"),
		WithALPN("test-proto"),
	)

	if t1.config.Enable0RTT {
		t.Error("Enable0RTT should be false")
	}
	if t1.config.EnableMigration {
		t.Error("EnableMigration should be false")
	}
	if t1.config.MaxStreams != 50 {
		t.Errorf("MaxStreams = %d, want 50", t1.config.MaxStreams)
	}
	if t1.config.KeepAlive != 30*time.Second {
		t.Errorf("KeepAlive = %v, want 30s", t1.config.KeepAlive)
	}
	if t1.config.ListenAddr != "127.0.0.1:0" {
		t.Errorf("ListenAddr = %s, want 127.0.0.1:0", t1.config.ListenAddr)
	}
	if t1.config.ALPN != "test-proto" {
		t.Errorf("ALPN = %s, want test-proto", t1.config.ALPN)
	}

	t.Log("QUIC config options test PASSED")
}

// TestQUICTransport_InterfaceCompliance verifies QUICTransport satisfies
// Read/Write/Close/LocalAddr/RemoteAddr interface methods.
func TestQUICTransport_InterfaceCompliance(t *testing.T) {
	cfg := DefaultQUICConfig()
	cfg.ListenAddr = "127.0.0.1:0"

	ln, err := NewListener(cfg)
	if err != nil {
		t.Fatalf("NewListener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	var wg sync.WaitGroup
	wg.Add(1)
	server := NewQUICTransport()
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.AcceptFromListener(ctx, ln)
	}()

	client := NewQUICTransport(WithTLSConfig(testClientTLSConfig()))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Dial(ctx, addr); err != nil {
		t.Fatalf("Dial: %v", err)
	}
	wg.Wait()

	data := []byte("interface compliance test")
	n, err := client.Write(data)
	if err != nil || n != len(data) {
		t.Fatalf("Write: n=%d, err=%v", n, err)
	}

	buf := make([]byte, 1024)
	n, err = server.Read(buf)
	if err != nil || n != len(data) {
		t.Fatalf("Read: n=%d, err=%v", n, err)
	}
	if !bytes.Equal(buf[:n], data) {
		t.Fatalf("data mismatch")
	}

	if client.LocalAddr() == nil {
		t.Error("client LocalAddr is nil")
	}
	if client.RemoteAddr() == nil {
		t.Error("client RemoteAddr is nil")
	}
	if server.LocalAddr() == nil {
		t.Error("server LocalAddr is nil")
	}
	if server.RemoteAddr() == nil {
		t.Error("server RemoteAddr is nil")
	}

	if err := client.Close(); err != nil {
		t.Errorf("client Close: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Errorf("server Close: %v", err)
	}

	t.Log("QUIC interface compliance test PASSED")
}

func TestQUICTransport_RequiresTLSConfig(t *testing.T) {
	client := NewQUICTransport()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := client.Dial(ctx, "127.0.0.1:1"); err == nil {
		t.Fatal("expected TLSConfig validation error")
	}
	if client.State() != QUICStateError {
		t.Fatalf("state = %v, want %v", client.State(), QUICStateError)
	}
}
