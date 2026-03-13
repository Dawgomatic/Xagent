package mcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// StdioTransport communicates with an MCP server via stdin/stdout of a subprocess.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	mu     sync.Mutex
}

// NewStdioTransport spawns a local MCP server process and connects via stdio.
func NewStdioTransport(command string, args []string, env []string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)
	if len(env) > 0 {
		cmd.Env = append(cmd.Environ(), env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mcp server: %w", err)
	}

	return &StdioTransport{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReader(stdout),
	}, nil
}

// Send writes a JSON-RPC message to the server's stdin, newline-delimited.
func (t *StdioTransport) Send(ctx context.Context, data []byte) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	line := append(data, '\n')
	_, err := t.stdin.Write(line)
	return err
}

// Receive reads a newline-delimited JSON-RPC message from the server's stdout.
func (t *StdioTransport) Receive(ctx context.Context) ([]byte, error) {
	// Use a channel to make the blocking read cancellable
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		line, err := t.reader.ReadBytes('\n')
		ch <- result{line, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.data, r.err
	}
}

// Close kills the subprocess and closes pipes.
func (t *StdioTransport) Close() error {
	t.stdin.Close()
	if t.cmd.Process != nil {
		t.cmd.Process.Kill()
	}
	return t.cmd.Wait()
}
