package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/icalvete/augmented-brain/internal/runlimit"
	"github.com/icalvete/augmented-brain/internal/version"
)

// ServerLimit is how long one conversation with Codex's app server may last: it reads
// and writes local files only.
const ServerLimit = 30 * time.Second

// appServer talks JSON-RPC to `codex app-server` over its standard input and output,
// one JSON message per line.
type appServer struct {
	cancel context.CancelFunc
	cmd    *exec.Cmd
	in     io.WriteCloser
	out    *bufio.Reader
	next   int
	stderr *strings.Builder
	mu     sync.Mutex
}

// Serve starts Codex's app server and says hello to it, within ServerLimit.
func Serve() (Server, error) {
	if !Available() {
		return nil, ErrNoCodex
	}
	ctx, cancel := context.WithTimeout(context.Background(), ServerLimit)
	cmd := runlimit.Command(ctx, "codex", "app-server")
	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("starting codex app-server: %v", err)
	}
	s := &appServer{cancel: cancel, cmd: cmd, in: in, out: bufio.NewReaderSize(out, 1<<20), stderr: &stderr}
	if err := s.Call("initialize", map[string]any{"clientInfo": map[string]any{"name": "augmented-brain", "version": "1"}}, nil); err != nil {
		s.Close()
		return nil, err
	}
	if err := s.send(map[string]any{"jsonrpc": "2.0", "method": "initialized"}); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *appServer) send(msg any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = s.in.Write(append(b, '\n'))
	return err
}

// Call sends a request and waits for its answer, skipping the notifications between.
func (s *appServer) Call(method string, params, result any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	id := s.next
	if err := s.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return fmt.Errorf("codex app-server %s: %v", method, err)
	}
	for {
		line, err := s.out.ReadBytes('\n')
		if err != nil {
			return fmt.Errorf("codex app-server %s: no answer (%v) %s", method, err, strings.TrimSpace(s.stderr.String()))
		}
		var msg struct {
			ID     *int            `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(line, &msg) != nil || msg.ID == nil || *msg.ID != id {
			continue
		}
		if msg.Error != nil {
			return fmt.Errorf("codex app-server %s: %s", method, msg.Error.Message)
		}
		if result == nil {
			return nil
		}
		return json.Unmarshal(msg.Result, result)
	}
}

// Close ends the app server.
func (s *appServer) Close() error {
	s.in.Close()
	s.cancel()
	s.cmd.Wait()
	return nil
}

// RunLimit is how long Exec waits for a codex command (adding a marketplace or a plugin
// copies local files).
const RunLimit = 2 * time.Minute

// Exec runs a real command within RunLimit.
func Exec(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), RunLimit)
	defer cancel()
	out, err := runlimit.Command(ctx, name, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		err = fmt.Errorf("no answer from %s in %s: %w", name, RunLimit, err)
	}
	return string(out), err
}

// Default is the Env of this program and computer, for the configuration folder cfgDir.
func Default(cfgDir string) (Env, error) {
	exe, err := os.Executable()
	if err != nil {
		return Env{}, err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	cwd, err := os.UserHomeDir()
	if err != nil {
		cwd, _ = os.Getwd()
	}
	return Env{Run: Exec, Serve: Serve, Dir: Dir(cfgDir), Exe: exe,
		Version: version.Version, GOOS: runtime.GOOS, Cwd: cwd}, nil
}
