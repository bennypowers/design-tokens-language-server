package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// LSPClient represents a connection to an LSP server
type LSPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	responses map[int]chan json.RawMessage
	mu        sync.Mutex
	nextID    int
	reader    *bufio.Reader
	ctx       context.Context
	cancel    context.CancelFunc
}

type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonrpcNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// NewLSPClient creates a new LSP client and starts the server
func NewLSPClient(serverCmd string) (*LSPClient, error) {
	parts := strings.Fields(serverCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty server command")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	client := &LSPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		responses: make(map[int]chan json.RawMessage),
		reader:    bufio.NewReader(stdout),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start reading responses in background
	go client.readResponses()

	// Drain stderr
	go io.Copy(io.Discard, stderr)

	return client, nil
}

func (c *LSPClient) readResponses() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Read Content-Length header
		header, err := c.reader.ReadString('\n')
		if err != nil {
			return
		}

		var length int
		if _, err := fmt.Sscanf(header, "Content-Length: %d\r\n", &length); err != nil {
			continue
		}

		// Read empty line
		c.reader.ReadString('\n')

		// Read content
		content := make([]byte, length)
		if _, err := io.ReadFull(c.reader, content); err != nil {
			return
		}

		// Parse response
		var resp jsonrpcResponse
		if err := json.Unmarshal(content, &resp); err != nil {
			// Might be a notification, ignore
			continue
		}

		// Deliver to waiting goroutine
		c.mu.Lock()
		if ch, ok := c.responses[resp.ID]; ok {
			ch <- resp.Result
			close(ch)
			delete(c.responses, resp.ID)
		}
		c.mu.Unlock()
	}
}

func (c *LSPClient) call(method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	respChan := make(chan json.RawMessage, 1)
	c.responses[id] = respChan
	c.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	if _, err := c.stdin.Write([]byte(msg)); err != nil {
		return nil, err
	}

	// Wait for response with timeout
	select {
	case result := <-respChan:
		return result, nil
	case <-time.After(5 * time.Second):
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (c *LSPClient) notify(method string, params interface{}) error {
	req := jsonrpcNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	_, err = c.stdin.Write([]byte(msg))
	return err
}

// Initialize sends the initialize request
func (c *LSPClient) Initialize() error {
	params := map[string]interface{}{
		"processId": nil,
		"rootUri":   "file:///test",
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"hover": map[string]interface{}{
					"contentFormat": []string{"markdown", "plaintext"},
				},
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": true,
					},
				},
			},
		},
	}

	_, err := c.call("initialize", params)
	if err != nil {
		return err
	}

	// Send initialized notification
	return c.notify("initialized", map[string]interface{}{})
}

// DidOpen sends a textDocument/didOpen notification
func (c *LSPClient) DidOpen(uri, languageID, text string) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageID,
			"version":    1,
			"text":       text,
		},
	}
	return c.notify("textDocument/didOpen", params)
}

// Hover sends a textDocument/hover request
func (c *LSPClient) Hover(uri string, line, character int) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": character,
		},
	}
	_, err := c.call("textDocument/hover", params)
	return err
}

// Completion sends a textDocument/completion request
func (c *LSPClient) Completion(uri string, line, character int) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": character,
		},
	}
	_, err := c.call("textDocument/completion", params)
	return err
}

// Diagnostic sends a textDocument/diagnostic request
func (c *LSPClient) Diagnostic(uri string) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	}
	_, err := c.call("textDocument/diagnostic", params)
	return err
}

// Definition sends a textDocument/definition request
func (c *LSPClient) Definition(uri string, line, character int) error {
	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
		"position": map[string]interface{}{
			"line":      line,
			"character": character,
		},
	}
	_, err := c.call("textDocument/definition", params)
	return err
}

// Close shuts down the LSP server
func (c *LSPClient) Close() error {
	c.cancel()
	c.stdin.Close()
	return c.cmd.Wait()
}

// GetProcessMemory attempts to get the process memory usage (Linux-specific)
func (c *LSPClient) GetProcessMemory() (uint64, error) {
	if c.cmd.Process == nil {
		return 0, fmt.Errorf("process not started")
	}

	// Try to read from /proc/[pid]/status
	statusPath := fmt.Sprintf("/proc/%d/status", c.cmd.Process.Pid)
	data, err := exec.Command("cat", statusPath).Output()
	if err != nil {
		// Fallback to approximate value
		return 50 * 1024 * 1024, nil // 50MB estimate
	}

	// Parse VmRSS (Resident Set Size)
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("VmRSS:")) {
			var size uint64
			var unit string
			fmt.Sscanf(string(line), "VmRSS: %d %s", &size, &unit)
			if unit == "kB" {
				return size * 1024, nil
			}
		}
	}

	return 0, fmt.Errorf("could not parse memory usage")
}
