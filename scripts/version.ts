import * as ParseArgs from "@std/cli/parse-args";
import * as SemVer from "jsr:@std/semver";
import * as Toml from "jsr:@std/toml";
const { _: [arg] } = ParseArgs.parseArgs(Deno.args);

const encoder = new TextEncoder();
const decoder = new TextDecoder();

async function getVersion(arg: string | number): Promise<string | null> {
  if (typeof arg !== "string") return null;
  if (arg === "help") return null;
  switch (arg) {
    case "major":
    case "minor":
    case "patch":
      try {
        const { default: manifest } = await import("../package.json", {
          with: { type: "json" },
        });
        const currentVersion = manifest.version;
        const semver = SemVer.parse(currentVersion);
        const nextVersion = SemVer.increment(semver, arg);
        return SemVer.format(nextVersion).trim();
      } catch {
        return null;
      }
    default:
      try {
        return SemVer.format(SemVer.parse(arg)).trim();
      } catch {
        return null;
      }
  }
}

const version = await getVersion(arg);

if (!version) {
  console.log(`usage:
  deno -A version.ts <version>
  deno -A version.ts patch
  deno -A version.ts minor
  deno -A version.ts major
`);
  Deno.exit(1);
}

console.log(`Bumping version to ${version}`);

/** changed files to be committed */
const updatedPaths: string[] = [];

for (
  const file of [
    "../package.json",
    "../extensions/vscode/client/package.json",
    "../extensions/vscode/package.json",
  ]
) {
  const { default: manifest } = await import(file, {
    with: { type: "json" },
  });
  if (manifest.version === version) continue;
  const path = file.replace("../", "");
  console.log(
    `  ... in ${path} from ${manifest.version} to ${version}`,
  );
  manifest.version = version.toString().replace(/^v/, "");
  const content = JSON.stringify(manifest, null, 2) + "\n";
  await Deno.writeFile(
    new URL(file, import.meta.url),
    encoder.encode(content),
  );
  updatedPaths.push(path);
}

extension: {
  const url = new URL("../extensions/zed/extension.toml", import.meta.url);
  const content = await Deno.readTextFile(url);
  const parsed = Toml.parse(content);
  if (parsed.version === version) break extension;
  parsed.version = version.toString().replace(/^v/, "");
  const path = url.pathname.replace(Deno.cwd() + "/", "");
  console.log(
    `  ... in ${path} from ${parsed.version} to ${version}`,
  );
  await Deno.writeFile(url, encoder.encode(Toml.stringify(parsed)));
  updatedPaths.push(path);
}

cargo: {
  const url = new URL("../extensions/zed/Cargo.toml", import.meta.url);
  const content = await Deno.readTextFile(url);
  const parsed = Toml.parse(content) as { package: { version: string } };
  if (parsed.package.version == version) break cargo;
  parsed.package.version = version.toString().replace(/^v/, "");
  const path = url.pathname.replace(Deno.cwd() + "/", "");
  console.log(
    `  ... in ${path} from ${parsed.package.version} to ${version}`,
  );
  await Deno.writeFile(url, encoder.encode(Toml.stringify(parsed)));
  updatedPaths.push(path);
}

/**
 * template literal tag which executes shell command and handles errors
 * ensures that input like $`git tag v${version}` and $`git commit -m "chore: prepare version v${version}"` is passed to the shell
 * correctly, without modifying the input
 */
async function $(
  strings: TemplateStringsArray,
  ...values: string[]
): Promise<void> {
  /**
   * The full shell command
   * @example $`git commit -m "chore: prepare version v${version}"` => `git commit -m "chore: prepare version v0.1.0"`
   */
  const fullCommand = strings
    .map((str, i) => str + (values[i] || ""))
    .join("")
    // ensure that quoted strings are preserved
    .replace(/(["'])/g, "$1");

  const [commandName] = fullCommand.split(" ");

  console.log(`$ ${fullCommand}`);

  /**
   * The command args, preserving quoted input strings
   * Split full command by spaces, but preserve quoted strings, i.e. do not split spaces within double or single quotes
   * @example "git commit -m "chore: prepare version v0.1.0"" => ["commit", "-m", "chore: prepare version v0.1.0"]
   */
  const [, ...args] = fullCommand.match(/(?:[^\s"]+|"[^"]*")+/g)?.map((arg) => {
    // remove quotes from args
    if (arg.startsWith('"') && arg.endsWith('"')) {
      return arg.slice(1, -1);
    }
    if (arg.startsWith("'") && arg.endsWith("'")) {
      return arg.slice(1, -1);
    }
    return arg;
  }) ?? [];

  const command = new Deno.Command(commandName, { args });
  const output = await command.output();
  if (output.code !== 0) {
    console.log(
      `failed to run command: ${commandName} ${args.join(" ")}`,
      decoder.decode(output.stdout),
      decoder.decode(output.stderr),
    );
    Deno.exit(1);
  }
}

await $`git add ${updatedPaths.join(" ")}`;
await $`git commit -m "chore: prepare version ${version}"`;
await $`git tag v${version}`;
