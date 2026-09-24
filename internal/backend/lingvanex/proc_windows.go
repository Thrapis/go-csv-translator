//go:build windows

package lingvanex

import (
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// configureProcAttr is a no-op on Windows; the process tree is torn down with
// taskkill in stopProcess instead of relying on a process group.
func configureProcAttr(*exec.Cmd) {}

// stopProcess kills the process and every child with taskkill /T, since `py`
// spawns a separate python.exe that a plain Process.Kill would orphan.
func stopProcess(cmd *exec.Cmd, timeout time.Duration, log *slog.Logger) error {
	pid := cmd.Process.Pid
	kill := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(pid))
	if out, err := kill.CombinedOutput(); err != nil {
		log.Warn("lingvanex: taskkill failed, falling back to Kill",
			"pid", pid, "error", err, "output", string(out))
		_ = cmd.Process.Kill()
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(timeout):
		log.Warn("lingvanex: process did not exit before timeout", "pid", pid)
	}
	return nil
}
