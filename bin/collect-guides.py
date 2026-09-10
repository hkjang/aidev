#!/usr/bin/env python3
"""프로젝트별 가이드를 aidev/docs/guides 아래로 모은다.

각 저장소가 docs/ 에 쓴 사용자·관리자 가이드를 한곳에서 훑어보게 한다. 25개
저장소를 하나씩 열어 보지 않고도 무엇이 나왔는지, 무엇이 아직인지 보인다.

원본은 각 저장소가 계속 갖고, 여기 있는 것은 사본이다. 사본을 고치면 다음
수집에서 덮인다.
"""
from __future__ import annotations

import html
import re
import shutil
import sys
from pathlib import Path

ROOT = Path("/mnt/c/Users/USER/projects")
AIDEV = Path(__file__).resolve().parent.parent
OUT = AIDEV / "docs" / "guides"
SKIP = {"aidev", "Naviq", "sqlpad"}
SKIP_PREFIX = ("_tmp",)
DOCS = [("USER_GUIDE", "사용자 가이드"), ("ADMIN_GUIDE", "관리자 가이드")]
IMAGE = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")


def copy_one(project: Path, target: Path) -> dict:
    """한 저장소의 가이드와 그 가이드가 싣는 그림을 사본으로 옮긴다."""
    found = {}
    for stem, label in DOCS:
        source = project / "docs" / f"{stem}.md"
        if not source.exists():
            continue
        target.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target / f"{stem}.md")
        entry = {"label": label, "md": f"{stem}.md", "pdf": None, "images": 0, "missing": []}
        pdf = source.with_suffix(".pdf")
        if pdf.exists():
            shutil.copy2(pdf, target / f"{stem}.pdf")
            entry["pdf"] = f"{stem}.pdf"
        # 그림은 문서가 가리키는 것만 옮긴다 — docs/ 전체를 복사하면 가이드와
        # 무관한 자료까지 따라와 사이트가 부풀어 오른다.
        for rel in IMAGE.findall(source.read_text(encoding="utf-8", errors="replace")):
            if rel.startswith(("http://", "https://", "data:")):
                continue
            src = (source.parent / rel).resolve()
            if not src.is_file():
                entry["missing"].append(rel)
                continue
            dest = (target / rel).resolve()
            if OUT.resolve() not in dest.parents:
                entry["missing"].append(rel)
                continue
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(src, dest)
            entry["images"] += 1
        found[stem] = entry
    return found


def write_index(collected: dict[str, dict]) -> None:
    rows = []
    for name in sorted(collected, key=str.lower):
        docs = collected[name]
        cells = []
        for stem, label in DOCS:
            entry = docs.get(stem)
            if not entry:
                cells.append('<td class="none">—</td>')
                continue
            links = [f'<a href="{name}/{entry["md"]}">MD</a>']
            if entry["pdf"]:
                links.append(f'<a href="{name}/{entry["pdf"]}">PDF</a>')
            note = f'<span class="shots">그림 {entry["images"]}</span>' if entry["images"] else ""
            cells.append(f'<td>{" · ".join(links)} {note}</td>')
        rows.append(f'<tr><th scope="row">{html.escape(name)}</th>{"".join(cells)}</tr>')

    OUT.mkdir(parents=True, exist_ok=True)
    (OUT / "index.html").write_text(
        """<!doctype html><html lang="ko"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>프로젝트 가이드</title>
<style>
:root{color-scheme:light dark;--ink:#14171a;--muted:#5b6570;--line:#dfe4ea;--accent:#1f5fa8;--bg:#fff}
@media (prefers-color-scheme:dark){:root{--ink:#e8ecf1;--muted:#9aa5b1;--line:#2a3138;--accent:#7fb3ec;--bg:#14171a}}
body{margin:0;padding:32px 16px;background:var(--bg);color:var(--ink);
     font-family:system-ui,-apple-system,"Malgun Gothic",sans-serif;line-height:1.6}
main{max-width:860px;margin:0 auto}
h1{font-size:1.6rem;margin:0 0 6px}
p.lead{color:var(--muted);margin:0 0 24px}
.wrap{overflow-x:auto}
table{border-collapse:collapse;width:100%;font-size:.95rem}
th,td{border-bottom:1px solid var(--line);padding:9px 10px;text-align:left}
thead th{font-size:.8rem;color:var(--muted);text-transform:uppercase;letter-spacing:.04em}
tbody th{font-weight:600;white-space:nowrap}
a{color:var(--accent)}
td.none{color:var(--muted)}
.shots{color:var(--muted);font-size:.8rem;margin-left:4px}
footer{margin-top:28px;color:var(--muted);font-size:.85rem}
</style></head><body><main>
<h1>프로젝트 가이드</h1>
<p class="lead">도커 이미지로 배포되는 프로젝트의 사용자·관리자 가이드 사본입니다. 원본은 각 저장소의 <code>docs/</code> 아래에 있습니다.</p>
<div class="wrap"><table>
<thead><tr><th scope="col">프로젝트</th><th scope="col">사용자 가이드</th><th scope="col">관리자 가이드</th></tr></thead>
<tbody>
"""
        + "\n".join(rows)
        + """
</tbody></table></div>
<footer><a href="../">← aidev 대시보드</a></footer>
</main></body></html>
""",
        encoding="utf-8",
    )


def main() -> int:
    collected: dict[str, dict] = {}
    for project in sorted(ROOT.iterdir()):
        if not project.is_dir() or project.name in SKIP or not (project / ".git").exists():
            continue
        if project.name.startswith(SKIP_PREFIX):
            continue
        docs = copy_one(project, OUT / project.name)
        if docs:
            collected[project.name] = docs
    write_index(collected)
    for name, docs in sorted(collected.items()):
        for stem, entry in docs.items():
            gap = f" 그림 누락 {len(entry['missing'])}건" if entry["missing"] else ""
            print(f"{name}/{stem}: md{' + pdf' if entry['pdf'] else ''}, 그림 {entry['images']}{gap}")
    print(f"가이드 {sum(len(d) for d in collected.values())}건 / 프로젝트 {len(collected)}개")
    return 0


if __name__ == "__main__":
    sys.exit(main())
