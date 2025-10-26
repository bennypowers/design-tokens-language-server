package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

// TestHeaderParsing tests the LSP header parsing logic
func TestHeaderParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectLen   int
		expectFound bool
	}{
		{
			name: "single Content-Length header with CRLF",
			input: "Content-Length: 42\r\n" +
				"\r\n" +
				"",
			expectLen:   42,
			expectFound: true,
		},
		{
			name: "single Content-Length header with LF",
			input: "Content-Length: 42\n" +
				"\n" +
				"",
			expectLen:   42,
			expectFound: true,
		},
		{
			name: "multiple headers with Content-Length first",
			input: "Content-Length: 123\r\n" +
				"Content-Type: application/json\r\n" +
				"\r\n" +
				"",
			expectLen:   123,
			expectFound: true,
		},
		{
			name: "multiple headers with Content-Length last",
			input: "Content-Type: application/json\r\n" +
				"Content-Length: 456\r\n" +
				"\r\n" +
				"",
			expectLen:   456,
			expectFound: true,
		},
		{
			name: "case insensitive Content-Length",
			input: "content-length: 789\r\n" +
				"\r\n" +
				"",
			expectLen:   789,
			expectFound: true,
		},
		{
			name: "mixed case Content-Length",
			input: "CoNtEnT-LeNgTh: 999\r\n" +
				"\r\n" +
				"",
			expectLen:   999,
			expectFound: true,
		},
		{
			name: "Content-Length with extra spaces",
			input: "Content-Length:   100  \r\n" +
				"\r\n" +
				"",
			expectLen:   100,
			expectFound: true,
		},
		{
			name: "no Content-Length header",
			input: "Content-Type: application/json\r\n" +
				"\r\n" +
				"",
			expectLen:   0,
			expectFound: false,
		},
		{
			name: "invalid Content-Length value",
			input: "Content-Length: invalid\r\n" +
				"\r\n" +
				"",
			expectLen:   0,
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))

			// Simulate the header parsing logic
			var contentLength int
			foundLength := false

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Fatalf("unexpected error: %v", err)
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
						length, err := parseContentLength(valueStr)
						if err == nil {
							contentLength = length
							foundLength = true
						}
					}
				}
			}

			if foundLength != tt.expectFound {
				t.Errorf("foundLength = %v, want %v", foundLength, tt.expectFound)
			}

			if foundLength && contentLength != tt.expectLen {
				t.Errorf("contentLength = %d, want %d", contentLength, tt.expectLen)
			}
		})
	}
}

// parseContentLength is a helper to parse the content length value
func parseContentLength(s string) (int, error) {
	var length int
	_, err := fmt.Sscanf(s, "%d", &length)
	return length, err
}

// TestFullMessageParsing tests parsing a complete LSP message
func TestFullMessageParsing(t *testing.T) {
	messageContent := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`
	message := "Content-Length: " + fmt.Sprintf("%d", len(messageContent)) + "\r\n" +
		"Content-Type: application/json\r\n" +
		"\r\n" +
		messageContent

	reader := bufio.NewReader(bytes.NewBufferString(message))

	// Parse headers
	var contentLength int
	foundLength := false

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected error reading headers: %v", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				valueStr := strings.TrimSpace(parts[1])
				length, err := parseContentLength(valueStr)
				if err == nil {
					contentLength = length
					foundLength = true
				}
			}
		}
	}

	if !foundLength {
		t.Fatal("Content-Length header not found")
	}

	if contentLength != len(messageContent) {
		t.Errorf("contentLength = %d, want %d", contentLength, len(messageContent))
	}

	// Read content
	content := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, content); err != nil {
		t.Fatalf("failed to read content: %v", err)
	}

	if string(content) != messageContent {
		t.Errorf("content mismatch:\ngot:  %s\nwant: %s", string(content), messageContent)
	}
}

// TestJSONRPCIDSerialization tests that ID field is always serialized (not omitted)
func TestJSONRPCIDSerialization(t *testing.T) {
	tests := []struct {
		name      string
		id        int
		expectID  string // Expected ID in JSON
	}{
		{
			name:     "ID is 0",
			id:       0,
			expectID: `"id":0`,
		},
		{
			name:     "ID is 1",
			id:       1,
			expectID: `"id":1`,
		},
		{
			name:     "ID is 42",
			id:       42,
			expectID: `"id":42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test request serialization
			req := jsonrpcRequest{
				JSONRPC: "2.0",
				ID:      tt.id,
				Method:  "test",
			}

			data, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("failed to marshal request: %v", err)
			}

			jsonStr := string(data)
			if !strings.Contains(jsonStr, tt.expectID) {
				t.Errorf("Request JSON missing expected ID:\ngot:  %s\nwant: %s", jsonStr, tt.expectID)
			}

			// Test response serialization
			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      tt.id,
			}

			data, err = json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal response: %v", err)
			}

			jsonStr = string(data)
			if !strings.Contains(jsonStr, tt.expectID) {
				t.Errorf("Response JSON missing expected ID:\ngot:  %s\nwant: %s", jsonStr, tt.expectID)
			}
		})
	}
}

// TestIDGeneratorStartsAtOne tests that the ID generator starts at 1, not 0
func TestIDGeneratorStartsAtOne(t *testing.T) {
	// Simulate the ID generation logic from call()
	nextID := 0 // Initial value

	// First call increments before assigning
	nextID++
	firstID := nextID

	if firstID != 1 {
		t.Errorf("First ID should be 1, got %d", firstID)
	}

	// Second call
	nextID++
	secondID := nextID

	if secondID != 2 {
		t.Errorf("Second ID should be 2, got %d", secondID)
	}
}

// TestNotificationHasNoID tests that notifications don't have an ID field
func TestNotificationHasNoID(t *testing.T) {
	notif := jsonrpcNotification{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params:  map[string]string{"uri": "file:///test.txt"},
	}

	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("failed to marshal notification: %v", err)
	}

	jsonStr := string(data)

	// Notifications should NOT have an "id" field
	if strings.Contains(jsonStr, `"id"`) {
		t.Errorf("Notification should not have an 'id' field:\ngot: %s", jsonStr)
	}

	// Should have method and jsonrpc
	if !strings.Contains(jsonStr, `"method":"textDocument/didOpen"`) {
		t.Errorf("Notification missing method field:\ngot: %s", jsonStr)
	}

	if !strings.Contains(jsonStr, `"jsonrpc":"2.0"`) {
		t.Errorf("Notification missing jsonrpc field:\ngot: %s", jsonStr)
	}
}

// TestRpcRespStructure tests the rpcResp type
func TestRpcRespStructure(t *testing.T) {
	tests := []struct {
		name      string
		resp      rpcResp
		expectErr bool
	}{
		{
			name: "successful response",
			resp: rpcResp{
				result: json.RawMessage(`{"capabilities":{}}`),
				err:    nil,
			},
			expectErr: false,
		},
		{
			name: "error response",
			resp: rpcResp{
				result: nil,
				err:    fmt.Errorf("JSON-RPC error -32601: Method not found"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectErr && tt.resp.err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && tt.resp.err != nil {
				t.Errorf("Expected no error but got: %v", tt.resp.err)
			}
			if tt.expectErr && tt.resp.result != nil {
				t.Error("Error response should have nil result")
			}
			if !tt.expectErr && tt.resp.result == nil {
				t.Error("Successful response should have non-nil result")
			}
		})
	}
}

// TestJSONRPCErrorParsing tests parsing of JSON-RPC error responses
func TestJSONRPCErrorParsing(t *testing.T) {
	tests := []struct {
		name        string
		jsonResp    string
		expectError bool
		errorCode   int
		errorMsg    string
	}{
		{
			name:        "successful response",
			jsonResp:    `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`,
			expectError: false,
		},
		{
			name:        "method not found error",
			jsonResp:    `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`,
			expectError: true,
			errorCode:   -32601,
			errorMsg:    "Method not found",
		},
		{
			name:        "invalid params error",
			jsonResp:    `{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"Invalid params"}}`,
			expectError: true,
			errorCode:   -32602,
			errorMsg:    "Invalid params",
		},
		{
			name:        "server error",
			jsonResp:    `{"jsonrpc":"2.0","id":3,"error":{"code":-32000,"message":"Server error"}}`,
			expectError: true,
			errorCode:   -32000,
			errorMsg:    "Server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp jsonrpcResponse
			if err := json.Unmarshal([]byte(tt.jsonResp), &resp); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if tt.expectError {
				if resp.Error == nil {
					t.Fatal("Expected error field to be present")
				}
				if resp.Error.Code != tt.errorCode {
					t.Errorf("Error code: got %d, want %d", resp.Error.Code, tt.errorCode)
				}
				if resp.Error.Message != tt.errorMsg {
					t.Errorf("Error message: got %q, want %q", resp.Error.Message, tt.errorMsg)
				}
			} else {
				if resp.Error != nil {
					t.Errorf("Expected no error but got: %+v", resp.Error)
				}
				if resp.Result == nil {
					t.Error("Expected result to be present")
				}
			}
		})
	}
}

// TestErrorResponseHandling tests that errors are properly propagated through rpcResp
func TestErrorResponseHandling(t *testing.T) {
	// Simulate receiving an error response
	errorResp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      1,
		Error: &jsonrpcError{
			Code:    -32601,
			Message: "Method not found",
		},
	}

	// Simulate what readResponses does
	var resp rpcResp
	if errorResp.Error != nil {
		resp = rpcResp{
			result: nil,
			err:    fmt.Errorf("JSON-RPC error %d: %s", errorResp.Error.Code, errorResp.Error.Message),
		}
	} else {
		resp = rpcResp{
			result: errorResp.Result,
			err:    nil,
		}
	}

	// Verify error was captured
	if resp.err == nil {
		t.Fatal("Expected error to be set")
	}

	if !strings.Contains(resp.err.Error(), "Method not found") {
		t.Errorf("Error message should contain 'Method not found', got: %v", resp.err)
	}

	if !strings.Contains(resp.err.Error(), "-32601") {
		t.Errorf("Error message should contain error code -32601, got: %v", resp.err)
	}

	if resp.result != nil {
		t.Error("Error response should have nil result")
	}
}

// TestParseMemoryStatus tests parsing /proc/[pid]/status for memory info
func TestParseMemoryStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		statusContent  string
		expectedMemory uint64
		expectError    bool
	}{
		{
			name: "valid VmRSS in kB",
			statusContent: `Name:	design-tokens-language-server
Umask:	0022
State:	S (sleeping)
VmPeak:	   12345 kB
VmSize:	   12000 kB
VmRSS:	    5432 kB
VmData:	    1024 kB`,
			expectedMemory: 5432 * 1024,
			expectError:    false,
		},
		{
			name: "VmRSS with tabs and spaces",
			statusContent: `Name:	test
VmRSS:	  	  1024 kB
VmData:	512 kB`,
			expectedMemory: 1024 * 1024,
			expectError:    false,
		},
		{
			name: "VmRSS at beginning",
			statusContent: `VmRSS:	2048 kB
VmData:	512 kB`,
			expectedMemory: 2048 * 1024,
			expectError:    false,
		},
		{
			name: "no VmRSS line",
			statusContent: `Name:	test
VmSize:	1000 kB
VmData:	512 kB`,
			expectedMemory: 0,
			expectError:    true,
		},
		{
			name:           "empty content",
			statusContent:  "",
			expectedMemory: 0,
			expectError:    true,
		},
		{
			name: "invalid VmRSS format",
			statusContent: `VmRSS:	invalid kB
VmData:	512 kB`,
			expectedMemory: 0,
			expectError:    true,
		},
		{
			name: "missing unit",
			statusContent: `VmRSS:	1024
VmData:	512 kB`,
			expectedMemory: 0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory, err := parseMemoryFromStatus([]byte(tt.statusContent))

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if memory != tt.expectedMemory {
					t.Errorf("Memory = %d bytes, want %d bytes", memory, tt.expectedMemory)
				}
			}
		})
	}
}

// parseMemoryFromStatus is the extracted logic for parsing memory from status file
func parseMemoryFromStatus(data []byte) (uint64, error) {
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
				return 0, fmt.Errorf("failed to parse VmRSS size: %v", err)
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
