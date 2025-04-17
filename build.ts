import * as esbuild from "esbuild";
import { denoPlugins } from "@luca/esbuild-deno-loader";

await esbuild.build({
  plugins: [...denoPlugins()],
  entryPoints: [ "src/main.ts" ],
  outfile: "./dist/main.js",
  bundle: true,
  format: "esm",
});

await esbuild.stop();

enum Arch {
  "aarch64-apple-darwin" = "aarch64-apple-darwin",
  "aarch64-unknown-linux-gnu" = "aarch64-unknown-linux-gnu",
  "x86_64-apple-darwin" = "x86_64-apple-darwin",
  "x86_64-pc-windows-msvc" = "x86_64-pc-windows-msvc",
  "x86_64-unknown-linux-gnu" = "x86_64-unknown-linux-gnu"
}

async function compile(arch?: Arch) {
  const args = [
    "compile",
    "--unstable-temporal",
    "--allow-all",
    "--no-lock",
    "--no-check",
    "--no-remote",
    "--no-config",
    "--reload",
    "--import-map=import-map-bundle.json",
    "--include=./src/server/css/tree-sitter/tree-sitter-css.wasm",
    ...!arch ? [] : [ "--target", `${arch}` ],
    `--output=dist/bin/design-tokens-language-server${arch ? `-${arch}` : ''}`,
    "dist/main.js",
    "--stdio",
  ].filter(x => typeof x === 'string');
  const { code } = await new Deno.Command(Deno.execPath(), {
    stdout: 'piped',
    stderr: 'piped',
    args,
  }).output();
  if (code === 0)
    console.log(`Built ${arch ?? 'native'}`)
  else {
    console.log(`deno ${args.join(' ')}`);
    throw new Error(`Could not build ${arch ?? 'native'}`);
  }
}

await compile();

await Promise.all(Object.values(Arch).map(compile));

