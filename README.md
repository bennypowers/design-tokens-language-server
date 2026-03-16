# Design Tokens Language Server

> [!IMPORTANT]
> **This project has moved to [asimonim][asimonim].**
> Editor extensions (VS Code, Zed, Claude Code) and the language server are now maintained there.
>
> This repository builds `design-tokens-language-server`, a thin wrapper binary
> that delegates to asimonim's LSP implementation. It exists for backward
> compatibility with editor configs that reference the old binary name.

## Installation

Install via [asimonim][asimonim] directly: 

```
go install bennypowers.dev/asimonim
```

Or download the latest release for your platform from the
[releases page][releases].

Please consider putting a ⭐ on the new repo, if you find this package helpful.

### Gentoo Linux

```bash
eselect repository enable bennypowers
emaint sync -r bennypowers
emerge dev-util/design-tokens-language-server
```

## Usage

```bash
design-tokens-language-server
```

The binary starts an LSP server on stdin/stdout, identical to `asimonim lsp`.

## Configuration

See the [asimonim documentation][docs].

## License

GPLv3

[asimonim]: https://github.com/bennypowers/asimonim
[releases]: https://github.com/bennypowers/asimonim/releases
[docs]: https://github.com/bennypowers/asimonim#configuration
