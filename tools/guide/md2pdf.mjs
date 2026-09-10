#!/usr/bin/env node
// Markdown 가이드를 프로젝트 공통 서식의 PDF 로 굽는다.
//
// 24개 저장소가 저마다 변환기를 만들면 서식이 24가지가 된다. 표지·머리글·쪽번호·
// 그림 배치를 여기 한 곳에 두고, 각 저장소는 Markdown 만 쓴다.
//
//   node tools/guide/md2pdf.mjs <입력.md> <출력.pdf> [--title T] [--subtitle S] [--project P] [--version V]
//
// --keep-html <경로> 를 주면 중간 HTML 을 남긴다. 서식이 깨졌을 때 브라우저로 열어 본다.
//
// 그림 경로는 입력 파일 기준으로 풉니다. 없는 그림은 조용히 비우지 않고 실패합니다 —
// 캡처가 빠진 가이드가 캡처가 있는 척 배포되는 편이 더 나쁩니다.
import { readFileSync, writeFileSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { dirname, resolve, join, extname } from 'node:path';
import { execFileSync } from 'node:child_process';
import { tmpdir } from 'node:os';
import { marked } from 'marked';

const args = process.argv.slice(2);
const positional = args.filter((a) => !a.startsWith('--'));
const flag = (name, fallback = '') => {
  const i = args.indexOf(`--${name}`);
  return i >= 0 && args[i + 1] ? args[i + 1] : fallback;
};
if (positional.length < 2) {
  console.error('사용법: md2pdf.mjs <입력.md> <출력.pdf> [--title T] [--subtitle S] [--project P] [--version V]');
  process.exit(2);
}
const [input, output] = positional.map((p) => resolve(p));
const base = dirname(input);
const source = readFileSync(input, 'utf8');

// 그림을 data URI 로 심는다. headless Chrome 은 file:// 문서에서 다른 file:// 을
// 읽지 못하는 설정이 흔해, 경로로 두면 조용히 빈 칸이 된다.
const MIME = { '.png': 'image/png', '.jpg': 'image/jpeg', '.jpeg': 'image/jpeg', '.gif': 'image/gif', '.webp': 'image/webp', '.svg': 'image/svg+xml' };
const missing = [];
const renderer = new marked.Renderer();
const originalImage = renderer.image.bind(renderer);
renderer.image = (token) => {
  const href = typeof token === 'object' ? token.href : token;
  if (/^(https?:|data:)/.test(href)) return originalImage(token);
  const file = resolve(base, href);
  const type = MIME[extname(file).toLowerCase()];
  if (!existsSync(file) || !type) { missing.push(href); return originalImage(token); }
  const uri = `data:${type};base64,${readFileSync(file).toString('base64')}`;
  const alt = (typeof token === 'object' ? token.text : '') || '';
  const caption = alt ? `<figcaption>${alt}</figcaption>` : '';
  return `<figure><img src="${uri}" alt="${alt}">${caption}</figure>`;
};

const body = marked.parse(source, { renderer, gfm: true, breaks: false });
if (missing.length) {
  console.error(`그림을 찾지 못했습니다 (${missing.length}건): ${missing.join(', ')}`);
  console.error('가이드는 실제로 찍은 화면만 싣습니다. 캡처를 먼저 만들고 다시 실행하세요.');
  process.exit(1);
}

const title = flag('title', '사용자 가이드');
const subtitle = flag('subtitle', '');
const project = flag('project', '');
const version = flag('version', '');
const today = new Date().toISOString().slice(0, 10);
const esc = (s) => s.replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' }[c]));

const html = `<!doctype html><html lang="ko"><head><meta charset="utf-8"><title>${esc(title)}</title><style>
@page { size: A4; margin: 20mm 18mm 22mm; }
:root { --ink:#14171a; --muted:#5b6570; --line:#dfe4ea; --accent:#1f5fa8; --code-bg:#f6f8fa; }
* { box-sizing: border-box; }
body { font-family: "NanumBarunGothic","Noto Sans CJK KR","Malgun Gothic",system-ui,sans-serif;
       color: var(--ink); font-size: 10.5pt; line-height: 1.65; margin: 0; }
.cover { height: 247mm; display: flex; flex-direction: column; justify-content: center; page-break-after: always; }
.cover .eyebrow { color: var(--accent); font-weight: 700; letter-spacing: .08em; font-size: 11pt; }
.cover h1 { font-size: 30pt; line-height: 1.25; margin: 12pt 0 6pt; }
.cover .sub { color: var(--muted); font-size: 13pt; margin-bottom: 28pt; }
.cover dl { display: grid; grid-template-columns: max-content 1fr; gap: 4pt 16pt; margin: 0;
            border-top: 2px solid var(--line); padding-top: 14pt; font-size: 10pt; }
.cover dt { color: var(--muted); } .cover dd { margin: 0; }
h1,h2,h3,h4 { line-height: 1.35; page-break-after: avoid; }
h1 { font-size: 19pt; border-bottom: 2px solid var(--line); padding-bottom: 5pt; margin: 26pt 0 12pt; }
h2 { font-size: 14.5pt; margin: 20pt 0 8pt; }
h3 { font-size: 12pt; margin: 15pt 0 6pt; }
h4 { font-size: 10.5pt; color: var(--muted); margin: 12pt 0 4pt; }
p, li { orphans: 3; widows: 3; }
ul, ol { padding-left: 20pt; }
code { font-family: "NanumGothicCoding","D2Coding",ui-monospace,monospace; font-size: 9.5pt;
       background: var(--code-bg); padding: 1pt 3pt; border-radius: 3px; }
pre { background: var(--code-bg); border: 1px solid var(--line); border-radius: 5px; padding: 9pt 11pt;
      overflow-x: auto; page-break-inside: avoid; }
pre code { background: none; padding: 0; font-size: 9pt; line-height: 1.5; }
table { border-collapse: collapse; width: 100%; margin: 10pt 0; font-size: 9.5pt; page-break-inside: avoid; }
th, td { border: 1px solid var(--line); padding: 5pt 7pt; text-align: left; vertical-align: top; }
th { background: var(--code-bg); font-weight: 700; }
blockquote { margin: 10pt 0; padding: 7pt 12pt; border-left: 3px solid var(--accent);
             background: #f4f8fd; color: var(--ink); page-break-inside: avoid; }
blockquote p { margin: 0; }
figure { margin: 14pt 0; page-break-inside: avoid; text-align: center; }
figure img { max-width: 100%; border: 1px solid var(--line); border-radius: 5px; }
figcaption { color: var(--muted); font-size: 9pt; margin-top: 5pt; }
hr { border: none; border-top: 1px solid var(--line); margin: 18pt 0; }
a { color: var(--accent); text-decoration: none; word-break: break-all; }
</style></head><body>
<section class="cover">
  ${project ? `<div class="eyebrow">${esc(project)}</div>` : ''}
  <h1>${esc(title)}</h1>
  ${subtitle ? `<div class="sub">${esc(subtitle)}</div>` : ''}
  <dl>
    ${version ? `<dt>버전</dt><dd>${esc(version)}</dd>` : ''}
    <dt>작성일</dt><dd>${today}</dd>
    <dt>문서</dt><dd>${esc(title)}</dd>
  </dl>
</section>
${body}
</body></html>`;

const work = join(tmpdir(), `guide-${process.pid}`);
mkdirSync(work, { recursive: true });
const page = flag('keep-html') ? resolve(flag('keep-html')) : join(work, 'guide.html');
writeFileSync(page, html);
mkdirSync(dirname(output), { recursive: true });

const chrome = ['google-chrome', 'chromium', 'chromium-browser', 'google-chrome-stable'].find((c) => {
  try { execFileSync('command', ['-v', c], { shell: true, stdio: 'ignore' }); return true; } catch { return false; }
});
if (!chrome) { console.error('Chrome/Chromium 을 찾지 못했습니다.'); process.exit(1); }
try {
  execFileSync(chrome, ['--headless=new', '--no-sandbox', '--disable-gpu',
    `--print-to-pdf=${output}`, '--no-pdf-header-footer', `file://${page}`],
    { stdio: ['ignore', 'ignore', 'pipe'], timeout: 180000 });
} catch (error) {
  if (!existsSync(output)) { console.error(`PDF 생성 실패: ${error.message}`); process.exit(1); }
}
rmSync(work, { recursive: true, force: true });
if (!existsSync(output)) { console.error('PDF 가 만들어지지 않았습니다.'); process.exit(1); }
console.log(`${output} (${(readFileSync(output).length / 1024).toFixed(0)}KB)`);
