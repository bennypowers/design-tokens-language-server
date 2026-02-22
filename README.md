# 🎨 Design Tokens Language Server 🪙

[![build][build]][buildyaml]
[![coverage][coverage]][coveragereport]

Editor tools for working with <abbr title="design tokens community
  group">[DTCG][dtcg]</abbr> formatted design tokens in CSS, HTML,
JavaScript/TypeScript (lit-element), and for token definitions in
JSON or YAML files.

Supports both **Editor's Draft** and **DTCG 2025.10 stable** schema versions,
including multi-schema workspaces where different token files use different schema versions.

> [!NOTE]
> This is pre-release software. If you encounter bugs or unexpected behavior,
> please file a detailed [issue][issue].

## ✨ Features

### ⛸️ Hover Docs

Display markdown-formatted token descriptions and value when hovering over token
names.

![Hover screenshot](./docs/hover.png)

### ✂️ Snippets

Auto complete for design tokens - get code snippets for token values with
optional fallbacks.

![Completions screenshot with menu open and ghost text of snippet](./docs/completions.png)

### 🥼 Diagnostics

DTLS complains when your stylesheet contains a `var()` call for a design token,
but the fallback value doesn't match the token's pre-defined `$value`.

![Diagnostics visible in editor](./docs/diagnostics.png)

### 🦸 Code actions

Toggle the presence of a token `var()` call's fallback value. Offers to fix
wrong token definitions in Diagnostics.

![Code actions menu open for a line](./docs/toggle-fallback.png)
![Code actions menu open for a diagnostic](./docs/autofix.png)

### 🎨 Document Color

Display token color values in your source, e.g. as swatches.

![Document color swatches](./docs/document-color.png)

### 👂 Semantic Tokens

Highlight token references inside token definition files.

![Semantic tokens highlighting legit token definitions](./docs/semantic-tokens.png)

### 🪪 Go to Definition

Jump to the position in the tokens file where the token is defined. Can also
jump from a token reference in a JSON file to the token's definition.

![Json file jump in neovim](./docs/goto-definition.png)

Go to definition in a split window using Neovim's [`<C-w C-]>` binding][cwcdash],
which defers to LSP methods when they're available.

### 🗺️ References

Locate all references to a token in open files, whether in CSS or in the token
definition JSON or YAML files

![References](./docs/references.png)

## 📚 Schema Support

DTLS supports both DTCG schema versions:

- **[Editor's Draft][editorsdraft]** - Original DTCG format
  - String color values (hex, rgb, hsl, named colors)
  - Curly brace references: `{color.brand.primary}`
  - Group markers for root tokens: `_`, `@`, `DEFAULT`

- **[2025.10 Stable][202510stable]** - Latest stable specification
  - Structured color values with 14 color spaces (sRGB, oklch, display-p3, etc.)
  - JSON Pointer references: `$ref: "#/color/brand/primary"`
  - Group inheritance: `$extends: "#/baseColors"`
  - Standardized `$root` token for root-level tokens
  - All draft features (backward compatible)

### Multi-Schema Workspaces

DTLS can load multiple token files with different schema versions simultaneously:

```json
{
  "designTokensLanguageServer": {
    "tokensFiles": [
      "legacy/draft-tokens.json",        // Editor's Draft
      "design-system/tokens.json"         // 2025.10 (auto-detected from $schema)
    ]
  }
}
```

Schema version detection priority:
1. `$schema` field in the token file (recommended)
2. Per-file `schemaVersion` config in `package.json`
3. Duck-typing based on features (structured colors, `$ref`, `$extends`)
4. Defaults to Editor's Draft for ambiguous files

For more information, see [Schema Versioning Guide](./docs/SCHEMA_VERSIONING.md).

## 🧰 Usage

Download the latest release for your platform and place the binary in your
`$PATH`, renaming it to `design-tokens-language-server`.

### 🛻 Installation

#### Neovim

Using native Neovim LSP (see [`:help lsp`][neovimlspdocs] for more
info):

Create a file like `~/.config/nvim/lsp/design_tokens_ls.lua`:

```lua
---@type vim.lsp.ClientConfig
return {
  cmd = { 'design-tokens-language-server' },
  root_markers = { '.git', 'package.json' },
  filetypes = { 'css', 'html', 'javascript', 'javascriptreact', 'typescript', 'typescriptreact', 'json', 'yaml' },
  settings = {
    dtls = {
      -- Optional: configure global token files
      tokensFiles = {
        {
          path = "~/path/to/tokens.json",
          prefix = "my-ds",
        },
      },
      -- Optional: configure group markers
      groupMarkers = { '_', '@', 'DEFAULT' },
    }
  },
  -- Optional: enable document color support
  on_attach = function(client, bufnr)
    if vim.lsp.document_color then
      vim.lsp.document_color.enable(true, bufnr, {
        style = 'virtual'
      })
    end
  end,
}
```

Then configure your LSP setup to load configs from `~/.config/nvim/lsp/`. This allows
you to manage each language server in its own file. See the
[neovim docs][neovimlspdocs] for details on setting up native LSP clients.

> [!TIP]
> If your tokens are in `node_modules` (e.g., `npm:@my-ds/tokens/tokens.json`),
> the default `root_markers` may find the wrong `package.json`. The example
> above uses `{ '.git', 'package.json' }` which prefers `.git` over nested
> `package.json` files - this works for most git-based projects.
>
> For non-git projects or monorepos, use a custom `root_dir` that explicitly
> skips `node_modules`:
>
> ```lua
> root_dir = function(bufnr, on_dir)
>   local root = vim.fs.root(bufnr, function(name, path)
>     if name == 'package.json' and not path:match('node_modules') then
>       return true
>     end
>     return name == '.git'
>   end)
>   if root then on_dir(root) end
> end,
> ```

#### VS Code

Install from the [VS Code Marketplace][vscodemarketplace].

The extension includes the language server binary, so no additional installation is
required.

#### Zed

Install from the [Zed Extensions][zedextensions] page, or search for
"Design Tokens" in Zed's extension panel.

## ⚙️ Configuration

In order to use DTLS, you need to first configure it to know where to find your
design tokens, and you can also provide it with options for how to deal with
them.

### 🪙 Token files

Design Tokens Language Server uses the
[DTCG](https://tr.designtokens.org/format/) format for design tokens. If you
have a design token file in a different format, you can use
[style-dictionary](https://styledictionary.com/info/dtcg/) to convert it to
DTCG.

#### Schema Version Detection

DTLS automatically detects which schema version your token files use based on
file contents (duck-typing). It looks for features like structured color values,
`$ref`, and `$extends` to identify 2025.10 stable, falling back to Editor's
Draft for ambiguous files. See [Schema Versioning Guide](./docs/SCHEMA_VERSIONING.md)
for details on detection and migration.

#### Configuration

You can configure the language server globally or on a per-project basis.
Per-project configuration is done via a `designTokensLanguageServer` block in
your project's `package.json`.

> [!NOTE]
> If you have better ideas on how to source token files in a given project,
> Please open an issue with your suggestions.

Add a `designTokensLanguageServer` block to your project's `package.json`, with
references to tokens files. Entries under `tokensFiles` can be either a string
or an object with `path` and `prefix` properties. The `path` property can be a
relative path or a deno-style npm specifier.

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

### Network Fallback

When using `npm:` specifiers for token packages, DTLS normally resolves them
from `node_modules`. If the package isn't installed locally, you can enable
**network fallback** to fetch tokens from a CDN (default:
[unpkg.com](https://unpkg.com), configurable via the `cdn` option).

This is opt-in and disabled by default.

#### Enable in package.json

```json
{
  "designTokensLanguageServer": {
    "networkFallback": true,
    "networkTimeout": 30,
    "cdn": "unpkg",
    "tokensFiles": [
      "npm:@my-design-system/tokens/tokens.json"
    ]
  }
}
```

#### Enable in VS Code

Set `designTokensLanguageServer.networkFallback` to `true` in your VS Code
settings.

#### Enable in `.config/design-tokens.yaml`

```yaml
cdn: unpkg
files:
  - npm:@my-design-system/tokens/tokens.json
```

Note: The `.config/design-tokens.yaml` format supports the `cdn` option, but
`networkFallback` and `networkTimeout` must be set via package.json or VS Code
settings.

#### Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `networkFallback` | `boolean` | `false` | Enable CDN fallback for package specifiers |
| `networkTimeout` | `number` | `30` | Max seconds to wait for CDN requests |
| `cdn` | `string` | `"unpkg"` | CDN provider: `unpkg`, `esm.sh`, `esm.run`, `jspm`, `jsdelivr` |

#### Security

- Network fallback is **opt-in** -- it never fetches from the network unless
  explicitly enabled
- Responses are limited to 10 MB to prevent resource exhaustion
- Requests have a configurable timeout (default 30 seconds)
- Only `npm:` specifiers with a file component trigger CDN lookups

### Global configuration

You can set up global configuration in your editor's LSP settings for DTLS. This
configuration will be used as fallbacks for all projects. There's no need to set
them if your project already has a `designTokensLanguageServer` block in its
`package.json`.

For example, in your lsp config for neovim:

```lua
return {
  cmd = { 'design-tokens-language-server' },
  root_markers = { 'package.json', '.git' },
  filetypes = { 'css', 'html', 'javascript', 'javascriptreact', 'typescript', 'typescriptreact' },
  settings = {
    dtls = {
      tokensFiles = {
        {
          path = "~/path/to/tokens.json",
          prefix = "my-ds",
        },
      }
    }
  }
}
```

### Token Prefixes

The DTCG format does not require a prefix for tokens, but it is recommended to
use a prefix to avoid conflicts with other design systems. if your token files
do not nest all of their tokens under a common prefix, you can pass one yourself
in the `prefix` property of the token file object.

### Group Markers

> [!IMPORTANT]
> Group markers are **only used with Editor's Draft schema**. The 2025.10 stable
> specification uses the standardized `$root` reserved token name instead.

Because the DTCG format is nested, a conflict can emerge when the token file
author wants to define a group of tokens, but have the group name also be a
token. For example, `--token-color-red` and `--token-color-red-darker` are both
valid tokens, but unless the author manually prefixes each token in the `red`
group with `red-`, it would not be possible to define a token called
`--token-color-red`.

**Editor's Draft Schema:**
Design Tokens Language Server uses "group markers" to contain the token data for
a group. The group marker is a well-known token name that is used to represent
the group's top-most token.

The default group markers are `_`, `@`, and `DEFAULT`, because those are
examples mentioned in the various issues on DTCG and style-dictionary which
address the problem of tokens which are also groups.

If you include the `groupMarkers` property (either project-wide or for a
specific token file), then tokens with the same name as any of the group markers
will be treated as a group, and that tokens's data will be used for the group
name, minus the group marker.

**2025.10 Stable Schema:**
Use the standardized `$root` reserved token name instead of group markers. No
configuration needed.

**Editor's Draft Example:**

```json
{
  "color": {
    "red": {
      "_": {
        "$value": "#FF0000",
        "$description": "Red color"
      },
      "darker": {
        "$value": "#AA0000",
        "$description": "Darker red color"
      }
    }
  }
}
```

This creates tokens: `--color-red` and `--color-red-darker`.

Configure group markers in `package.json`:

```json
"designTokensLanguageServer": {
  "prefix": "my-ds",
  "groupMarkers": ["_"],
}
```

**2025.10 Example:**

```json
{
  "$schema": "https://www.designtokens.org/schemas/2025.10/format.json",
  "color": {
    "red": {
      "$root": {
        "$type": "color",
        "$value": {
          "colorSpace": "srgb",
          "components": [1.0, 0, 0],
          "alpha": 1.0
        }
      },
      "darker": {
        "$type": "color",
        "$value": {
          "colorSpace": "srgb",
          "components": [0.67, 0, 0],
          "alpha": 1.0
        }
      }
    }
  }
}
```

This creates the same tokens: `--color-red` and `--color-red-darker`. No
configuration needed - `$root` is standardized.

## 🤝 Contributing

See [CONTRIBUTING.md][contributingmd]

[build]:     https://github.com/bennypowers/design-tokens-language-server/actions/workflows/build.yaml/badge.svg
[buildyaml]: https://github.com/bennypowers/design-tokens-language-server/actions/workflows/build.yaml
[coverage]: https://codecov.io/gh/bennypowers/design-tokens-language-server/graph/badge.svg?token=9VOMFXI5GQ
[coveragereport]: https://codecov.io/gh/bennypowers/design-tokens-language-server/
[dtcg]: https://www.designtokens.org/
[contributingmd]: ./CONTRIBUTING.md
[issue]: https://github.com/bennypowers/design-tokens-language-server/issues/new/
[zedextensions]: https://zed.dev/extensions/design-tokens
[vscodemarketplace]: https://marketplace.visualstudio.com/items?itemName=pwrs.design-tokens-language-server-vscode
[neovimlspdocs]: https://neovim.io/doc/user/lsp.html
[cwcdash]: https://neovim.io/doc/user/windows.html#CTRL-W_g_CTRL-%5D
[editorsdraft]: https://second-editors-draft.tr.designtokens.org/format/
[202510stable]: https://www.designtokens.org/tr/2025.10/
