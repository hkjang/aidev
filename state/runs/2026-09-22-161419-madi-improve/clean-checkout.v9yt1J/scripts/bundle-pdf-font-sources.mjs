// Connected build stage only. Preserve the GPL Liberation 1.x source separately
// from the Alpine package sources: these fonts arrive through pdfjs-dist.
import { createHash } from "node:crypto";
import { readFile, writeFile, mkdir, copyFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import path from "node:path";

const sha = (value) => createHash("sha256").update(value).digest("hex");
export const liberationSource = {
  url: "https://releases.pagure.org/liberation-fonts/liberation-fonts-1.07.4.tar.gz",
  sha256: "ad98b7498dc2992f7f0868f79b65ce4a720a3acdb63ab3f1f1cb6881117a5406",
};
const documentation = {
  url: "https://raw.githubusercontent.com/mozilla/pdf.js/v6.3.289/external/standard_fonts/README.md",
  sha256: "f21d2a0f65f7dd53bbe42ca51b40c4331f566fe91a71e689682c1b762889c47e",
};
const fonts = {
  "LiberationSans-Regular.ttf": "f8ace1f892b2bd9dc1792ba7f097fa7588f84fed48321480e04de5390828221f",
  "LiberationSans-Bold.ttf": "361c61b82d575c5c35fd9157fda8b0194bcfcd0d88ea8521a4fb5dd53d33dddc",
  "LiberationSans-Italic.ttf": "832b4406dbef23628800d3aaad21048534ac84d7e3ad955be83b8172ed8ef512",
  "LiberationSans-BoldItalic.ttf": "a224075ac17495ad0a3af3bc0a419ac0704a8b3fd1095456201fb9b095fc281d",
};

export async function bundlePDFFontSources(pdfDir, output, get) {
  const pkg = JSON.parse(await readFile(path.join(pdfDir, "package.json"), "utf8"));
  if (pkg.name !== "pdfjs-dist" || pkg.version !== "6.3.289")
    throw new Error("PDF.js version changed: review font/source provenance before bundling");
  const target = path.join(output, "pdfjs-liberation");
  await mkdir(target, { recursive: true });
  const files = [];
  for (const [name, expected] of Object.entries(fonts)) {
    const bytes = await readFile(path.join(pdfDir, "standard_fonts", name));
    if (sha(bytes) !== expected) throw new Error(`PDF.js font changed: ${name}`);
    await writeFile(path.join(target, name), bytes);
    files.push({ name, sha256: expected, bytes: bytes.length });
  }
  for (const [name, expected] of [["liberation-fonts-1.07.4.tar.gz", liberationSource], ["PDFJS-README.md", documentation]]) {
    const bytes = await get(expected.url, 8 << 20);
    if (sha(bytes) !== expected.sha256) throw new Error(`Font source SHA-256 mismatch: ${name}`);
    await writeFile(path.join(target, name), bytes);
  }
  const archive = path.join(target, "liberation-fonts-1.07.4.tar.gz");
  const member = (name) => execFileSync("tar", ["-xOzf", archive, `liberation-fonts-1.07.4/${name}`], { maxBuffer: 8 << 20 });
  if (!/^VER\s*=\s*1\.07\.4\s*$/m.test(member("Makefile").toString()))
    throw new Error("Font source recipe version mismatch");
  for (const name of Object.keys(fonts)) {
    if (!/^Version: 1\.07\.4\s*$/m.test(member("src/" + name.replace(/\.ttf$/, ".sfd")).toString()))
      throw new Error(`Font source version mismatch: ${name}`);
  }
  // Notices are also available outside the compressed source for offline users.
  for (const name of ["License.txt", "COPYING", "README", "Makefile"])
    await writeFile(path.join(target, name), member(name));
  await copyFile(path.join(pdfDir, "standard_fonts", "LICENSE_LIBERATION"), path.join(target, "PDFJS-LICENSE_LIBERATION"));
  const manifest = {
    format: "madi-pdf-font-sources-v1", pdfjs_version: pkg.version,
    version: "1.07.4", license: "GPL-2.0 with Liberation font exception",
    source: liberationSource, provenance: documentation, files,
    source_repository: "https://github.com/liberationfonts/liberation-1.7-fonts",
    note: "The PDF.js 6.3.289 font files are unchanged. The official version-matched source archive includes editable SFDs, Makefile and FontForge scripts. This is not a claim of bit-for-bit reproducible binaries; historical FontForge output differs. Do not replace these files with Liberation 2.x without updating PDF.js glyph-index mappings and licenses.",
  };
  await writeFile(path.join(target, "manifest.json"), JSON.stringify(manifest, null, 2) + "\n");
  await writeFile(path.join(target, "REBUILD.txt"),
    "PDF.js Liberation Sans 1.07.4 대응 소스\n\n" +
    "공식 원본 liberation-fonts-1.07.4.tar.gz에는 수정 가능한 SFD, Makefile, scripts/fontexport.pe 및 라이선스가 있습니다. 압축을 풀고 FontForge·make가 준비된 빌드 환경에서 make ttf를 실행하면 export/에 TTF가 생성됩니다. 런타임에는 빌드 도구나 네트워크가 필요하지 않습니다.\n\n" +
    "여기에 함께 보존한 네 TTF는 madi가 변경하지 않은 PDF.js 6.3.289 배포 파일과 같습니다. 원본 공개 TTF archive와 byte 단위로 같다고 주장하지 않습니다. FontForge 버전과 빌드에 따라 출력이 달라질 수 있습니다. PDF.js의 glyph 인덱스/폭/배율 매핑에 연결되므로 폰트 교체 시 src/core/liberationsans_widths.js와 *_factors.js도 함께 재생성·시험해야 합니다. 변경 배포 시 원래 GPL/예외·상표 조건을 유지하세요.\n");
  return manifest;
}
