const esbuild = require("esbuild");
const fs = require("fs");
const path = require("path");

const production = process.argv.includes("--production");
const watch = process.argv.includes("--watch");

async function main() {
  // Build the VSCode extension client (TypeScript → JavaScript)
  const ctx = await esbuild.context({
    entryPoints: ["client/src/extension.ts"],
    bundle: true,
    format: "cjs",
    minify: production,
    sourcemap: !production,
    sourcesContent: false,
    platform: "node",
    outfile: "client/out/extension.js",
    external: ["vscode"],
    logLevel: "warning",
    plugins: [
      /* add to the end of plugins array */
      esbuildProblemMatcherPlugin,
    ],
  });

  if (watch) {
    await ctx.watch();
  } else {
    await ctx.rebuild();
    await ctx.dispose();
  }

  // Verify that language server binaries exist in dist/bin/
  // (These should be copied by CI or built with `make vscode-package`)
  const binDir = path.join(__dirname, "dist", "bin");
  if (fs.existsSync(binDir)) {
    const binaries = fs.readdirSync(binDir).filter((f) =>
      f.startsWith("design-tokens-language-server-")
    );
    if (binaries.length > 0) {
      console.log(
        `[vscode-build] Found ${binaries.length} language server binaries:`
      );
      binaries.forEach((b) => console.log(`  - ${b}`));
    } else {
      console.warn(
        "[vscode-build] Warning: No language server binaries found in dist/bin/"
      );
      console.warn(
        "[vscode-build] Run 'make vscode-package' to build all platform binaries"
      );
    }
  } else {
    console.warn(
      "[vscode-build] Warning: dist/bin/ directory not found"
    );
    console.warn(
      "[vscode-build] Run 'make vscode-package' to build all platform binaries"
    );
  }
}

/**
 * @type {import('esbuild').Plugin}
 */
const esbuildProblemMatcherPlugin = {
  name: "esbuild-problem-matcher",

  setup(build) {
    build.onStart(() => {
      console.log("[watch] build started");
    });
    build.onEnd((result) => {
      result.errors.forEach(({ text, location }) => {
        console.error(`✘ [ERROR] ${text}`);
        if (location == null) return;
        console.error(
          `    ${location.file}:${location.line}:${location.column}:`,
        );
      });
      console.log("[watch] build finished");
    });
  },
};

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
