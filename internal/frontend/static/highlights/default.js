import { StreamTokenizer, codeToHtml, codeToTokens, createHighlighter, init, isSupportedLanguage } from "./highlighter.js";
import { LiveTokenizer, tokenNames } from "./live.js";
import "./index.js";
import wasmBytes from "./highlights.wasm.mjs";
//#region lib/default.ts
init(new WebAssembly.Module(wasmBytes));
//#endregion
export { LiveTokenizer, StreamTokenizer, codeToHtml, codeToTokens, createHighlighter, init, isSupportedLanguage, tokenNames };

//# sourceMappingURL=default.js.map