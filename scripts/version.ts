import { parseArgs } from "@std/cli/parse-args";
import { parse, stringify } from "jsr:@std/toml";
const { _: [version] } = parseArgs(Deno.args);

if (!version) {
  console.log(Deno.args);
  console.log("usage: deno -A version.ts --version <version>");
  Deno.exit(1);
}

const encoder = new TextEncoder();

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
    `Bumping version in ${path} from ${manifest.version} to ${version}`,
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
  const parsed = parse(content);
  if (parsed.version === version) break extension;
  parsed.version = version.toString().replace(/^v/, "");
  const path = url.pathname.replace(Deno.cwd() + "/", "");
  console.log(
    `Bumping version in ${path} from ${parsed.version} to ${version}`,
  );
  await Deno.writeFile(url, encoder.encode(stringify(parsed)));
  updatedPaths.push(path);
}

cargo: {
  const url = new URL("../extensions/zed/Cargo.toml", import.meta.url);
  const content = await Deno.readTextFile(url);
  const parsed = parse(content) as { package: { version: string } };
  if (parsed.package.version == version) break cargo;
  parsed.package.version = version.toString().replace(/^v/, "");
  const path = url.pathname.replace(Deno.cwd() + "/", "");
  console.log(
    `Bumping version in ${path} from ${parsed.package.version} to ${version}`,
  );
  await Deno.writeFile(url, encoder.encode(stringify(parsed)));
  updatedPaths.push(path);
}

const add = await new Deno.Command("git", { args: ["add", ...updatedPaths] })
  .output();

if (add.code !== 0) {
  console.error("Failed to add files to git");
  console.error(new TextDecoder().decode(add.stdout));
  console.error(new TextDecoder().decode(add.stderr));
  Deno.exit(1);
}

const commit = await new Deno.Command("git", {
  args: ["commit", "-m", `chore: prepare version v${version}`],
}).output();

if (commit.code !== 0) {
  console.error("Failed to commit files to git");
  console.error(new TextDecoder().decode(commit.stdout));
  console.error(new TextDecoder().decode(commit.stderr));
  Deno.exit(1);
}

const tag = await new Deno.Command("git", {
  args: ["tag", `v${version}`],
}).output();

if (tag.code !== 0) {
  console.error("Failed to tag commit");
  console.error(new TextDecoder().decode(tag.stdout));
  console.error(new TextDecoder().decode(tag.stderr));
  Deno.exit(1);
}
