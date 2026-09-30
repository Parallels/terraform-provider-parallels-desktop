package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

type testServer struct {
	stallSessions atomic.Bool
	stallSFTP     atomic.Bool
	acceptDone    chan struct{}
	listener      net.Listener
	config        *gossh.ServerConfig
	accepts       atomic.Int32
	execs         atomic.Int32
	resets        int32
	started       chan string
	mu            sync.Mutex
	connections   []net.Conn
	wg            sync.WaitGroup
}

func newTestServer(t *testing.T, resets int32) *testServer {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := gossh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	config := &gossh.ServerConfig{PasswordCallback: func(meta gossh.ConnMetadata, password []byte) (*gossh.Permissions, error) {
		if meta.User() == "deploy" && string(password) == "a'\"$`\\ secret" {
			return nil, nil
		}
		return nil, errors.New("authentication denied")
	}}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &testServer{acceptDone: make(chan struct{}), listener: listener, config: config, resets: resets, started: make(chan string, 20)}
	server.wg.Add(1)
	go func() {
		defer close(server.acceptDone)
		defer server.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			server.mu.Lock()
			server.connections = append(server.connections, conn)
			server.mu.Unlock()
			n := server.accepts.Add(1)
			if n <= server.resets {
				if err := conn.(*net.TCPConn).SetLinger(0); err != nil {
					t.Errorf("set TCP linger: %v", err)
				}
				conn.Close()
				continue
			}
			server.wg.Add(1)
			go server.serve(conn)
		}
	}()
	t.Cleanup(func() {
		listener.Close()
		<-server.acceptDone
		server.mu.Lock()
		for _, conn := range server.connections {
			conn.Close()
		}
		server.mu.Unlock()
		server.wg.Wait()
	})
	return server
}

func (s *testServer) serve(raw net.Conn) {
	defer s.wg.Done()
	defer raw.Close()
	conn, channels, requests, err := gossh.NewServerConn(raw, s.config)
	if err != nil {
		return
	}
	defer conn.Close()
	go gossh.DiscardRequests(requests)
	for channel := range channels {
		if s.stallSessions.Load() {
			_ = conn.Wait()
			return
		}
		ch, requests, err := channel.Accept()
		if err != nil {
			return
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer ch.Close()
			for request := range requests {
				if request.Type == "subsystem" {
					if s.stallSFTP.Load() {
						s.started <- "sftp"
						_ = conn.Wait()
						return
					}
					var subsystem struct{ Name string }
					if err := gossh.Unmarshal(request.Payload, &subsystem); err != nil {
						return
					}
					if subsystem.Name != "sftp" {
						if err := request.Reply(false, nil); err != nil {
							return
						}
						continue
					}
					if err := request.Reply(true, nil); err != nil {
						return
					}
					server, err := sftp.NewServer(ch)
					if err != nil {
						return
					}
					defer server.Close()
					_ = server.Serve()
					return
				}
				if request.Type != "exec" {
					if err := request.Reply(false, nil); err != nil {
						return
					}
					continue
				}
				var payload struct{ Command string }
				if gossh.Unmarshal(request.Payload, &payload) != nil {
					return
				}
				s.execs.Add(1)
				if err := request.Reply(true, nil); err != nil {
					return
				}
				s.started <- payload.Command
				if payload.Command == "'drop'" {
					raw.Close()
					return
				}
				if payload.Command == "'block'" {
					_ = conn.Wait()
					return
				}
				cmd := exec.Command("/bin/sh", "-c", payload.Command)
				cmd.Stdin = ch
				output, err := cmd.CombinedOutput()
				_, _ = ch.Write(output)
				code := uint32(0)
				if err != nil {
					code = 1
				}
				_, _ = ch.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{code}))
				return
			}
		}()
	}
}

func testClient(t *testing.T, s *testServer) *SshClient {
	t.Helper()
	host, port, _ := net.SplitHostPort(s.listener.Addr().String())
	c, err := NewSshClient(host, port, SshAuthorization{User: "deploy", Password: "a'\"$`\\ secret"})
	if err != nil {
		t.Fatal(err)
	}
	c.policy.delay = func(int) time.Duration { return 0 }
	t.Cleanup(func() { c.Close() })
	return c
}

func TestHandshakeRetriesAndConnectionReuse(t *testing.T) {
	s := newTestServer(t, 2)
	c := testClient(t, s)
	if err := c.ConnectContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		output, err := c.RunCommandContext(context.Background(), "printf", []string{"%s", "hello"})
		if err != nil || output != "hello" {
			t.Fatalf("%q, %v", output, err)
		}
	}
	if s.accepts.Load() != 3 || s.execs.Load() != 3 {
		t.Fatalf("connections=%d execs=%d", s.accepts.Load(), s.execs.Load())
	}
	c.Close()
	c.Close()
	if _, err := c.RunCommand("echo", nil); err == nil {
		t.Fatal("closed client accepted a command")
	}
}

func TestCommandsAreLiteralAndInputPreserved(t *testing.T) {
	s := newTestServer(t, 0)
	c := testClient(t, s)
	value := "a'\"$HOME `echo bad` $(echo bad); \\ spaces\nsecond line"
	out, err := c.RunCommandContext(context.Background(), "printf", []string{"%s", value})
	if err != nil || out != value {
		t.Fatalf("literal args: %q %v", out, err)
	}
	out, err = c.RunCommandInput(context.Background(), "cat", nil, strings.NewReader(value))
	if err != nil || out != value {
		t.Fatalf("stdin: %q %v", out, err)
	}
	_, err = c.RunCommandContext(context.Background(), "/bin/sh", []string{"-c", "printf '%s' \"$1\" >&2; exit 1", "test", value})
	if err == nil || strings.Contains(err.Error(), value) {
		t.Fatalf("error exposed output: %v", err)
	}
	var exit *gossh.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("lost exit cause: %v", err)
	}
}

func TestDroppedExecutionIsNotReplayed(t *testing.T) {
	s := newTestServer(t, 0)
	c := testClient(t, s)
	_, err := c.RunCommand("drop", nil)
	var operation *OperationError
	if !errors.As(err, &operation) || !operation.Uncertain {
		t.Fatalf("expected uncertain execution: %v", err)
	}
	if s.execs.Load() != 1 || s.accepts.Load() != 1 {
		t.Fatal("command replayed")
	}
	out, err := c.RunCommand("echo", []string{"next"})
	if err != nil || out != "next\n" {
		t.Fatalf("next command: %q %v", out, err)
	}
	if s.execs.Load() != 2 || s.accepts.Load() != 2 {
		t.Fatal("did not reconnect for next command")
	}
}

func TestReconnectBeforeOpeningSession(t *testing.T) {
	s := newTestServer(t, 0)
	c := testClient(t, s)
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	c.mu.Lock()
	c.conn.Close()
	c.mu.Unlock()
	if _, err := c.RunCommand("true", nil); err != nil {
		t.Fatal(err)
	}
	if s.accepts.Load() != 2 || s.execs.Load() != 1 {
		t.Fatal("unexpected connection or exec count")
	}
}

func TestAuthenticationAndHostKeyFailuresAreTerminal(t *testing.T) {
	for _, kind := range []string{"authentication", "host-key"} {
		t.Run(kind, func(t *testing.T) {
			s := newTestServer(t, 0)
			c := testClient(t, s)
			if kind == "authentication" {
				c.config.Auth = []gossh.AuthMethod{gossh.Password("wrong")}
			} else {
				c.config.HostKeyCallback = func(string, net.Addr, gossh.PublicKey) error {
					return fmt.Errorf("host key rejected: %w", syscall.ECONNRESET)
				}
			}
			err := c.Connect()
			if err == nil {
				t.Fatal("expected error")
			}
			if s.accepts.Load() != 1 {
				t.Fatalf("terminal error retried %d times", s.accepts.Load())
			}
			if strings.Contains(err.Error(), c.Password()) {
				t.Fatal("password leaked")
			}
		})
	}
}

func TestBoundedHandshakeAndCancellation(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(strconv.FormatBool(cancelled), func(t *testing.T) {
			c, err := NewSshClient("test", "", SshAuthorization{User: "user", Password: "secret"})
			if err != nil {
				t.Fatal(err)
			}
			defer c.Close()
			var peers []net.Conn
			defer func() {
				for _, p := range peers {
					p.Close()
				}
			}()
			c.policy.dial = func(context.Context, string, string) (net.Conn, error) {
				a, b := net.Pipe()
				peers = append(peers, b)
				return a, nil
			}
			c.policy.handshakeTimeout = 20 * time.Millisecond
			c.policy.attempts = 2
			c.policy.delay = func(int) time.Duration { return 0 }
			ctx := context.Background()
			if cancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 5*time.Millisecond)
				defer cancel()
			}
			start := time.Now()
			err = c.ConnectContext(ctx)
			var operation *OperationError
			if !errors.As(err, &operation) || operation.Phase != "handshake" {
				t.Fatalf("%v", err)
			}
			if time.Since(start) > time.Second {
				t.Fatal("handshake was not bounded")
			}
			if cancelled && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("lost context error")
			}
			if !cancelled && operation.Attempts != 2 {
				t.Fatalf("attempts=%d", operation.Attempts)
			}
		})
	}
}

func TestRetryBudgetAndCause(t *testing.T) {
	c, _ := NewSshClient("test", "22", SshAuthorization{User: "u", Password: "p"})
	defer c.Close()
	c.policy.dial = func(context.Context, string, string) (net.Conn, error) {
		return nil, fmt.Errorf("dial: %w", syscall.ECONNRESET)
	}
	var waits []time.Duration
	c.policy.delay = func(attempt int) time.Duration { return time.Second * time.Duration(1<<(attempt-1)) }
	c.policy.wait = func(ctx context.Context, d time.Duration) error { waits = append(waits, d); return nil }
	err := c.Connect()
	var operation *OperationError
	if !errors.As(err, &operation) || operation.Attempts != 5 || !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("%v", err)
	}
	if fmt.Sprint(waits) != "[1s 2s 4s 8s]" {
		t.Fatalf("waits: %v", waits)
	}
	c.policy.budget = 10 * time.Millisecond
	c.policy.wait = waitContext
	start := time.Now()
	err = c.Connect()
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("budget: %v", err)
	}
}

func TestCancelCommandAndCloseInterrupts(t *testing.T) {
	for _, closeClient := range []bool{false, true} {
		t.Run(strconv.FormatBool(closeClient), func(t *testing.T) {
			s := newTestServer(t, 0)
			c := testClient(t, s)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { _, err := c.RunCommandContext(ctx, "block", nil); result <- err }()
			select {
			case <-s.started:
			case <-time.After(time.Second):
				t.Fatal("command not started")
			}
			if closeClient {
				c.Close()
			} else {
				cancel()
			}
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("expected cancellation")
				}
			case <-time.After(time.Second):
				t.Fatal("command not cancelled")
			}
			if s.execs.Load() != 1 {
				t.Fatal("cancelled command replayed")
			}
		})
	}
}

func TestSeparateClientsAndConcurrentSessions(t *testing.T) {
	s := newTestServer(t, 0)
	a, b := testClient(t, s), testClient(t, s)
	var wg sync.WaitGroup
	for i := range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := a
			if i%2 == 0 {
				c = b
			}
			if _, err := c.RunCommand("true", nil); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if s.accepts.Load() != 2 || s.execs.Load() != 12 {
		t.Fatalf("connections=%d execs=%d", s.accepts.Load(), s.execs.Load())
	}
}

func TestTransferUsesExistingConnection(t *testing.T) {
	s := newTestServer(t, 0)
	c := testClient(t, s)
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	dest := filepath.Join(dir, "dest")
	if err := os.WriteFile(source, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := c.TransferFileContext(context.Background(), source, dest); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dest)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file permissions: %v %v", info, err)
	}
	content, _ := os.ReadFile(dest)
	if string(content) != "secret" {
		t.Fatal("wrong content")
	}
	if err := c.TransferFile(source, dest); err == nil {
		t.Fatal("overwrote existing destination")
	}
	if s.accepts.Load() != 1 {
		t.Fatal("transfer opened new transport")
	}
}

func TestValidateSSHConfiguration(t *testing.T) {
	if _, err := NewSshClient("host", "", SshAuthorization{User: "u", PrivateKey: "invalid"}); err == nil {
		t.Fatal("invalid key accepted")
	}
	for _, port := range []string{"-1", "0", "65536", "bad"} {
		if _, err := NewSshClient("host", port, SshAuthorization{User: "u", Password: "p"}); err == nil {
			t.Fatalf("port %s accepted", port)
		}
	}
	c, err := NewSshClient("::1", "", SshAuthorization{User: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.BaseAddress() != "[::1]:22" {
		t.Fatal(c.BaseAddress())
	}
}

func TestSessionSetupHasBoundedBudget(t *testing.T) {
	s := newTestServer(t, 0)
	c := testClient(t, s)
	if err := c.Connect(); err != nil {
		t.Fatal(err)
	}
	s.stallSessions.Store(true)
	c.policy.budget = 20 * time.Millisecond
	start := time.Now()
	_, err := c.RunCommand("true", nil)
	var operation *OperationError
	if !errors.As(err, &operation) || operation.Phase != "open-session" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("%v", err)
	}
	if time.Since(start) > time.Second || s.execs.Load() != 0 {
		t.Fatal("unbounded setup or command submitted")
	}
}

func TestTransferCancellation(t *testing.T) {
	s := newTestServer(t, 0)
	s.stallSFTP.Store(true)
	c := testClient(t, s)
	source := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(source, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- c.TransferFileContext(ctx, source, "/unused") }()
	select {
	case <-s.started:
	case <-time.After(time.Second):
		t.Fatal("SFTP did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SFTP was not cancelled")
	}
	if s.accepts.Load() != 1 {
		t.Fatal("transfer was replayed")
	}
}
