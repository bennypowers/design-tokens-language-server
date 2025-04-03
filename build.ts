import * as esbuild from "esbuild";
import { denoPlugins } from "@luca/esbuild-deno-loader";

await esbuild.build({
  plugins: [...denoPlugins()],
  entryPoints: ["main.ts"],
  outfile: "./dist/main.js",
  bundle: true,
  format: "esm",
});

esbuild.stop();

