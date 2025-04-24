import { Language, Parser } from "web-tree-sitter";

import { readAll } from "jsr:@std/io/read-all";

const f = await Deno.open(new URL("./tree-sitter-css.wasm", import.meta.url));

const grammar = await readAll(f);

await Parser.init();

const Css = await Language.load(grammar);

export const parser = new Parser().setLanguage(Css);
