import { readdir, readFile } from "node:fs/promises";
import { join } from "node:path";
// Reject lossy encoding before it reaches profile labels or production bundles.
async function check(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      await check(path);
      continue;
    }
    if (!/\.(tsx?|css)$/.test(path)) continue;
    const bytes = await readFile(path);
    const text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
    if (text.includes("\uFFFD"))
      throw new Error(`Invalid replacement character in ${path}`);
  }
}
await Promise.all(["app", "components", "lib"].map(check));
console.log("UI text is valid UTF-8 without replacement characters.");
