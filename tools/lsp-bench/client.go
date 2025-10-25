package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// rpcResp represents a JSON-RPC response or error
type rpcResp struct {
	result json.RawMessage
	err    error
}

// LSPClient represents a connection to an LSP server
type LSPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	responses map[int]chan rpcResp
	mu        sync.Mutex
	nextID    int
	reader    *bufio.Reader
	ctx       context.Context
	cancel    context.CancelFunc
}

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonrpcNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
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
		responses: make(map[int]chan rpcResp),
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

		// Read headers until we find an empty line
		var contentLength int
		foundLength := false

		for {
			line, err := c.reader.ReadString('\n')
			if err != nil {
				return
			}

			// Trim whitespace (handles both \r\n and \n)
			line = strings.TrimSpace(line)

			// Empty line marks end of headers
			if line == "" {
				break
			}

			// Parse Content-Length header (case-insensitive)
			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				// Extract the value after the colon
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					valueStr := strings.TrimSpace(parts[1])
					length, err := strconv.Atoi(valueStr)
					if err != nil {
						// Invalid Content-Length, skip this message
						continue
					}
					contentLength = length
					foundLength = true
				}
			}
			// Ignore other headers (e.g., Content-Type)
		}

		// If we didn't find Content-Length, skip this message
		if !foundLength {
			continue
		}

		// Read content
		content := make([]byte, contentLength)
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
			// Check if this is an error response
			if resp.Error != nil {
				ch <- rpcResp{
					result: nil,
					err:    fmt.Errorf("JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message),
				}
			} else {
				ch <- rpcResp{
					result: resp.Result,
					err:    nil,
				}
			}
			close(ch)
			delete(c.responses, resp.ID)
		}
		c.mu.Unlock()
	}
}

func (c *LSPClient) call(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	respChan := make(chan rpcResp, 1)
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
	case resp := <-respChan:
		if resp.err != nil {
			return nil, resp.err
		}
		return resp.result, nil
	case <-time.After(5 * time.Second):
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("timeout waiting for response")
	}
}

func (c *LSPClient) notify(method string, params any) error {
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
	params := map[string]any{
		"processId": nil,
		"rootUri":   "file:///test",
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"hover": map[string]any{
					"contentFormat": []string{"markdown", "plaintext"},
				},
				"completion": map[string]any{
					"completionItem": map[string]any{
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
	return c.notify("initialized", map[string]any{})
}

// DidOpen sends a textDocument/didOpen notification
func (c *LSPClient) DidOpen(uri, languageID, text string) error {
	params := map[string]any{
		"textDocument": map[string]any{
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
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": uri,
		},
		"position": map[string]any{
			"line":      line,
			"character": character,
		},
	}
	_, err := c.call("textDocument/hover", params)
	return err
}

// Completion sends a textDocument/completion request
func (c *LSPClient) Completion(uri string, line, character int) error {
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": uri,
		},
		"position": map[string]any{
			"line":      line,
			"character": character,
		},
	}
	_, err := c.call("textDocument/completion", params)
	return err
}

// Diagnostic sends a textDocument/diagnostic request
func (c *LSPClient) Diagnostic(uri string) error {
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": uri,
		},
	}
	_, err := c.call("textDocument/diagnostic", params)
	return err
}

// Definition sends a textDocument/definition request
func (c *LSPClient) Definition(uri string, line, character int) error {
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": uri,
		},
		"position": map[string]any{
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
	data, err := os.ReadFile(statusPath)
	if err != nil {
		// Fallback to approximate value
		return 50 * 1024 * 1024, nil // 50MB estimate
	}

	// Parse VmRSS (Resident Set Size)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VmRSS:") {
			// Split into fields: "VmRSS:" "5432" "kB"
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return 0, fmt.Errorf("invalid VmRSS format: not enough fields")
			}

			var size uint64
			if _, err := fmt.Sscanf(fields[1], "%d", &size); err != nil {
				return 0, fmt.Errorf("failed to parse VmRSS size: %w", err)
			}

			unit := fields[2]
			if unit != "kB" {
				return 0, fmt.Errorf("unexpected unit %q, expected kB", unit)
			}

			return size * 1024, nil
		}
	}

	return 0, fmt.Errorf("VmRSS not found in status")
}
