# Design Tokens Language Server for VS Code

[![build](https://github.com/bennypowers/design-tokens-language-server/actions/workflows/build.yaml/badge.svg)](https://github.com/bennypowers/design-tokens-language-server/actions/workflows/build.yaml)
[![coverage](https://codecov.io/gh/bennypowers/design-tokens-language-server/graph/badge.svg?token=9VOMFXI5GQ)](https://codecov.io/gh/bennypowers/design-tokens-language-server)

Editor tools for working with [DTCG formatted design tokens](https://tr.designtokens.org/format/) in CSS, JSON, and YAML files.

> [!NOTE]
> This is pre-release software. If you encounter bugs or unexpected behavior, please file a detailed [issue](https://github.com/bennypowers/design-tokens-language-server/issues/new).

## Features

### Hover Documentation
Display markdown-formatted token descriptions and values when hovering over token names.

![Hover screenshot](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/hover.png)

### Intelligent Snippets
Auto-complete for design tokens - get code snippets for token values with optional fallbacks.

![Completions screenshot with menu open and ghost text of snippet](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/completions.png)

### Diagnostics
DTLS complains when your stylesheet contains a `var()` call for a design token, but the fallback value doesn't match the token's pre-defined `$value`.

![Diagnostics visible in editor](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/diagnostics.png)

### Code Actions
Toggle the presence of a token `var()` call's fallback value. Offers to fix wrong token definitions in diagnostics.

![Code actions menu open for a line](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/toggle-fallback.png)
![Code actions menu open for a diagnostic](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/autofix.png)

### Document Color
Display token color values in your source as swatches.

![Document color swatches](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/document-color.png)

### Semantic Tokens
Highlight token references inside token definition files.

![Semantic tokens highlighting legit token definitions](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/semantic-tokens.png)

### Go to Definition
Jump to the position in the tokens file where the token is defined. Can also jump from a token reference in a JSON file to the token's definition.

![Json file jump in neovim](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/goto-definition.png)

### References
Locate all references to a token in open files, whether in CSS or in the token definition JSON or YAML files.

![References](https://raw.githubusercontent.com/bennypowers/design-tokens-language-server/main/docs/references.png)

## Quick Start

1. **Install this extension** from the [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=pwrs.design-tokens-language-server-vscode)
2. **Configure token files** in your project's `package.json` (see Configuration below)
3. **Start coding** - you'll get autocomplete, hover info, and diagnostics for design tokens in CSS, JSON, and YAML files

**That's it!** The extension includes the Design Tokens Language Server binary, so no additional installation is required.

## Configuration

### Token Files

Design Tokens Language Server uses the [DTCG](https://tr.designtokens.org/format/) format for design tokens. If you have a design token file in a different format, you can use [style-dictionary](https://styledictionary.com/info/dtcg/) to convert it to DTCG.

Add a `designTokensLanguageServer` block to your project's `package.json`, with references to token files:

```json
{
  "name": "@my-design-system/elements",
  "designTokensLanguageServer": {
    "prefix": "my-ds",
    "tokensFiles": [
      "npm:@my-design-system/tokens/tokens.json",
      {
        "path": "npm:@his-design-system/tokens/tokens.json",
        "prefix": "his-ds",
        "groupMarkers": ["GROUP"]
      },
      {
        "path": "./docs/docs-site-tokens.json",
        "prefix": "docs-site"
      },
      {
        "path": "~/secret-projects/fanciest-tokens.json",
        "prefix": "shh"
      }
    ]
  }
}
```

Entries under `tokensFiles` can be either a string or an object with `path` and `prefix` properties. The `path` property can be a relative path or a deno-style npm specifier.

### Extension Settings

This extension contributes the following settings:

* `designTokensLanguageServer.tokensFiles`: List of design token files to watch for changes. Elements can be strings (paths) or objects with `path`, `prefix`, and `groupMarkers` properties.
* `designTokensLanguageServer.prefix`: Global prefix for all design tokens. Useful for namespacing your design tokens.
* `designTokensLanguageServer.groupMarkers`: List of token names which will be treated as group names (default: `["_", "@", "DEFAULT"]`).

### Token Prefixes

The DTCG format does not require a prefix for tokens, but it is recommended to use a prefix to avoid conflicts with other design systems. If your token files do not nest all of their tokens under a common prefix, you can pass one yourself in the `prefix` property.

### Group Markers

Because the DTCG format is nested, a conflict can emerge when the token file author wants to define a group of tokens, but have the group name also be a token. For example, `--token-color-red` and `--token-color-red-darker` are both valid tokens.

Design Tokens Language Server uses "group markers" to contain the token data for a group. The default group markers are `_`, `@`, and `DEFAULT`.

For example, if you have a token file with the following tokens:

```json
{
  "color": {
    "red": {
      "GROUP": {
        "$value": "#FF0000",
        "$description": "Red color",
        "darker": {
          "$value": "#AA0000",
          "$description": "Darker red color"
        }
      }
    }
  }
}
```

Then set the `groupMarkers` property to `["GROUP"]` in your `package.json`:

```json
"designTokensLanguageServer": {
  "prefix": "my-ds",
  "groupMarkers": ["GROUP"]
}
```

## Supported File Types

- **CSS** (`.css`) - Full design token support with `var()` functions
- **JSON** (`.json`) - Token definition files
- **YAML** (`.yaml`, `.yml`) - Token definition files

## Troubleshooting

### No Completions or Hover Information

1. **Verify your project has a `designTokensLanguageServer` block** in `package.json` with valid token file paths
2. **Check the Output panel** (View → Output → "Design Tokens Language Server") for error messages
3. **Ensure token files are valid DTCG format** - use a JSON validator to check syntax
4. **Check file paths** - relative paths are resolved from the workspace root

### Language Server Won't Start

1. **Check the Output panel** for error messages
2. **Try restarting VS Code**
3. **Verify the extension is enabled** in the Extensions panel
4. **Report issues** on the [GitHub repository](https://github.com/bennypowers/design-tokens-language-server/issues)

### Debugging

The server logs to `~/.local/state/design-tokens-language-server/dtls.log` by default. You can view these logs for detailed debugging information:

```bash
tail -f ~/.local/state/design-tokens-language-server/dtls.log
```

## Related Links

- [Design Tokens Language Server GitHub](https://github.com/bennypowers/design-tokens-language-server)
- [DTCG Format Specification](https://tr.designtokens.org/format/)
- [Style Dictionary](https://styledictionary.com/info/dtcg/)

## License

GPL-3.0-only
