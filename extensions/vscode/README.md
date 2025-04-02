# Minimum Viable VS Code Language Server Extension

NOTE: This is heavily based on [lsp-sample from vscode-extension-samples][sample] with the goal of removing example-specific code to ease starting a new Language Server.

This project aims to provide a starting point for developing a self-contained Language Server Extension for VS Code using TypeScript.

"Self-contained" in this context means that this extension bundles its own language server code rather than wrapping an existing language server executable.

As an MVP, this omits

- linting
- testing
- behavior in the language server itself (besides connecting and listening to document changes)

## Getting Started

1. Clone this repo
2. Replace items in `package.json` marked `REPLACE_ME` with text related to your extension
3. Do the same for `client/package.json` and `server/package.json`
4. Do the same in `client/src/extension.ts`
5. Run `npm install` from the repo root.

To make it easy to get started, this language server will run on _every_ file type by default. To target specific languages, change

`package.json`'s `activationEvents` to something like

```
"activationEvents": [
  "onLanguage:plaintext"
],
```

And change the `documentSelector` in `client/src/extension.ts` to replace the `*` (e.g.)

```
documentSelector: [{ scheme: "file", language: "plaintext" }],
```

## Developing your extension

To help verify everything is working properly, we've included the following code in `server.ts` after the `onInitialize` function:

```typescript
documents.onDidChangeContent((change) => {
  connection.window.showInformationMessage(
    "onDidChangeContent: " + change.document.uri
  );
});
```

From the root directory of this project, run `code .` Then in VS Code

1. Build the extension (both client and server) with `⌘+shift+B` (or `ctrl+shift+B` on windows)
2. Open the Run and Debug view and press "Launch Client" (or press `F5`). This will open a `[Extension Development Host]` VS Code window.
3. Opening or editing a file in that window should show an information message in VS Code like you see below.

   ![example information message](https://semanticart.com/misc-images/minimum-viable-vscode-language-server-extension-info-message.png)

4. Edits made to your `server.ts` will be rebuilt immediately but you'll need to "Launch Client" again (`⌘-shift-F5`) from the primary VS Code window to see the impact of your changes.

[Debugging instructions can be found here][debug]

## Distributing your extension

Read the full [Publishing Extensions doc][publish] for the full details.

Note that you can package and distribute a standalone `.vsix` file without publishing it to the marketplace by following [these instructions][vsix].

## Anatomy

```
.
├── .vscode
│   ├── launch.json         // Tells VS Code how to launch our extension
│   └── tasks.json          // Tells VS Code how to build our extension
├── LICENSE
├── README.md
├── client
│   ├── package-lock.json   // Client dependencies lock file
│   ├── package.json        // Client manifest
│   ├── src
│   │   └── extension.ts    // Code to tell VS Code how to run our language server
│   └── tsconfig.json       // TypeScript config for the client
├── package-lock.json       // Top-level Dependencies lock file
├── package.json            // Top-level manifest
├── server
│   ├── package-lock.json   // Server dependencies lock file
│   ├── package.json        // Server manifest
│   ├── src
│   │   └── server.ts       // Language server code
│   └── tsconfig.json       // TypeScript config for the client
└── tsconfig.json           // Top-level TypeScript config
```

[debug]: https://code.visualstudio.com/api/language-extensions/language-server-extension-guide#debugging-both-client-and-server
[sample]: https://github.com/microsoft/vscode-extension-samples/tree/main/lsp-sample
[publish]: https://code.visualstudio.com/api/working-with-extensions/publishing-extension
[vsix]: https://code.visualstudio.com/api/working-with-extensions/publishing-extension#packaging-extensions
