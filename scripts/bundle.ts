import * as esbuild from "esbuild";
import { denoPlugins } from "@luca/esbuild-deno-loader";

import { expandGlob } from "jsr:@std/fs";

const decoder = new TextDecoder();

await esbuild.build({
  plugins: [...denoPlugins()],
  entryPoints: ["src/main.ts"],
  outfile: "./dist/main.js",
  bundle: true,
  sourcemap: "inline",
  format: "esm",
});

await esbuild.stop();

enum Arch {
  "aarch64-apple-darwin" = "aarch64-apple-darwin",
  "aarch64-unknown-linux-gnu" = "aarch64-unknown-linux-gnu",
  "x86_64-apple-darwin" = "x86_64-apple-darwin",
  "x86_64-pc-windows-msvc" = "x86_64-pc-windows-msvc",
  "x86_64-unknown-linux-gnu" = "x86_64-unknown-linux-gnu",
}

const includes = await Array.fromAsync(
  expandGlob("dist/*.{wasm,js.map}"),
  (file) => `--include=${file.path}`,
);

async function compile(arch?: Arch) {
  const args = [
    "compile",
    "--allow-all",
    "--no-lock",
    "--no-check",
    "--no-remote",
    "--no-config",
    "--import-map=import-map-bundle.json",
    ...includes,
    ...!arch ? [] : ["--target", `${arch}`],
    `--output=dist/bin/design-tokens-language-server${arch ? `-${arch}` : ""}`,
    "dist/main.js",
    "--stdio",
  ].filter((x) => typeof x === "string");
  console.log(`deno ${args.join(" ")}`);
  const { code, stdout, stderr } = await new Deno.Command(Deno.execPath(), {
    stdout: "piped",
    args,
  }).output();
  if (code === 0) {
    console.log(decoder.decode(stdout));
    console.log(`Built ${arch ?? "native"}`);
  } else {
    console.log(`deno ${args.join(" ")}\n`);
    console.log(decoder.decode(stdout));
    console.log(decoder.decode(stderr));
    throw new Error(`Could not build ${arch ?? "native"}`);
  }
}

if (Deno.env.has("CI")) await Promise.all(Object.values(Arch).map(compile));
else await compile();
