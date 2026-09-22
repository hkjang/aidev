// Complete user-owned extracted -dev symlinks using already-installed runtime libs.
import { readdir, readlink, access, cp } from "node:fs/promises";
import path from "node:path";
const root = process.env.MADI_DESKTOP_SYSROOT;
if (!root) throw new Error("MADI_DESKTOP_SYSROOT required");
const dir = path.join(root, "usr/lib/x86_64-linux-gnu");
let count = 0;
for (const item of await readdir(dir, { withFileTypes: true })) {
  if (!item.isSymbolicLink()) continue;
  const target = await readlink(path.join(dir, item.name));
  const output = path.resolve(dir, target);
  if (!output.startsWith(root + "/")) continue;
  try {
    await access(output);
    continue;
  } catch {}
  const source = path.join("/usr/lib/x86_64-linux-gnu", path.basename(target));
  try {
    await access(source);
  } catch {
    continue;
  }
  await cp(source, output, {
    dereference: true,
    errorOnExist: true,
    force: false,
  });
  count++;
}
console.log(`Completed ${count} runtime libraries in user sysroot.`);
