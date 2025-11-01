## 🏗️ Building

DTLS is written in Go and uses Make for building. Clone this repo and run:

```sh
make build
```

To install the binary to `~/.local/bin/design-tokens-language-server`:

```sh
make install
```

To build binaries for all platforms (Linux, macOS, Windows):

```sh
make build-all
```

See `make help` for all available build targets.

### 🧪 Testing

Run tests with:

```sh
make test
```

Generate coverage reports with:

```sh
make test-coverage
```

View coverage in browser:

```sh
make show-coverage
```

## 🔍 Linting

Run golangci-lint:

```sh
make lint
```

This will automatically install golangci-lint if not already present.

## 🪲 Debugging

The server logs to `~/.local/state/design-tokens-language-server/dtls.log` by
default.

```sh
tail -f ~/.local/state/design-tokens-language-server/dtls.log
```

If you'd like to trace lsp messages in real time, try
[lsp-devtools](https://lsp-devtools.readthedocs.io/en/latest/lsp-devtools/guide/inspect-command.html)

[dtcg]: https://tr.designtokens.org/format/

## 🚚 Releasing

To create a new release, use the Makefile:

```bash
make release v0.1.1
```
or

```bash
make release minor
```

This single command handles everything:
1. Updates version in `extensions/vscode/package.json` and `extensions/zed/extension.toml`
2. Prompts you to commit the changes
3. Runs `gh release create` which:
   - Creates git tag `v0.1.1`
   - Pushes commit and tag to GitHub
   - Opens interactive wizard where you can edit release notes
   - Creates the GitHub release
4. Triggers CI to build binaries and publish extensions

**Important:** Always use `make release` instead of `gh release create` directly. The CI includes validation that will fail if extension versions don't match the git tag version.

#### What if I forget and use `gh release create`?

If you accidentally create a release without updating versions:
1. The CI build will fail immediately with a clear error message
2. Delete the broken release and tag from GitHub
3. Run `make release v0.1.1` to create the release properly (this handles version updates automatically)
