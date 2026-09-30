package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SshAuthorization struct{ User, Password, PrivateKey, KeyFile string }

// OperationError deliberately excludes commands, credentials and remote output.
// The wrapped cause remains available to errors.Is/As.
type OperationError struct {
	Target, Phase string
	Attempts      int
	Elapsed       time.Duration
	Uncertain     bool
	Err           error
}

func (e *OperationError) Error() string {
	uncertainty := ""
	if e.Uncertain {
		uncertainty = "; remote execution may have started; command was not replayed"
	}
	cause := e.Err.Error()
	var exit *ssh.ExitError
	if errors.As(e.Err, &exit) {
		// SSH exit-signal messages are remote-controlled and may echo secrets.
		cause = fmt.Sprintf("remote process exited with status %d", exit.ExitStatus())
	}
	return fmt.Sprintf("SSH %s: %s failed after %d attempt(s) (%s): %s%s", e.Target, e.Phase, e.Attempts, e.Elapsed.Round(time.Millisecond), cause, uncertainty)
}
func (e *OperationError) Unwrap() error { return e.Err }

type connectionPolicy struct {
	dialTimeout, handshakeTimeout, budget time.Duration
	attempts                              int
	dial                                  func(context.Context, string, string) (net.Conn, error)
	wait                                  func(context.Context, time.Duration) error
	delay                                 func(int) time.Duration
}

type SshClient struct {
	config     *ssh.ClientConfig
	Host, Port string
	Auth       SshAuthorization
	policy     connectionPolicy
	mu         sync.Mutex
	conn       *ssh.Client
	lifecycle  context.Context //nolint:containedctx // Close cancels this context to stop in-flight operations.
	cancel     context.CancelFunc
	operations chan struct{}
}

func NewSshClient(host, port string, auth SshAuthorization) (*SshClient, error) {
	if strings.TrimSpace(host) == "" || strings.TrimSpace(auth.User) == "" {
		return nil, errors.New("SSH host and user are required")
	}
	if port == "" {
		port = "22"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return nil, errors.New("SSH port must be between 1 and 65535")
	}
	config := &ssh.ClientConfig{User: auth.User, HostKeyCallback: ssh.InsecureIgnoreHostKey()}
	switch {
	case auth.KeyFile != "" || auth.PrivateKey != "":
		key := []byte(auth.PrivateKey)
		if auth.KeyFile != "" {
			key, err = os.ReadFile(auth.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("read SSH private key: %w", err)
			}
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("parse SSH private key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	case auth.Password != "":
		config.Auth = []ssh.AuthMethod{ssh.Password(auth.Password)}
	default:
		return nil, errors.New("SSH password or private key is required")
	}
	lifecycle, cancel := context.WithCancel(context.Background())
	return &SshClient{config: config, Host: host, Port: port, Auth: auth, lifecycle: lifecycle, cancel: cancel, operations: make(chan struct{}, 1), policy: connectionPolicy{
		dialTimeout: 15 * time.Second, handshakeTimeout: 30 * time.Second, budget: 90 * time.Second, attempts: 5,
		dial: (&net.Dialer{}).DialContext, wait: waitContext,
		delay: func(attempt int) time.Duration {
			return time.Second*time.Duration(1<<(attempt-1)) + time.Duration(rand.Int64N(int64(250*time.Millisecond))) // #nosec G404 -- jitter only spreads retry attempts.
		},
	}}, nil
}

func waitContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
func (c *SshClient) BaseAddress() string { return net.JoinHostPort(strings.Trim(c.Host, "[]"), c.Port) }
func (c *SshClient) Username() string    { return c.Auth.User }
func (c *SshClient) Password() string    { return c.Auth.Password }

// begin serializes sessions for this operation-scoped client and propagates Close.
func (c *SshClient) begin(parent context.Context) (context.Context, func(), error) {
	ctx, cancel := context.WithCancel(parent)
	stop := context.AfterFunc(c.lifecycle, cancel)
	cleanup := func() { stop(); cancel() }
	if c.lifecycle.Err() != nil {
		cleanup()
		return nil, nil, errors.New("SSH client is closed")
	}
	select {
	case c.operations <- struct{}{}:
		return ctx, func() { <-c.operations; cleanup() }, nil
	case <-ctx.Done():
		cleanup()
		return nil, nil, ctx.Err()
	}
}
func (c *SshClient) Connect() error { return c.ConnectContext(context.Background()) }
func (c *SshClient) ConnectContext(parent context.Context) error {
	ctx, done, err := c.begin(parent)
	if err != nil {
		return err
	}
	defer done()
	_, err = c.connection(ctx)
	return err
}

func (c *SshClient) connection(ctx context.Context) (*ssh.Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn != nil {
		return conn, nil
	}
	ctx, cancel := context.WithTimeout(ctx, c.policy.budget)
	defer cancel()
	start := time.Now()
	var err error
	phase := "connect"
	attempts := 0
	for attempts < c.policy.attempts {
		if ctx.Err() != nil {
			err = ctx.Err()
			break
		}
		attempts++
		conn, phase, err = c.dial(ctx)
		if err == nil {
			c.mu.Lock()
			if c.lifecycle.Err() != nil || ctx.Err() != nil {
				c.mu.Unlock()
				conn.Close()
				return nil, context.Canceled
			}
			c.conn = conn
			c.mu.Unlock()
			return conn, nil
		}
		if !transient(err) || attempts == c.policy.attempts {
			break
		}
		if err = c.policy.wait(ctx, c.policy.delay(attempts)); err != nil {
			break
		}
	}
	return nil, &OperationError{Target: c.BaseAddress(), Phase: phase, Attempts: attempts, Elapsed: time.Since(start), Err: err}
}

func (c *SshClient) dial(ctx context.Context) (*ssh.Client, string, error) {
	dialCtx, cancel := context.WithTimeout(ctx, c.policy.dialTimeout)
	raw, err := c.policy.dial(dialCtx, "tcp", c.BaseAddress())
	cancel()
	if err != nil {
		return nil, "connect", err
	}
	success := false
	defer func() {
		if !success {
			raw.Close()
		}
	}()
	deadline := time.Now().Add(c.policy.handshakeTimeout)
	if limit, ok := ctx.Deadline(); ok && limit.Before(deadline) {
		deadline = limit
	}
	if err = raw.SetDeadline(deadline); err != nil {
		return nil, "handshake", err
	}
	stop := context.AfterFunc(ctx, func() { raw.Close() })
	defer stop()
	config := *c.config
	config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := c.config.HostKeyCallback(hostname, remote, key); err != nil {
			return &hostKeyError{err: err}
		}
		return nil
	}
	conn, channels, requests, err := ssh.NewClientConn(raw, c.BaseAddress(), &config)
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		phase := "handshake"
		// x/crypto returns an unexported authentication error; classify it for diagnostics only.
		if strings.Contains(err.Error(), "unable to authenticate") {
			phase = "authenticate"
		}
		return nil, phase, err
	}
	if err = raw.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, "handshake", err
	}
	success = true
	return ssh.NewClient(conn, channels, requests), "", nil
}

type hostKeyError struct{ err error }

func (e *hostKeyError) Error() string { return "host key verification failed" }
func (e *hostKeyError) Unwrap() error { return e.err }

func transient(err error) bool {
	var keyError *hostKeyError
	if errors.As(err, &keyError) {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func (c *SshClient) discard(conn *ssh.Client) {
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	c.mu.Unlock()
	conn.Close()
}

// RunCommand treats arguments as literal argv, like exec.Command. Shell syntax
// must be passed explicitly to /bin/sh -c or /bin/bash -c.
func (c *SshClient) RunCommand(command string, args []string) (string, error) {
	return c.RunCommandContext(context.Background(), command, args)
}

func (c *SshClient) RunCommandContext(ctx context.Context, command string, args []string) (string, error) {
	return c.RunCommandInput(ctx, command, args, nil)
}

func (c *SshClient) RunCommandInput(parent context.Context, command string, args []string, input io.Reader) (string, error) {
	ctx, done, err := c.begin(parent)
	if err != nil {
		return "", err
	}
	defer done()
	start := time.Now()
	// Bound the whole setup, including replacement of a stale cached connection.
	setupCtx, cancelSetup := context.WithTimeout(ctx, c.policy.budget)
	defer cancelSetup()
	// Only opening a session is retryable: no exec request has been sent yet.
	var conn *ssh.Client
	var session *ssh.Session
	for attempt := 1; attempt <= 2; attempt++ {
		conn, err = c.connection(setupCtx)
		if err != nil {
			return "", err
		}
		sessionConn := conn
		stop := context.AfterFunc(setupCtx, func() { sessionConn.Close() })
		session, err = conn.NewSession()
		stop()
		if err == nil && setupCtx.Err() == nil {
			break
		}
		if session != nil {
			_ = session.Close()
		}
		c.discard(conn)
		if setupCtx.Err() != nil {
			err = setupCtx.Err()
		}
		if setupCtx.Err() != nil || !transient(err) || attempt == 2 {
			return "", &OperationError{Target: c.BaseAddress(), Phase: "open-session", Attempts: attempt, Elapsed: time.Since(start), Err: err}
		}
	}
	cancelSetup()
	defer session.Close()
	activeConn := conn
	stop := context.AfterFunc(ctx, func() { activeConn.Close() })
	defer stop()
	if err = ctx.Err(); err != nil {
		c.discard(conn)
		return "", err
	}
	session.Stdin = input
	parts := append([]string{command}, args...)
	for i := range parts {
		parts[i] = shellQuote(parts[i])
	}
	output, err := session.CombinedOutput(strings.Join(parts, " "))
	if err != nil {
		var exit *ssh.ExitError
		uncertain := !errors.As(err, &exit)
		if uncertain {
			c.discard(conn)
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		return string(output), &OperationError{Target: c.BaseAddress(), Phase: "run-command", Attempts: 1, Elapsed: time.Since(start), Uncertain: uncertain, Err: err}
	}
	return string(output), nil
}
func shellQuote(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

func (c *SshClient) TransferFile(local, remote string) error {
	return c.TransferFileContext(context.Background(), local, remote)
}

func (c *SshClient) TransferFileContext(parent context.Context, local, remote string) (result error) {
	ctx, done, err := c.begin(parent)
	if err != nil {
		return err
	}
	defer done()
	start := time.Now()
	defer func() {
		if result != nil {
			if ctx.Err() != nil {
				result = ctx.Err()
			}
			result = &OperationError{Target: c.BaseAddress(), Phase: "file-transfer", Attempts: 1, Elapsed: time.Since(start), Err: result}
		}
	}()
	f, err := os.Open(local) // #nosec G304 -- local is the caller-selected source file for transfer.
	if err != nil {
		return err
	}
	defer f.Close()
	conn, err := c.connection(ctx)
	if err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("SSH %s: open SFTP session: %w", c.BaseAddress(), err)
	}
	defer client.Close()
	// Exclusive creation prevents truncating or following an existing remote file.
	dest, err := client.OpenFile(remote, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return err
	}
	defer dest.Close()
	if err = dest.Chmod(0o600); err != nil {
		return err
	}
	if _, err = io.Copy(dest, f); err != nil {
		return fmt.Errorf("SSH %s: file transfer incomplete (not replayed): %w", c.BaseAddress(), err)
	}
	return dest.Close()
}

func (c *SshClient) Close() error {
	c.cancel()
	c.mu.Lock()
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
	return nil
}
