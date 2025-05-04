const version = await new Deno.Command("git", {
  args: [
    "describe",
    "--tags",
    "--abbrev=0",
    "--match=v*",
  ],
})
  .output()
  .then((result) => {
    if (result.code === 0) {
      return new TextDecoder().decode(result.stdout).trim().replace(/^v/, "");
    } else {
      return null;
    }
  });

if (!version) {
  console.error("Failed to get the latest version tag");
  Deno.exit(1);
}

import manifest from "../package.json" with { type: "json" };

const errors = [];

console.log(`Checking for consistency of version ${manifest.version}...`);

if (manifest.version !== version) {
  errors.push(
    `package.json: version (${manifest.version}) does not match the latest tag (${version})`,
  );
}

for (
  const file of [
    "extensions/vscode/client/package.json",
    "extensions/vscode/package.json",
  ]
) {
  const { default: manifest } = await import(`../${file}`, {
    with: { type: "json" },
  });
  if (manifest.version !== version) {
    errors.push(
      `${file}: version (${manifest.version}) does not match the latest tag (${version})`,
    );
  }
}

for (
  const file of [
    "extensions/zed/extension.toml",
    "extensions/zed/Cargo.toml",
  ]
) {
  const content = await Deno.readTextFile(file);
  const version = content.match(/version = "(.*?)"/)?.[1];
  if (version !== manifest.version) {
    errors.push(
      `${file}: version (${manifest.version}) does not match the latest tag (${version})`,
    );
  }
}

if (errors.length) {
  for (const error of errors) {
    console.log(error);
  }
  Deno.exit(1);
}

Deno.exit(0);
