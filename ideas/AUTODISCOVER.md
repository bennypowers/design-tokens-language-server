# Auto-Discovery Feature Design

## Status
**Proposed** - Not yet implemented

## Problem Statement

Currently, users must explicitly configure token files in their `package.json` or LSP settings:

```json
{
  "designTokensLanguageServer": {
    "tokensFiles": [
      "./tokens/colors.json",
      "./tokens/spacing.json"
    ]
  }
}
```

This creates friction in the developer experience:
- Users must know the exact file paths
- Configuration becomes stale when files are renamed/moved
- No "zero-config" experience for standard project layouts
- Increases barrier to adoption

## Proposed Solution

Add automatic token file discovery when `tokensFiles` is not explicitly configured. The server will search for common token file naming patterns in the workspace.

## Design

### Auto-Discovery Patterns

Match common naming conventions from popular design systems:

**JSON Files**:
- `**/tokens.json` - Root level tokens file
- `**/*.tokens.json` - Namespaced tokens (e.g., `colors.tokens.json`)
- `**/design-tokens.json` - Explicit "design tokens" naming

**YAML Files**:
- `**/tokens.yaml` / `**/tokens.yml`
- `**/*.tokens.yaml` / `**/*.tokens.yml`
- `**/design-tokens.yaml` / `**/design-tokens.yml`

### Activation Rules

Auto-discovery activates when:
1. No `tokensFiles` configuration provided (nil), OR
2. `tokensFiles` is explicitly set to empty array `[]`

Auto-discovery does NOT activate when:
- Explicit file paths are configured
- This preserves explicit configuration intent

### Configuration Schema

No changes to configuration schema required. Uses existing structure:

```typescript
interface ServerConfig {
  tokensFiles?: string[] | TokenFileSpec[];  // undefined/null/[] triggers auto-discovery
  prefix?: string;
  groupMarkers?: string[];
}
```

### State Management

Track auto-discovery mode as runtime state (separate from user configuration):

```go
type ServerState struct {
    AutoDiscoveryMode bool   // Runtime state
    RootPath          string // Workspace root
}

type ServerConfig struct {
    TokensFiles   []any    // User configuration
    Prefix        string
    GroupMarkers  []string
}
```

## Implementation

### Core Logic

```go
// LoadTokensFromConfig determines how to load tokens based on configuration
func (s *Server) LoadTokensFromConfig() error {
    cfg := s.GetConfig()
    state := s.GetState()

    // Explicit configuration takes precedence
    if cfg.TokensFiles != nil {
        s.setAutoDiscoveryMode(false)

        if len(cfg.TokensFiles) == 0 {
            // Empty array: try auto-discovery if workspace exists
            if state.RootPath != "" {
                s.setAutoDiscoveryMode(true)
                return s.loadTokenFilesAutoDiscover()
            }
            return nil
        }

        // Non-empty: load explicit files
        return s.loadExplicitTokenFiles()
    }

    // No configuration: auto-discover if workspace exists
    if state.RootPath != "" {
        s.tokens.Clear()
        s.setAutoDiscoveryMode(true)
        return s.loadTokenFilesAutoDiscover()
    }

    return nil
}
```

### Auto-Discovery Implementation

```go
// loadTokenFilesAutoDiscover scans workspace for token files using patterns
func (s *Server) loadTokenFilesAutoDiscover() error {
    cfg := s.GetConfig()
    state := s.GetState()

    tokenConfig := TokenFileConfig{
        RootDir:      state.RootPath,
        Patterns:     types.AutoDiscoverPatterns,
        Prefix:       cfg.Prefix,
        GroupMarkers: cfg.GroupMarkers,
    }

    return s.LoadTokenFiles(tokenConfig)
}

// AutoDiscoverPatterns in lsp/types/config.go
var AutoDiscoverPatterns = []string{
    "**/tokens.json",
    "**/*.tokens.json",
    "**/design-tokens.json",
    "**/tokens.yaml",
    "**/*.tokens.yaml",
    "**/design-tokens.yaml",
    "**/tokens.yml",
    "**/*.tokens.yml",
    "**/design-tokens.yml",
}
```

### File Scanning

```go
// LoadTokenFiles scans for token files matching glob patterns
func (s *Server) LoadTokenFiles(config TokenFileConfig) error {
    var errs []error
    foundFiles := 0

    for _, pattern := range config.Patterns {
        matches, err := filepath.Glob(filepath.Join(config.RootDir, pattern))
        if err != nil {
            errs = append(errs, fmt.Errorf("glob pattern %s failed: %w", pattern, err))
            continue
        }

        for _, match := range matches {
            // Skip directories
            info, err := os.Stat(match)
            if err != nil || info.IsDir() {
                continue
            }

            // Load token file
            opts := &TokenFileOptions{
                Prefix:       config.Prefix,
                GroupMarkers: config.GroupMarkers,
            }

            if err := s.LoadTokenFileWithOptions(match, opts); err != nil {
                errs = append(errs, fmt.Errorf("failed to load %s: %w", match, err))
                continue
            }

            foundFiles++
            fmt.Fprintf(os.Stderr, "[DTLS] Auto-discovered: %s\n", match)
        }
    }

    if foundFiles == 0 {
        fmt.Fprintf(os.Stderr, "[DTLS] No token files found via auto-discovery\n")
    } else {
        fmt.Fprintf(os.Stderr, "[DTLS] Auto-discovered %d token file(s)\n", foundFiles)
    }

    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    return nil
}
```

## Testing Strategy

### Unit Tests

```go
func TestAutoDiscovery(t *testing.T) {
    t.Run("discovers tokens.json", func(t *testing.T) {
        tmpDir := t.TempDir()
        tokensFile := filepath.Join(tmpDir, "tokens.json")
        os.WriteFile(tokensFile, []byte(`{"color": {"primary": {"$value": "#ff0000"}}}`), 0644)

        server, _ := NewServer()
        defer server.Close()

        server.SetRootPath(tmpDir)
        server.SetConfig(types.ServerConfig{
            TokensFiles: nil, // Trigger auto-discovery
        })

        err := server.LoadTokensFromConfig()
        require.NoError(t, err)

        assert.True(t, server.GetState().AutoDiscoveryMode)
        assert.Greater(t, server.TokenCount(), 0)
    })

    t.Run("discovers *.tokens.json pattern", func(t *testing.T) {
        tmpDir := t.TempDir()
        os.WriteFile(filepath.Join(tmpDir, "colors.tokens.json"),
            []byte(`{"red": {"$value": "#ff0000"}}`), 0644)
        os.WriteFile(filepath.Join(tmpDir, "spacing.tokens.json"),
            []byte(`{"small": {"$value": "8px"}}`), 0644)

        server, _ := NewServer()
        defer server.Close()

        server.SetRootPath(tmpDir)
        server.SetConfig(types.ServerConfig{
            TokensFiles: []any{}, // Empty array also triggers discovery
        })

        err := server.LoadTokensFromConfig()
        require.NoError(t, err)

        assert.Equal(t, 2, server.TokenCount())
    })

    t.Run("respects explicit configuration over discovery", func(t *testing.T) {
        tmpDir := t.TempDir()

        // Create multiple token files
        os.WriteFile(filepath.Join(tmpDir, "tokens.json"),
            []byte(`{"color": {"primary": {"$value": "#ff0000"}}}`), 0644)
        os.WriteFile(filepath.Join(tmpDir, "other.json"),
            []byte(`{"spacing": {"small": {"$value": "8px"}}}`), 0644)

        server, _ := NewServer()
        defer server.Close()

        server.SetRootPath(tmpDir)
        server.SetConfig(types.ServerConfig{
            TokensFiles: []any{filepath.Join(tmpDir, "other.json")}, // Explicit
        })

        err := server.LoadTokensFromConfig()
        require.NoError(t, err)

        // Should NOT be in auto-discovery mode
        assert.False(t, server.GetState().AutoDiscoveryMode)

        // Should only have tokens from explicit file
        assert.Equal(t, 1, server.TokenCount())
        assert.NotNil(t, server.Token("spacing-small"))
        assert.Nil(t, server.Token("color-primary")) // Not loaded
    })

    t.Run("handles YAML files", func(t *testing.T) {
        tmpDir := t.TempDir()
        os.WriteFile(filepath.Join(tmpDir, "tokens.yaml"),
            []byte("color:\n  primary:\n    $value: \"#ff0000\"\n"), 0644)

        server, _ := NewServer()
        defer server.Close()

        server.SetRootPath(tmpDir)
        server.SetConfig(types.ServerConfig{})

        err := server.LoadTokensFromConfig()
        require.NoError(t, err)

        assert.Greater(t, server.TokenCount(), 0)
    })

    t.Run("no discovery without workspace", func(t *testing.T) {
        server, _ := NewServer()
        defer server.Close()

        // No root path set
        server.SetConfig(types.ServerConfig{})

        err := server.LoadTokensFromConfig()
        require.NoError(t, err)

        assert.False(t, server.GetState().AutoDiscoveryMode)
        assert.Equal(t, 0, server.TokenCount())
    })
}
```

### Integration Tests

```go
func TestAutoDiscoveryFileWatching(t *testing.T) {
    tmpDir := t.TempDir()

    server := testutil.NewTestServer(t)
    defer server.Close()

    server.SetRootPath(tmpDir)
    server.SetConfig(types.ServerConfig{})

    // Initially no tokens
    assert.Equal(t, 0, server.TokenCount())

    // Create a token file (simulate file creation)
    tokensFile := filepath.Join(tmpDir, "tokens.json")
    os.WriteFile(tokensFile, []byte(`{"color": {"primary": {"$value": "#ff0000"}}}`), 0644)

    // Trigger file watcher reload
    workspace.DidChangeWatchedFiles(server, nil, &protocol.DidChangeWatchedFilesParams{
        Changes: []protocol.FileEvent{
            {URI: "file://" + tokensFile, Type: protocol.FileChangeTypeCreated},
        },
    })

    // Should auto-discover new file
    assert.Greater(t, server.TokenCount(), 0)
}
```

## User Experience

### Before (Explicit Configuration)
```json
{
  "designTokensLanguageServer": {
    "tokensFiles": [
      "./design/tokens/colors.json",
      "./design/tokens/spacing.json",
      "./design/tokens/typography.json"
    ]
  }
}
```

**Pain points**:
- Manual path configuration
- Breaks when files are reorganized
- Requires updating after adding new token files

### After (Zero-Config with Auto-Discovery)
```json
{
  "designTokensLanguageServer": {
    // No configuration needed! Auto-discovers:
    // - tokens.json
    // - *.tokens.json
    // - design-tokens.json
    // etc.
  }
}
```

**Benefits**:
- Works out-of-the-box for standard layouts
- Automatically finds new token files
- Less configuration maintenance

### Hybrid Approach (Explicit + Discovery)
```json
{
  "designTokensLanguageServer": {
    "tokensFiles": [
      "npm:@company/design-tokens",  // Explicit npm package
      "./custom-tokens.json"          // Explicit local file
      // Plus any auto-discovered files in workspace
    ]
  }
}
```

## Migration Path

### For Existing Users
No breaking changes - explicit configuration continues to work exactly as before.

### For New Users
Zero-config experience: just install the extension and it works if token files follow common naming patterns.

## Performance Considerations

### Glob Performance
- Use `filepath.Glob()` which is efficient for most workspaces
- Limit patterns to specific common names (not `**/*.json` wildcard)
- Cache discovered files to avoid repeated scans

### Large Workspaces
- Glob patterns are specific enough to avoid scanning entire node_modules
- Could add `.dtlsignore` file support if needed

### File Watching
- Auto-discovered files should be registered with file watchers
- Reload when token files are created/deleted/modified

## Open Questions

1. **Should auto-discovery search node_modules?**
   - Pro: Finds tokens from npm packages
   - Con: Performance impact, might find unintended files
   - **Recommendation**: No, use explicit `npm:` protocol for packages

2. **Should we support a `.dtlsignore` file?**
   - Pro: Users can exclude specific directories
   - Con: Additional complexity
   - **Recommendation**: Start without, add if requested

3. **Should auto-discovery merge with explicit files?**
   - Current design: No, explicit configuration disables auto-discovery
   - Alternative: Allow both (might cause confusion)
   - **Recommendation**: Keep current design for clarity

4. **Should we show which files were auto-discovered?**
   - Pro: Transparency, debugging
   - Con: Log noise
   - **Recommendation**: Yes, but only in verbose/debug mode

## Alternatives Considered

### Alternative 1: Always Auto-Discover
Auto-discover in all cases, even with explicit configuration.

**Rejected because**:
- Confusing to have both explicit and implicit files
- Harder to debug which tokens came from where
- Violates principle of least surprise

### Alternative 2: Opt-in Flag
Add explicit `autoDiscover: true` configuration flag.

**Rejected because**:
- More configuration (opposite of goal)
- Extra flag to learn
- `tokensFiles: []` already signals intent

### Alternative 3: Use `.dtlsrc` File
Create a dedicated config file for auto-discovery patterns.

**Rejected because**:
- Proliferation of config files
- Existing `package.json` structure works well

## Success Metrics

1. **Adoption**: % of users with zero `tokensFiles` config (using auto-discovery)
2. **Support tickets**: Reduction in "why aren't my tokens loading?" issues
3. **Time to first tokens**: Reduction in setup time for new users
4. **Performance**: No measurable performance degradation in large workspaces

## Future Enhancements

1. **Custom Patterns**: Allow users to configure additional glob patterns
   ```json
   {
     "designTokensLanguageServer": {
       "autoDiscoverPatterns": ["**/my-tokens/*.json"]
     }
   }
   ```

2. **Ignore Patterns**: Allow excluding specific paths
   ```json
   {
     "designTokensLanguageServer": {
       "autoDiscoverIgnore": ["**/node_modules/**", "**/dist/**"]
     }
   }
   ```

3. **Discovery Cache**: Cache discovery results for performance
4. **LSP Command**: `dtls.showDiscoveredFiles` command to list what was found

## References

- [Style Dictionary default config](https://amzn.github.io/style-dictionary/#/config)
- [Design Tokens Format Module](https://tr.designtokens.org/format/)
- [VS Code workspace configuration](https://code.visualstudio.com/docs/getstarted/settings)

## Implementation Checklist

- [ ] Add auto-discovery logic to `LoadTokensFromConfig()`
- [ ] Define `AutoDiscoverPatterns` constant
- [ ] Implement `loadTokenFilesAutoDiscover()` function
- [ ] Add `AutoDiscoveryMode` to `ServerState`
- [ ] Update file watcher to track auto-discovered files
- [ ] Add comprehensive unit tests
- [ ] Add integration tests with file watching
- [ ] Update documentation
- [ ] Add telemetry/logging for discovered files
- [ ] Performance test with large workspaces (1000+ files)
- [ ] Update VSCode extension README with zero-config example

## Estimated Effort

- **Design & Review**: 1 day
- **Implementation**: 2-3 days
- **Testing**: 1-2 days
- **Documentation**: 1 day
- **Total**: ~1 week

## Dependencies

- None - can be implemented independently
- Should be implemented after npm: protocol support for complete feature set
