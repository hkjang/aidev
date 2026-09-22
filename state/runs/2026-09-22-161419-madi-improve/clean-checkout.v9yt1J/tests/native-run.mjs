// Optional user-owned Linux sysroot runner for environments without apt install rights.
// A private mount namespace overlays symlinks; the host filesystem is never changed.
import { readdir, mkdtemp, mkdir, symlink } from "node:fs/promises";
import { spawn } from "node:child_process";
import os from "node:os";
import path from "node:path";
const prefix = process.env.MADI_DESKTOP_SYSROOT;
if (!prefix) throw new Error("MADI_DESKTOP_SYSROOT is required");
const scratch = await mkdtemp(path.join(os.tmpdir(), "madi-native-run-"));
const merged = path.join(scratch, "libraries"),
  profile = path.join(scratch, "profile");
await Promise.all([mkdir(merged), mkdir(profile)]);
const system = "/usr/lib/x86_64-linux-gnu",
  extra = path.join(prefix, "usr/lib/x86_64-linux-gnu"),
  isolatedExtra = "/usr/madi-native/usr/lib/x86_64-linux-gnu";
const entries = new Map();
for (const name of await readdir(system))
  entries.set(name, path.join("/usr/madi-system-libraries", name));
for (const name of await readdir(extra))
  entries.set(name, path.join(isolatedExtra, name));
for (const [name, target] of entries)
  await symlink(target, path.join(merged, name));
const [command, ...args] = process.argv.slice(2);
if (!command) throw new Error("A command is required");
// Keep dependency targets under /usr: GTK's nested image-loader sandbox only
// binds /usr, so /tmp symlink targets would vanish inside its security boundary.
const usrMounts = (await readdir("/usr")).flatMap((name) => [
  "--ro-bind",
  `/usr/${name}`,
  `/usr/${name}`,
]);
const child = spawn(
  "bwrap",
  [
    "--die-with-parent",
    "--unshare-pid",
    "--ro-bind",
    "/",
    "/",
    "--bind",
    "/tmp",
    "/tmp",
    "--dev-bind",
    "/dev",
    "/dev",
    "--proc",
    "/proc",
    "--tmpfs",
    "/usr",
    ...usrMounts,
    "--ro-bind",
    prefix,
    "/usr/madi-native",
    "--ro-bind",
    system,
    "/usr/madi-system-libraries",
    "--ro-bind",
    merged,
    system,
    command,
    ...args,
  ],
  {
    stdio: "inherit",
    env: {
      ...process.env,
      LD_LIBRARY_PATH: isolatedExtra,
      XDG_CONFIG_HOME: profile,
      XDG_DATA_HOME: profile,
      XDG_CACHE_HOME: profile,
      TAURI_WEBVIEW_AUTOMATION: "true",
      GDK_BACKEND: "x11",
    },
  },
);
console.log("Native isolated profile:", profile);
child.on("exit", (code) => {
  process.exitCode = code || 0;
});
for (const signal of ["SIGINT", "SIGTERM"])
  process.on(signal, () => child.kill(signal));
