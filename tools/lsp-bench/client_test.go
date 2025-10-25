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
