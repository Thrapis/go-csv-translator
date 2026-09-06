// Package lingvanex supervises the local Lingvanex Python translation server:
// it starts the process on demand, waits for the port to accept connections,
// streams its output to the logger, and stops it (and its children) on shutdown.
//
// It is separate from internal/translate/lingvanex, which is the HTTP client
// used to actually request translations.
package lingvanex

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os/exec"
	"time"

	"github.com/Thrapis/go-csv-translator/internal/config"
)

// Server manages one Lingvanex server process.
type Server struct {
	cfg  config.Lingvanex
	log  *slog.Logger
	addr string
	cmd  *exec.Cmd
}

// New returns a supervisor for the given config. It does not start anything.
func New(cfg config.Lingvanex, log *slog.Logger) *Server {
	return &Server{
		cfg:  cfg,
		log:  log,
		addr: net.JoinHostPort(cfg.Address, fmt.Sprint(cfg.Port)),
	}
}

// Start launches the server (when cfg.Manage) and blocks until its port is
// reachable or the health timeout elapses. When cfg.Manage is false it only
// verifies that something is already listening, failing fast otherwise.
func (s *Server) Start(ctx context.Context) error {
	if !s.cfg.Manage {
		s.log.Info("lingvanex: using externally managed server", "addr", s.addr)
		if err := s.waitReachable(ctx, 5*time.Second); err != nil {
			return fmt.Errorf("lingvanex: no server reachable at %s and manage=false: %w", s.addr, err)
		}
		return nil
	}

	if len(s.cfg.Command) == 0 {
		return errors.New("lingvanex: command is empty")
	}

	s.log.Info("lingvanex: starting server",
		"command", s.cfg.Command, "workdir", s.cfg.Workdir, "addr", s.addr)

	cmd := exec.Command(s.cfg.Command[0], s.cfg.Command[1:]...)
	cmd.Dir = s.cfg.Workdir
	configureProcAttr(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("lingvanex: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("lingvanex: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("lingvanex: start %v: %w", s.cfg.Command, err)
	}
	s.cmd = cmd
	go s.pipe("stdout", stdout)
	go s.pipe("stderr", stderr)

	if err := s.waitReachable(ctx, s.cfg.HealthTimeout.Duration()); err != nil {
		_ = s.Stop()
		return fmt.Errorf("lingvanex: server did not come up: %w", err)
	}
	s.log.Info("lingvanex: server ready", "addr", s.addr)
	return nil
}

// Stop terminates the managed process and its children. It is a no-op when the
// server is externally managed or was never started, and is safe to call twice.
func (s *Server) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	s.log.Info("lingvanex: stopping server", "pid", s.cmd.Process.Pid)
	err := stopProcess(s.cmd, s.cfg.StopTimeout.Duration(), s.log)
	s.cmd = nil
	return err
}

func (s *Server) waitReachable(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		conn, err := net.DialTimeout("tcp", s.addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s not reachable within %s", s.addr, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *Server) pipe(stream string, r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		s.log.Debug("lingvanex", "stream", stream, "line", sc.Text())
	}
}
