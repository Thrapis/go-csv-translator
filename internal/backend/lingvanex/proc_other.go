//go:build !windows

package lingvanex

import (
	"log/slog"
	"os/exec"
	"syscall"
	"time"
)

// configureProcAttr puts the child in its own process group so the whole group
// can be signalled at once on shutdown.
func configureProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// stopProcess sends SIGTERM to the process group, waits up to timeout, then
// SIGKILLs whatever is left.
func stopProcess(cmd *exec.Cmd, timeout time.Duration, log *slog.Logger) error {
	pid := cmd.Process.Pid
	signal := func(sig syscall.Signal) {
		if err := syscall.Kill(-pid, sig); err != nil {
			_ = cmd.Process.Signal(sig)
		}
	}

	signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		log.Warn("lingvanex: SIGTERM timed out, sending SIGKILL", "pid", pid)
		signal(syscall.SIGKILL)
		<-done
		return nil
	}
}
