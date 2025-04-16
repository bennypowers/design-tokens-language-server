import { parser, TSQueryResult } from "./documents.ts";
import { HardNode } from "https://deno.land/x/deno_tree_sitter@0.2.8.5/tree_sitter.js";
import { LightDarkValuesQuery } from "./tree-sitter/queries.ts";

export function getLightDarkValues(value: string) {
  const tree = parser.parse(`a{b:${value}}`);
  const results = (tree.rootNode as HardNode).query(LightDarkValuesQuery, {}) as unknown as TSQueryResult[];
  const lightNode = results.flatMap(cap => cap.captures).find(cap => cap.name === 'lightValue');
  const darkNode = results.flatMap(cap => cap.captures).find(cap => cap.name === 'darkValue');
  return [lightNode?.node.text, darkNode?.node.text];
}
