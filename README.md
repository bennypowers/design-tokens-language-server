# Design Tokens Language Server

Editor tools for working with [design tokens][dtcg]

> [!CAUTION]
> This is extremely early software. Most features are buggy or incomplete

## Features

- **Hover**: Token description and value
- **Completions**: auto complete for design tokens
- **Diagnostics**: wrong fallback value for token
- **Code actions**: toggle fallback values in `var()` calls
- **Document Color**: display token color values in your source

## Building

> [!WARNING]
> This is being developed on Linux, might work with MacOS, and probably won't 
> yet work on windows

Install Deno and clone this repo

```sh
deno task install
```

[dtcg]: https://tr.designtokens.org/format/
