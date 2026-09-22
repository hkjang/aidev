import { readdirSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import type { Plugin } from "vite";

// PDF decoding never uses a CDN, including CMaps, fallback fonts and WASM.
export function pdfjsAssets(): Plugin {
  return {
    name: "madi-offline-pdf-assets",
    generateBundle() {
      for (const group of ["cmaps", "standard_fonts", "wasm"]) {
        const root = new URL(
          `./node_modules/pdfjs-dist/${group}/`,
          import.meta.url,
        );
        for (const entry of readdirSync(fileURLToPath(root), {
          withFileTypes: true,
        })) {
          if (!entry.isFile()) continue;
          // PDF scripting/forms are not enabled in madi. This optional sandbox
          // is not referenced by the rendering worker or the document API.
          if (group === "wasm" && /^quickjs-eval\.(js|wasm)$/.test(entry.name))
            continue;
          this.emitFile({
            type: "asset",
            fileName: `assets/pdfjs/${group}/${entry.name}`,
            source: readFileSync(new URL(entry.name, root)),
          });
        }
      }
    },
  };
}
