const fs = require('fs');
const path = require('path');
const { chromium } = require('/home/hkjang/.cache/auto-improve-wt/kanpic/web/node_modules/playwright-core');

function formatInline(text) {
  if (!text) return '';
  let s = text;
  // Code `code`
  s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
  // Bold **bold**
  s = s.replace(/\*\*([\s\S]+?)\*\*/g, '<strong>$1</strong>');
  // Italic *italic*
  s = s.replace(/(^|[^\*])\*([^\*\s][^\*]*?[^\*\s]|[^\*\s])\*/g, '$1<em>$2</em>');
  // Strikethrough ~~del~~
  s = s.replace(/~~([^~]+)~~/g, '<del>$1</del>');
  return s;
}

function mdToHtml(md) {
  const placeholders = [];

  function addPlaceholder(htmlSnippet) {
    const id = `___PLACEHOLDER_${placeholders.length}___`;
    placeholders.push({ id, htmlSnippet });
    return id;
  }

  let text = md;

  // 1. Extract Mermaid blocks
  text = text.replace(/```mermaid\s*\n([\s\S]*?)```/g, (match, code) => {
    const div = `<div class="mermaid-container"><div class="mermaid">\n${code.trim()}\n</div></div>`;
    return addPlaceholder(div);
  });

  // 2. Extract Code blocks
  text = text.replace(/```(\w*)\s*\n([\s\S]*?)```/g, (match, lang, code) => {
    const escaped = code.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    const pre = `<pre><code class="language-${lang}">${escaped}</code></pre>`;
    return addPlaceholder(pre);
  });

  // 3. Extract Callouts (> [!NOTE], etc.)
  text = text.replace(/^>\s*\[\!(NOTE|TIP|WARNING|IMPORTANT|CAUTION)\]\s*\n((?:^>.*$\n?)+)/gim, (match, type, body) => {
    const cleanBody = body.replace(/^>\s?/gm, '').trim();
    const typeClass = type.toLowerCase();
    const titles = {
      note: '참고 (Note)',
      tip: '팁 (Tip)',
      warning: '경고 (Warning)',
      important: '중요 (Important)',
      caution: '주의 (Caution)'
    };
    const div = `<div class="callout callout-${typeClass}">
      <div class="callout-title">${titles[typeClass] || type}</div>
      <div class="callout-content">${formatInline(cleanBody)}</div>
    </div>`;
    return addPlaceholder(div);
  });

  // 4. Extract Tables
  const lines = text.split('\n');
  let inTable = false;
  let tableBuffer = [];
  let newLines = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].trim();
    if (line.startsWith('|') && line.endsWith('|')) {
      if (!inTable) {
        inTable = true;
        tableBuffer = [line];
      } else {
        tableBuffer.push(line);
      }
    } else {
      if (inTable) {
        const tableHtml = renderTable(tableBuffer);
        newLines.push(addPlaceholder(tableHtml));
        inTable = false;
        tableBuffer = [];
      }
      newLines.push(lines[i]);
    }
  }
  if (inTable) {
    const tableHtml = renderTable(tableBuffer);
    newLines.push(addPlaceholder(tableHtml));
  }
  text = newLines.join('\n');

  // 5. Headings (h6 down to h1) with double newline isolation
  text = text.replace(/^###### (.*$)/gim, (m, g1) => `\n\n<h6>${formatInline(g1)}</h6>\n\n`);
  text = text.replace(/^##### (.*$)/gim, (m, g1) => `\n\n<h5>${formatInline(g1)}</h5>\n\n`);
  text = text.replace(/^#### (.*$)/gim, (m, g1) => `\n\n<h4>${formatInline(g1)}</h4>\n\n`);
  text = text.replace(/^### (.*$)/gim, (m, g1) => `\n\n<h3>${formatInline(g1)}</h3>\n\n`);
  text = text.replace(/^## (.*$)/gim, (m, g1) => `\n\n<h2>${formatInline(g1)}</h2>\n\n`);
  text = text.replace(/^# (.*$)/gim, (m, g1) => `\n\n<h1>${formatInline(g1)}</h1>\n\n`);

  // Horizontal Rule (---, ***, ___)
  text = text.replace(/^\s*[-*_]{3,}\s*$/gm, '\n\n<hr>\n\n');

  // Unordered Lists (*, -, +)
  text = text.replace(/^\s*[-*+]\s+(.*$)/gim, (m, g1) => `<li>${formatInline(g1)}</li>`);
  text = text.replace(/((?:<li>.*?<\/li>\s*)+)/gs, '\n\n<ul>$1</ul>\n\n');

  // Ordered Lists (1., 2., etc.)
  text = text.replace(/^\s*\d+\.\s+(.*$)/gim, (m, g1) => `<li>${formatInline(g1)}</li>`);

  // 6. Paragraph splitting
  const blocks = text.split(/\n\n+/);
  text = blocks.map(block => {
    block = block.trim();
    if (!block) return '';
    if (block.startsWith('<h') || block.startsWith('<ul') || block.startsWith('<ol') || block.startsWith('<hr') || block.startsWith('___PLACEHOLDER_')) {
      return block;
    }
    return `<p>${formatInline(block)}</p>`;
  }).filter(Boolean).join('\n\n');

  // 7. Restore placeholders
  placeholders.forEach(({ id, htmlSnippet }) => {
    text = text.replace(id, htmlSnippet);
  });

  return text;
}

function renderTable(rows) {
  if (rows.length < 2) return '';
  const parseRow = r => r.split('|').slice(1, -1).map(c => c.trim());
  const header = parseRow(rows[0]);
  const bodyRows = rows.slice(2).map(parseRow);

  let html = '<div class="table-container"><table><thead><tr>';
  header.forEach(h => {
    html += `<th>${formatInline(h)}</th>`;
  });
  html += '</tr></thead><tbody>';
  bodyRows.forEach(row => {
    html += '<tr>';
    row.forEach(c => {
      html += `<td>${formatInline(c)}</td>`;
    });
    html += '</tr>';
  });
  html += '</tbody></table></div>';
  return html;
}

async function convertFile(mdPath, pdfPath, title, screenshotPath = null) {
  const mdContent = fs.readFileSync(mdPath, 'utf8');
  const bodyHtml = mdToHtml(mdContent);

  const fullHtml = `
<!DOCTYPE html>
<html lang="ko">
<head>
  <meta charset="UTF-8">
  <title>${title}</title>
  <script src="https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js"></script>
  <style>
    @import url('https://cdn.jsdelivr.net/gh/orioncactus/pretendard/dist/web/static/pretendard.css');
    @page {
      size: A4;
      margin: 18mm 15mm 20mm 15mm;
    }
    body {
      font-family: Pretendard, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      color: #1e293b;
      line-height: 1.65;
      font-size: 12.5px;
      margin: 0;
      padding: 0;
      background: #ffffff;
    }
    header.doc-header {
      border-bottom: 3px solid #0f766e;
      padding-bottom: 10px;
      margin-bottom: 22px;
      display: flex;
      justify-content: space-between;
      align-items: flex-end;
    }
    header.doc-header .brand-title {
      font-size: 24px;
      color: #0f766e;
      margin: 0;
      font-weight: 800;
      letter-spacing: -0.03em;
      display: flex;
      align-items: center;
      gap: 8px;
    }
    header.doc-header .doc-subtitle {
      font-size: 12px;
      color: #64748b;
      font-weight: 600;
    }
    h1 {
      font-size: 20px;
      color: #0f766e;
      margin-top: 26px;
      margin-bottom: 12px;
      font-weight: 800;
      border-bottom: 1px solid #e2e8f0;
      padding-bottom: 6px;
      letter-spacing: -0.02em;
      page-break-after: avoid;
    }
    h2 {
      font-size: 16px;
      color: #0f766e;
      margin-top: 20px;
      margin-bottom: 10px;
      font-weight: 700;
      letter-spacing: -0.01em;
      page-break-after: avoid;
    }
    h3 {
      font-size: 13.5px;
      color: #334155;
      margin-top: 16px;
      margin-bottom: 8px;
      font-weight: 700;
      page-break-after: avoid;
    }
    h4 {
      font-size: 13px;
      color: #0f766e;
      margin-top: 14px;
      margin-bottom: 6px;
      font-weight: 700;
      page-break-after: avoid;
    }
    h5, h6 {
      font-size: 12.5px;
      color: #334155;
      margin-top: 12px;
      margin-bottom: 6px;
      font-weight: 700;
      page-break-after: avoid;
    }
    p {
      margin: 8px 0;
      word-break: keep-all;
    }
    ul, ol {
      padding-left: 20px;
      margin: 8px 0;
    }
    li {
      margin-bottom: 4px;
      word-break: keep-all;
    }
    li strong {
      color: #0f766e;
      font-weight: 700;
    }
    code {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      background: #f1f5f9;
      color: #0f766e;
      padding: 2px 6px;
      border-radius: 4px;
      font-size: 11px;
      border: 1px solid #e2e8f0;
    }
    pre {
      background: #0f172a;
      color: #f8fafc;
      padding: 12px 16px;
      border-radius: 8px;
      overflow-x: auto;
      font-size: 11px;
      line-height: 1.5;
      page-break-inside: avoid;
      box-shadow: inset 0 0 0 1px rgba(255,255,255,0.1);
    }
    pre code {
      background: transparent;
      color: inherit;
      padding: 0;
      border: none;
    }
    .table-container {
      margin: 18px 0;
      border-radius: 8px;
      overflow: hidden;
      border: 1px solid #cbd5e1;
      box-shadow: 0 1px 3px rgba(0,0,0,0.04);
      page-break-inside: avoid;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 11.5px;
      background: #ffffff;
    }
    th, td {
      border: 1px solid #cbd5e1;
      padding: 9px 13px;
      text-align: left;
      vertical-align: top;
      line-height: 1.5;
    }
    th {
      background: #f0fdf4;
      color: #0f766e;
      font-weight: 700;
      border-bottom: 2px solid #0f766e;
      font-size: 12px;
    }
    tr:nth-child(even) {
      background: #f8fafc;
    }
    td strong {
      color: #0f766e;
      font-weight: 700;
    }
    td code {
      font-size: 10.5px;
    }
    hr {
      border: 0;
      height: 1px;
      background: #cbd5e1;
      margin: 22px 0;
    }
    strong {
      color: #0f172a;
      font-weight: 700;
    }
    .callout {
      padding: 12px 16px;
      border-radius: 8px;
      margin: 14px 0;
      page-break-inside: avoid;
      font-size: 12px;
    }
    .callout-note {
      background: #f0fdfa;
      border-left: 4px solid #0f766e;
      color: #115e59;
    }
    .callout-tip {
      background: #f0fdf4;
      border-left: 4px solid #16a34a;
      color: #166534;
    }
    .callout-warning {
      background: #fffbeb;
      border-left: 4px solid #f59e0b;
      color: #92400e;
    }
    .callout-title {
      font-weight: 700;
      margin-bottom: 4px;
    }
    .mermaid-container {
      display: flex;
      justify-content: center;
      margin: 18px 0;
      padding: 16px;
      background: #f8fafc;
      border: 1px solid #e2e8f0;
      border-radius: 10px;
      page-break-inside: avoid;
    }
    .mermaid {
      width: 100%;
      text-align: center;
    }
  </style>
</head>
<body>
  <header class="doc-header">
    <div class="brand-title">
      <span>kanpic</span>
    </div>
    <div class="doc-subtitle">${title} | Enterprise Official Document</div>
  </header>
  <div class="content">
    ${bodyHtml}
  </div>

  <script>
    mermaid.initialize({
      startOnLoad: false,
      theme: 'neutral',
      securityLevel: 'loose',
      fontFamily: 'Pretendard, sans-serif'
    });
  </script>
</body>
</html>
  `;

  const browser = await chromium.launch({executablePath:'/home/hkjang/.cache/ms-playwright/chromium-1228/chrome-linux64/chrome'});
  const page = await browser.newPage({ viewport: { width: 1000, height: 1400 } });
  
  page.on('console', msg => console.log('PAGE LOG:', msg.text()));
  page.on('pageerror', err => console.log('PAGE ERROR:', err));

  await page.setContent(fullHtml, { waitUntil: 'networkidle' });

  // Run mermaid explicitly in page context
  await page.evaluate(async () => {
    if (window.mermaid) {
      await window.mermaid.run();
    }
  });

  // Wait until all .mermaid divs contain <svg>
  try {
    await page.waitForFunction(() => {
      const mermaids = document.querySelectorAll('.mermaid');
      if (mermaids.length === 0) return true;
      return Array.from(mermaids).every(m => m.querySelector('svg') !== null);
    }, { timeout: 8000 });
  } catch (e) {
    console.warn('Warning: Some mermaid diagrams may not have rendered SVG:', e.message);
  }

  if (screenshotPath) {
    await page.screenshot({ path: screenshotPath, fullPage: true });
  }

  await page.pdf({
    path: pdfPath,
    format: 'A4',
    margin: { top: '16mm', bottom: '20mm', left: '15mm', right: '15mm' },
    printBackground: true,
    displayHeaderFooter: true,
    footerTemplate: `
      <div style="font-size: 9px; color: #94a3b8; width: 100%; padding: 0 15mm; display: flex; justify-content: space-between; font-family: Pretendard, sans-serif;">
        <span>kanpic Platform Official Documentation</span>
        <span>Page <span class="pageNumber"></span> of <span class="totalPages"></span></span>
      </div>
    `
  });
  await browser.close();
  console.log(`Generated high-quality PDF: ${pdfPath}`);
}

async function main() {
  const docsDir = '/home/hkjang/.cache/auto-improve-wt/kanpic/docs';
  await convertFile(path.join(docsDir, 'USER_GUIDE.md'), path.join(docsDir, 'USER_GUIDE.pdf'), '사용자 가이드 (User Guide)', null);

}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
