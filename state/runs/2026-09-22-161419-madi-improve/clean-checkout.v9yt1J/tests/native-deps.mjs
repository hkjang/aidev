// Read-only package manager plan + user-owned extraction: never apt install/sudo.
import { execFileSync } from "node:child_process";
import { mkdtemp, mkdir, readdir } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
const dir = await mkdtemp(path.join(os.tmpdir(), "madi-desktop-deps-"));
const prefix = path.join(dir, "root");
await mkdir(prefix);
const plan = execFileSync(
  "apt-get",
  [
    "-s",
    "--no-install-recommends",
    "install",
    "libwebkit2gtk-4.1-dev",
    "libayatana-appindicator3-dev",
    "librsvg2-dev",
    "patchelf",
    "webkitgtk-webdriver",
    "xdotool",
    "gstreamer1.0-plugins-base",
  ],
  { encoding: "utf8", maxBuffer: 10 << 20 },
);
const packages = [...plan.matchAll(/^Inst ([^ ]+)/gm)].map((value) => value[1]);
console.log(
  `Extracting ${packages.length} official repository packages into ${prefix}`,
);
if (packages.length)
  execFileSync("apt-get", ["download", ...packages], {
    cwd: dir,
    stdio: "inherit",
  });
for (const file of await readdir(dir))
  if (file.endsWith(".deb"))
    execFileSync("dpkg-deb", ["-x", path.join(dir, file), prefix]);
console.log(`MADI_DESKTOP_SYSROOT=${prefix}`);
