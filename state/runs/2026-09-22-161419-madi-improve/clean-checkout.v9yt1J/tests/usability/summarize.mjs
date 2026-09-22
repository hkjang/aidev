import { readFile, stat } from "node:fs/promises";
import { summarize } from "./model.mjs";
const files = process.argv.slice(2);
if (!files.length) {
  console.error(
    "사용법: node tests/usability/summarize.mjs <관찰JSON> [추가JSON...]",
  );
  process.exit(2);
}
try {
  let records = [];
  for (const file of files) {
    if ((await stat(file)).size > 10485760)
      throw new Error("입력 파일은 10MiB 이하만 읽습니다");
    const value = JSON.parse(await readFile(file, "utf8"));
    if (!Array.isArray(value)) throw new Error("기록 배열 파일이 필요합니다");
    records.push(...value);
    if (records.length > 10000)
      throw new Error("최대10000개 기록을 넘었습니다");
  }
  process.stdout.write(JSON.stringify(summarize(records), null, 2) + "\n");
} catch (e) {
  console.error(e.message);
  process.exitCode = 1;
}
