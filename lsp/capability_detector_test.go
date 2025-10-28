package lsp

import (
	"encoding/json"
	"testing"
)

func TestDetectPullDiagnosticsSupport(t *testing.T) {
	tests := []struct {
		name     string
		rawJSON  string
		expected bool
	}{
		{
			name: "LSP 3.17 client with diagnostic capability",
			rawJSON: `{
				"capabilities": {
					"textDocument": {
						"diagnostic": {
							"dynamicRegistration": false
						}
					}
				}
			}`,
			expected: true,
		},
		{
			name: "LSP 3.17 client with empty diagnostic object",
			rawJSON: `{
				"capabilities": {
					"textDocument": {
						"diagnostic": {}
					}
				}
			}`,
			expected: true,
		},
		{
			name: "LSP 3.16 client without diagnostic field",
			rawJSON: `{
				"capabilities": {
					"textDocument": {
						"completion": {
							"completionItem": {
								"snippetSupport": true
							}
						},
						"hover": {
							"contentFormat": ["markdown", "plaintext"]
						}
					}
				}
			}`,
			expected: false,
		},
		{
			name: "Client with null textDocument capabilities",
			rawJSON: `{
				"capabilities": {
					"textDocument": null
				}
			}`,
			expected: false,
		},
		{
			name: "Client with missing textDocument capabilities",
			rawJSON: `{
				"capabilities": {
					"workspace": {
						"workspaceFolders": true
					}
				}
			}`,
			expected: false,
		},
		{
			name: "Empty capabilities object",
			rawJSON: `{
				"capabilities": {}
			}`,
			expected: false,
		},
		{
			name: "Malformed JSON",
			rawJSON: `{
				"capabilities": {
					"textDocument": {
						"diagnostic"
					}
				}
			}`,
			expected: false,
		},
		{
			name:     "Empty JSON",
			rawJSON:  `{}`,
			expected: false,
		},
		{
			name:     "Invalid JSON",
			rawJSON:  `not valid json`,
			expected: false,
		},
		{
			name: "Real-world VSCode LSP 3.17 capabilities",
			rawJSON: `{
				"processId": 12345,
				"clientInfo": {
					"name": "Visual Studio Code",
					"version": "1.75.0"
				},
				"capabilities": {
					"workspace": {
						"applyEdit": true,
						"workspaceFolders": true
					},
					"textDocument": {
						"synchronization": {
							"dynamicRegistration": true,
							"willSave": true,
							"willSaveWaitUntil": true,
							"didSave": true
						},
						"completion": {
							"dynamicRegistration": true,
							"completionItem": {
								"snippetSupport": true
							}
						},
						"hover": {
							"dynamicRegistration": true,
							"contentFormat": ["markdown", "plaintext"]
						},
						"diagnostic": {
							"dynamicRegistration": true,
							"relatedDocumentSupport": false
						}
					}
				}
			}`,
			expected: true,
		},
		{
			name: "Real-world Neovim LSP 3.16 capabilities (no diagnostic)",
			rawJSON: `{
				"processId": 54321,
				"clientInfo": {
					"name": "Neovim",
					"version": "0.7.0"
				},
				"capabilities": {
					"workspace": {
						"applyEdit": true,
						"workspaceFolders": true
					},
					"textDocument": {
						"synchronization": {
							"dynamicRegistration": false,
							"willSave": true,
							"didSave": true
						},
						"completion": {
							"completionItem": {
								"snippetSupport": true
							}
						},
						"hover": {
							"contentFormat": ["markdown", "plaintext"]
						}
					}
				}
			}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawParams := json.RawMessage(tt.rawJSON)
			result := DetectPullDiagnosticsSupport(rawParams)

			if result != tt.expected {
				t.Errorf("DetectPullDiagnosticsSupport() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
