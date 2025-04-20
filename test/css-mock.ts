import { TestDocuments } from "#test-helpers";

export const documents = new TestDocuments();

export {
  captureIsTokenCall,
  captureIsTokenName,
  getLightDarkValues,
  lspPosToTsPos,
  lspRangeIsInTsNode,
  lspRangeToTsRange,
  tsNodeIsInLspRange,
  tsRangeToLspRange,
} from "#css";
