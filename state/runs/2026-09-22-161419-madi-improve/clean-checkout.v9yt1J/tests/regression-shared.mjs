import { spawn } from "node:child_process";
import { mkdir, writeFile, readFile, open } from "node:fs/promises";
import {randomUUID} from 'node:crypto';
import path from 'node:path';
import {freshScreenshots,snapshotScreenshots,validBundle} from './screenshot-evidence.mjs';

// A disposable development service only. Every suite owns its fixtures; suites
// requiring isolated Go protocol fixtures deliberately remain separate.
const suites = process.argv.slice(2);
if (!suites.length)
  throw new Error("Specify shared-server browser suite names");
await mkdir("test-results/regression-shared", { recursive: true });
let results = [];
try {
  results = JSON.parse(await readFile("test-results/regression-shared/report.json", "utf8"))
    .filter(result => !suites.includes(result.suite));
} catch {}
async function currentBundle() {
  const response=await fetch(process.env.MADI_BASE_URL || "http://127.0.0.1:8080",{signal:AbortSignal.timeout(10000),cache:'no-store'});
  if(!response.ok)throw new Error('Cannot verify served bundle');
  const bundle=(await response.text()).match(/(?:main|index)-[A-Za-z0-9_-]+\.js/)?.[0];
  if(!validBundle(bundle))throw new Error('Cannot identify served bundle');
  return bundle;
}
const bundle=await currentBundle(),batchID=randomUUID();
for (const suite of suites) {
  if (!/^[a-z0-9-]+$/.test(suite)) throw new Error("Invalid suite name");
  const log = `test-results/regression-shared/${suite}.log`;
  const output = await open(log, "w");
  const env = { ...process.env };
  env.MADI_SCREENSHOT_DIR = path.resolve(env.MADI_REGRESSION_SCREENSHOT_ROOT || 'test-results/regression-shared/screenshots',suite);
  await mkdir(env.MADI_SCREENSHOT_DIR, { recursive: true });
  const before=await snapshotScreenshots(env.MADI_SCREENSHOT_DIR);
  const bundleBefore=await currentBundle(),started=Date.now();
  let timedOut=false;
  if (suite === "browser" && env.MADI_REGRESSION_EMAIL) {
    env.MADI_TEST_EMAIL = env.MADI_REGRESSION_EMAIL;
    env.MADI_TEST_PASSWORD = env.MADI_REGRESSION_PASSWORD;
  }
  console.log(`START ${suite}`);
  const exitCode = await new Promise((resolve) => {
    const child = spawn(process.execPath, [`tests/${suite}.mjs`], {
      env,
      stdio: ["ignore", output.fd, output.fd],
    });
    const timer = setTimeout(() => {timedOut=true;child.kill("SIGTERM")}, 240_000);
    child.on("exit", (code, signal) => {
      clearTimeout(timer);
      resolve(code ?? signal ?? -1);
    });
    child.on("error", (error) => {
      clearTimeout(timer);
      resolve(error.message);
    });
  });
  await output.close();
  const finished=Date.now();
  let bundleAfter='',evidenceError='';
  let screenshots=[];
  try {
    bundleAfter=await currentBundle();
    screenshots=await freshScreenshots(env.MADI_SCREENSHOT_DIR,before,started,finished);
  } catch(error) {evidenceError=error.message}
  const result = {
    evidence_version:2,
    batch_id:batchID,
    suite,
    bundle,
    bundle_before:bundleBefore,
    bundle_after:bundleAfter,
    started_at:new Date(started).toISOString(),
    checked_at: new Date(finished).toISOString(),
    screenshot_dir:path.relative(process.cwd(),env.MADI_SCREENSHOT_DIR),
    screenshots,
    evidence_error:evidenceError,
    timed_out:timedOut,
    ok: exitCode === 0 && !timedOut && !evidenceError && bundleBefore===bundle && bundleAfter===bundle,
    exit_code: exitCode,
    seconds: Math.round((finished - started) / 100) / 10,
    log,
  };
  results.push(result);
  await writeFile(
    "test-results/regression-shared/report.json",
    JSON.stringify(results, null, 2),
  );
  console.log(
    `${result.ok ? "PASS" : "FAIL"} ${suite} (${result.seconds}s) — ${log}`,
  );
}
process.exitCode = results.every((r) => r.ok) ? 0 : 1;
