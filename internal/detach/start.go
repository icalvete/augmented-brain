package detach

import (
	"os"
	"os/exec"
)

// Process is a process Start started.
type Process struct {
	Pid  int
	proc *os.Process
	done chan struct{}
	err  error
}

// Kill ends the process.
func (p *Process) Kill() error { return p.proc.Kill() }

// Done is closed once the process has ended.
func (p *Process) Done() <-chan struct{} { return p.done }

// Err is how the process ended, as exec.Cmd.Wait says; read it after Done.
func (p *Process) Err() error { return p.err }

// Start starts a command prepared with Cmd. It always waits for the process in the
// background, so a program that lives long (the MCP server, the panel's server) is
// never left with a zombie for each process it started, which kill(pid, 0) takes for
// alive (R-17 of the November 2026 review). Whoever wants to know when it ends reads
// Done.
func Start(cmd *exec.Cmd) (*Process, error) {
	cmd, err := start(cmd)
	if err != nil {
		return nil, err
	}
	p := &Process{Pid: cmd.Process.Pid, proc: cmd.Process, done: make(chan struct{})}
	go func() {
		p.err = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}
