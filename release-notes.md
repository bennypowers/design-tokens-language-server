## What's New

### DTCG Resolver Document Support

Load tokens from [DTCG resolver documents](https://tr.designtokens.org/format/#resolver-documents) via the `resolvers` config option. Resolver documents define a `resolutionOrder` with `$ref` entries pointing to source token files, enabling multi-file token resolution with proper alias handling across sources.

Configure in `package.json`:
```json
{
  "designTokensLanguageServer": {
    "resolvers": ["./src/tokens/tokens.resolver.json"]
  }
}
```

Or in `.config/design-tokens.yaml`:
```yaml
resolvers:
  - ./src/tokens/tokens.resolver.json
```

Supports:
- Inline sources and named set references (`#/sets/<name>`)
- JSON Pointer escaping (RFC 6901)
- Fragment identifiers in `$ref` values
- Relative and absolute resolver paths
- CDN fallback for package specifiers (`npm:`, `jsr:`)
- Cross-file alias resolution across resolver sources

### Improved Hover Formatting

Structured color values now display with improved formatting in hover tooltips.

### Documentation

- Added Gentoo Linux installation instructions
