#!/usr/bin/env python3
"""docs/ 아래 GitHub Pages 용 일일 보고를 만든다.

입력  docs/data/runs.jsonl   — 러너가 회차마다 한 줄씩 남기는 기록
      docs/data/usage.jsonl  — 회차별 claude 사용량 (비용·토큰·시간)
      state/<프로젝트>.md    — 에이전트 원장 (그날 항목을 보고에 인용)
      state/<프로젝트>.release.json  — 마지막 릴리즈 (tag, assets_count, prev_assets_count …)
      state/fix-queue.tsv    — 수정 과제 큐
출력  docs/index.md            — 대시보드 (주의 필요, 오늘, 14일 차트, 일일/주간/월간 보고, 프로젝트별 현황, 비용, FAQ)
      docs/reports/<날짜>.md   — 일일 보고
      docs/weekly/<YYYY-Www>.md, docs/monthly/<YYYY-MM>.md — 주간·월간 보고
      docs/projects/<이름>.md  — 프로젝트 페이지 (원장 전체, 릴리즈, 회차 이력)
      docs/feed.xml            — Atom 피드 (일일 보고)
      docs/data/summary.json   — 요약 JSON API
러너가 회차 끝마다 호출하므로 항상 그날 보고가 최신이다. 인자 없이 실행하면 전체를 다시 만든다.

표는 HTML 로 직접 만든다 — 각 셀에 data-label 을 달아 좁은 화면에서 카드로 접히게 하고(style.css),
행에 data-status 를 달아 레이아웃의 필터가 상태 칩으로 거를 수 있게 한다. 본문은 Liquid raw 로 감싼다.
"""
import html
import json
import os
import re
import subprocess
from collections import defaultdict
from datetime import date, datetime, timedelta

REPO_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
ROOT = os.environ.get("ROOT", "/mnt/c/Users/USER/projects")
DOCS = os.path.join(REPO_DIR, "docs")
DATA = os.path.join(DOCS, "data", "runs.jsonl")
USAGE = os.path.join(DOCS, "data", "usage.jsonl")
STATE = os.path.join(REPO_DIR, "state")
REPORTS = os.path.join(DOCS, "reports")
PROJECTS = os.path.join(DOCS, "projects")
SITE = "https://hkjang.github.io/aidev"
GH = "https://github.com/hkjang"
NOW = datetime.now().astimezone()
RAW_OPEN, RAW_CLOSE = "{% raw %}", "{% endraw %}"

STATUS = {"released": "🚀 릴리즈", "merged": "✅ 머지", "nochange": "➖ 변경 없음", "failed": "❌ 실패", "other": "• 기타"}
# 상태 팔레트 (dataviz 참조 팔레트의 status 슬롯; 회색은 중립) — 아이콘·글자와 항상 함께 쓴다
COLOR = {"released": "#0ca30c", "merged": "#fab219", "nochange": "#9ca3af", "failed": "#d03b3b"}
WARN_PATTERNS = [
    ("ASSETS MISSING", "릴리즈 자산 누락 — 이전 릴리즈엔 있던 파일이 이번엔 없음"),
    ("queued for fix", "릴리즈 워크플로 2회 실패 — 수정 과제 배정됨"),
    ("CI failed", "CI 실패로 PR 미머지"),
    ("merge failed", "PR 머지 실패"),
    ("release push failed", "릴리즈 푸시 실패"),
    ("release failed", "릴리즈 단계 실패"),
    ("release missing", "릴리즈 에이전트가 결과를 남기지 못함"),
    ("assets failed", "자산 빌드 실패"),
    ("assets missing", "자산 빌드 결과 없음"),
    ("asset upload failed", "자산 업로드 실패"),
]


# ---------- 기본 유틸 ----------
def classify(result):
    r = result or ""
    if re.search(r"released v?\d", r):
        return "released"
    if "merge failed" in r or "release push failed" in r or "CI failed" in r:
        return "failed"
    if "merged" in r:
        return "merged"
    if "no change" in r or "변경없음" in r:
        return "nochange"
    return "other"


_repo_cache = {}


def repo_name(project, result=""):
    m = re.search(r"github\.com/hkjang/([^/]+)/pull", result or "")
    if m:
        _repo_cache.setdefault(project, m.group(1))
        return m.group(1)
    if project in _repo_cache:
        return _repo_cache[project]
    name = project
    try:
        url = subprocess.check_output(["git", "-C", os.path.join(ROOT, project), "remote", "get-url", "origin"],
                                      text=True, stderr=subprocess.DEVNULL).strip()
        name = re.sub(r"\.git$", "", url.rstrip("/").split("/")[-1])
    except Exception:
        pass
    _repo_cache[project] = name
    return name


def released_tag(result):
    m = re.search(r"released (v?[0-9][\w.\-]*)", result or "")
    return m.group(1) if m else ""


def esc(s):
    return html.escape(str(s), quote=True)


def result_html(project, result):
    out = esc(result or "")
    out = re.sub(r"(https://github\.com/\S+?/pull/(\d+))", r'<a href="\1">PR #\2</a>', out)
    tag = released_tag(result)
    if tag:
        out = out.replace(f"released {esc(tag)}",
                          f'released <a href="{GH}/{repo_name(project, result)}/releases/tag/{esc(tag)}">{esc(tag)}</a>')
    for key, _ in WARN_PATTERNS:
        out = out.replace(esc(key), f"<strong>{esc(key)}</strong>")
    return out


def pill(status):
    return f'<span class="pill pill-{status}">{STATUS[status]}</span>'


def ts_hm(r):
    return (r.get("ts") or "")[11:16]


def load_jsonl(path):
    rows = []
    if not os.path.exists(path):
        return rows
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                try:
                    rows.append(json.loads(line))
                except json.JSONDecodeError:
                    pass
    return rows


def ledger_text(project):
    path = os.path.join(STATE, f"{project}.md")
    return open(path, encoding="utf-8").read() if os.path.exists(path) else ""


def ledger_entry(project, day):
    m = re.search(rf"^## {re.escape(day)}\s*$(.*?)(?=^## |\Z)", ledger_text(project), re.M | re.S)
    return m.group(1).strip() if m else ""


def release_info(project):
    path = os.path.join(STATE, f"{project}.release.json")
    try:
        return json.load(open(path, encoding="utf-8"))
    except Exception:
        return {}


def fix_queue():
    path = os.path.join(STATE, "fix-queue.tsv")
    out = []
    if os.path.exists(path):
        for line in open(path, encoding="utf-8"):
            if "\t" in line:
                p, note = line.rstrip("\n").split("\t", 1)
                out.append({"project": p, "note": note})
    return out


def counts(day_runs):
    c = defaultdict(int)
    for r in day_runs:
        c[classify(r.get("result"))] += 1
    c["total"] = len(day_runs)
    c["projects"] = len({r.get("project") for r in day_runs})
    return c


def usage_sum(rows):
    u = defaultdict(float)
    for r in rows:
        u["cost"] += float(r.get("cost_usd") or 0)
        u["minutes"] += float(r.get("duration_ms") or 0) / 60000
        u["turns"] += int(r.get("num_turns") or 0)
        u["in"] += int(r.get("input_tokens") or 0) + int(r.get("cache_read") or 0) + int(r.get("cache_create") or 0)
        u["out"] += int(r.get("output_tokens") or 0)
        u["sessions"] += 1
    return u


def fmt_min(m):
    m = int(round(m))
    return f"{m//60}시간 {m%60}분" if m >= 60 else f"{m}분"


def fmt_tok(n):
    n = int(n)
    return f"{n/1e6:.1f}M" if n >= 1e6 else (f"{n/1e3:.0f}K" if n >= 1e3 else str(n))


def yaml_str(s):
    return json.dumps(s, ensure_ascii=False)


def front_matter(title, description, day=None, extra=None):
    fm = ["---", f"title: {yaml_str(title)}", f"description: {yaml_str(description)}"]
    if day:
        fm.append(f"date: {day}")
    fm.append(f"last_modified_at: {NOW.strftime('%Y-%m-%d %H:%M:%S %z')}")
    for k, v in (extra or {}).items():
        fm.append(f"{k}: {v}")
    fm.append("---\n" + RAW_OPEN)
    return "\n".join(fm)


def write_page(path, lines):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        f.write("\n".join(lines) + "\n" + RAW_CLOSE + "\n")


def jsonld(obj):
    return '<script type="application/ld+json">\n' + json.dumps(obj, ensure_ascii=False, indent=1) + "\n</script>\n"


def stats_html(c, usage=None):
    items = [("회차", c["total"]), ("프로젝트", c["projects"]), ("릴리즈", c["released"]),
             ("머지(릴리즈 없음)", c["merged"]), ("변경 없음", c["nochange"]), ("실패", c["failed"])]
    if usage and usage["sessions"]:
        items += [("비용", f"${usage['cost']:.2f}"), ("에이전트 시간", fmt_min(usage["minutes"]))]
    return '<ul class="stats">' + "".join(f"<li><b>{v}</b><span>{k}</span></li>" for k, v in items) + "</ul>\n"


def table(headers, rows, filterable=False, caption=None):
    th = "".join(f"<th{(' class=' + chr(34) + c + chr(34)) if c else ''}>{esc(h)}</th>" for h, c in headers)
    body = []
    for status, cells in rows:
        tds = []
        for i, cell in enumerate(cells):
            cls = headers[i][1]
            attrs = f' data-label="{esc(headers[i][0])}"' + (f' class="{cls}"' if cls else "")
            tds.append(f"<td{attrs}>{cell}</td>")
        body.append(f'<tr data-status="{status}">' + "".join(tds) + "</tr>")
    cap = f"<caption class=\"meta\">{esc(caption)}</caption>" if caption else ""
    df = ' data-filter="1"' if filterable else ""
    return (f'<div class="table-wrap"><table class="rt"{df}>{cap}<thead><tr>{th}</tr></thead>'
            f'<tbody>{"".join(body)}</tbody></table></div>\n')


def summary_sentence(label, c, runs_, usage=None):
    rel = [f"{r['project']} {released_tag(r.get('result'))}" for r in runs_ if classify(r.get("result")) == "released"]
    s = (f"{label} 자율 개선 에이전트가 {c['projects']}개 프로젝트에서 {c['total']}회차를 돌려 "
         f"{c['released']}건을 릴리즈하고 {c['merged']}건은 머지만 했으며 {c['nochange']}건은 변경이 없었고 실패는 {c['failed']}건이다.")
    if usage and usage["sessions"]:
        s += f" 에이전트 시간 {fmt_min(usage['minutes'])}, 추정 비용 ${usage['cost']:.2f}."
    if rel:
        s += " 릴리즈: " + ", ".join(rel[:8]) + ("…" if len(rel) > 8 else "") + "."
    return s


# ---------- 주의 필요 ----------
def live_release(project, tag):
    """(자산 수, 태그 워크플로 결론) — 경고가 아직 유효한지 GitHub 에서 다시 본다."""
    repo = repo_name(project)
    try:
        n = int(subprocess.check_output(["gh", "release", "view", tag, "-R", f"hkjang/{repo}", "--json", "assets", "--jq", ".assets|length"],
                                        text=True, stderr=subprocess.DEVNULL, timeout=30).strip())
    except Exception:
        n = None
    try:
        wf = subprocess.check_output(["gh", "run", "list", "-R", f"hkjang/{repo}", "--limit", "40", "--json", "headBranch,conclusion,name",
                                      "--jq", f'[.[] | select(.headBranch=="{tag}")] | .[0] | "\\(.name): \\(.conclusion)"'],
                                     text=True, stderr=subprocess.DEVNULL, timeout=30).strip()
    except Exception:
        wf = ""
    return n, wf


def alerts(by_day, days):
    out = []
    for d in days[:2]:
        for r in by_day[d]:
            res = r.get("result") or ""
            for key, why in WARN_PATTERNS:
                if key in res:
                    if key == "ASSETS MISSING" and released_tag(res):
                        n, wf = live_release(r.get("project"), released_tag(res))
                        if n:
                            break
                        if n is None:
                            why = f"릴리즈 {released_tag(res)} 가 GitHub 에 없음" + (f" — 워크플로 {wf}" if wf else "")
                        elif wf:
                            why = f"릴리즈 {released_tag(res)} 자산 0개 — 워크플로 {wf}"
                    out.append({"date": d, "time": ts_hm(r), "project": r.get("project"), "why": why, "result": res})
                    break
    for p in sorted({r.get("project") for d in days[:3] for r in by_day[d]}):
        rel = release_info(p)
        if rel.get("status") == "released" and (rel.get("prev_assets_count") or 0) > 0 and rel.get("assets_count", 0) == 0:
            out.append({"date": "", "time": "", "project": p, "why": f"최신 릴리즈 {rel.get('tag')} 자산 0개 (이전 {rel.get('prev_tag')}: {rel.get('prev_assets_count')}개)", "result": ""})
    for q in fix_queue():
        out.append({"date": "", "time": "", "project": q["project"], "why": "수정 과제 대기 중 — " + q["note"][:160], "result": ""})
    return out


def alerts_html(items):
    if not items:
        return '<div class="alerts ok" role="status"><strong>✅ 주의 필요 없음</strong> — 최근 2일 회차에 실패·누락이 없습니다.</div>\n'
    lis = "".join(
        f'<li><a href="{SITE}/projects/{esc(a["project"])}/">{esc(a["project"])}</a> — {esc(a["why"])}'
        f'{(" <span class=meta>(" + esc(a["date"] + " " + a["time"]) + ")</span>") if a["date"] else ""}</li>' for a in items)
    return (f'<div class="alerts" role="alert"><strong>⚠️ 주의 필요 {len(items)}건</strong> '
            f'<span class="meta">— 새 경고는 GitHub Issue·Slack·이메일·Windows 알림으로도 보냅니다</span><ul>{lis}</ul></div>\n')


# ---------- 차트 (인라인 SVG, 누적 막대) ----------
def chart_svg(by_day, days=None, title="최근 14일 회차 수"):
    days = days or [(date.today() - timedelta(days=i)).isoformat() for i in range(13, -1, -1)]
    order = ["released", "merged", "nochange", "failed"]
    W, H, L, B, T = 640, 200, 28, 28, 10
    slot = (W - L - 8) / max(len(days), 1)
    bw = min(24, slot * 0.6)
    maxv = max([counts(by_day.get(d, []))["total"] for d in days] + [1])
    step = 10 if maxv > 30 else (5 if maxv > 10 else (2 if maxv > 4 else 1))
    maxv = ((maxv + step - 1) // step) * step
    scale = (H - B - T) / maxv
    parts = [f'<svg viewBox="0 0 {W} {H}" role="img" aria-labelledby="chart-t chart-d">',
             f'<title id="chart-t">{esc(title)}</title>',
             '<desc id="chart-d">날짜별 회차 수를 릴리즈·머지·변경 없음·실패로 나눠 쌓은 막대. 같은 값은 아래 표에 있다.</desc>']
    for v in range(0, maxv + 1, step):
        y = H - B - v * scale
        parts.append(f'<line class="grid" x1="{L}" x2="{W-4}" y1="{y:.1f}" y2="{y:.1f}"/><text x="{L-6}" y="{y+4:.1f}" text-anchor="end">{v}</text>')
    every = 1 if len(days) <= 8 else (2 if len(days) <= 16 else 5)
    for i, d in enumerate(days):
        c = counts(by_day.get(d, []))
        x = L + i * slot + (slot - bw) / 2
        y = H - B
        tip = f"{d}: 회차 {c['total']} · 릴리즈 {c['released']} · 머지 {c['merged']} · 변경 없음 {c['nochange']} · 실패 {c['failed']}"
        parts.append(f'<g><title>{esc(tip)}</title>')
        segs = [(k, c[k]) for k in order if c[k] > 0]
        for j, (k, v) in enumerate(segs):
            h = v * scale
            y -= h
            if j == len(segs) - 1:
                r = min(4, h / 2)
                path = (f"M{x:.1f},{y+h:.1f} V{y+r:.1f} Q{x:.1f},{y:.1f} {x+r:.1f},{y:.1f} H{x+bw-r:.1f} "
                        f"Q{x+bw:.1f},{y:.1f} {x+bw:.1f},{y+r:.1f} V{y+h:.1f} Z")
                parts.append(f'<path d="{path}" fill="{COLOR[k]}"/>')
            else:
                parts.append(f'<rect x="{x:.1f}" y="{y:.1f}" width="{bw:.1f}" height="{max(h-2,0):.1f}" fill="{COLOR[k]}"/>')
        if c["total"] > 0 and (i == len(days) - 1 or c["total"] == maxv):
            parts.append(f'<text x="{x+bw/2:.1f}" y="{y-4:.1f}" text-anchor="middle">{c["total"]}</text>')
        if i % every == (len(days) - 1) % every:
            parts.append(f'<text x="{x+bw/2:.1f}" y="{H-B+14}" text-anchor="middle">{d[5:].replace("-", "/")}</text>')
        parts.append("</g>")
    parts.append(f'<line class="grid" x1="{L}" x2="{W-4}" y1="{H-B}" y2="{H-B}"/></svg>')
    legend = "".join(f'<li><i style="background:{COLOR[k]}"></i>{STATUS[k]}</li>' for k in order)
    return f'<div class="chart">{"".join(parts)}<ul class="legend" aria-hidden="true">{legend}</ul></div>\n'


# ---------- 공통 표 ----------
def runs_table(runs_, with_date=False, filterable=True, caption=None):
    rows = []
    for r in runs_:
        p = r.get("project", "")
        when = (r.get("ts") or "")[:16].replace("T", " ") if with_date else ts_hm(r)
        rows.append((classify(r.get("result")),
                     [esc(when), f'<a href="{SITE}/projects/{esc(p)}/">{esc(p)}</a>',
                      pill(classify(r.get("result"))) + " " + result_html(p, r.get("result"))]))
    return table([("일시" if with_date else "시각", ""), ("프로젝트", "primary"), ("결과", "")], rows, filterable=filterable, caption=caption)


def usage_table(rows, caption=None):
    out = []
    for u in rows:
        out.append(("other", [esc((u.get("ts") or "")[11:16]), f'<a href="{SITE}/projects/{esc(u.get("project",""))}/">{esc(u.get("project",""))}</a>',
                              esc({"improve": "개선", "release": "릴리즈", "assets": "자산"}.get(u.get("phase"), u.get("phase", ""))),
                              fmt_min(float(u.get("duration_ms") or 0) / 60000), str(u.get("num_turns") or 0),
                              f"${float(u.get('cost_usd') or 0):.2f}",
                              fmt_tok(int(u.get("input_tokens") or 0) + int(u.get("cache_read") or 0) + int(u.get("cache_create") or 0)) + " / " + fmt_tok(u.get("output_tokens") or 0),
                              esc(u.get("subtype", ""))]))
    return table([("시각", ""), ("프로젝트", "primary"), ("단계", ""), ("시간", "num"), ("턴", "num"), ("비용", "num"), ("토큰 입력/출력", "num"), ("종료", "")],
                 out, caption=caption)


def project_release_table(runs_):
    byp = defaultdict(list)
    for r in runs_:
        byp[r.get("project", "")].append(r)
    rows = []
    for p in sorted(byp, key=str.lower):
        c = counts(byp[p])
        tags = [released_tag(r.get("result")) for r in byp[p] if classify(r.get("result")) == "released"]
        repo = repo_name(p, byp[p][-1].get("result"))
        tag_html = ", ".join(f'<a href="{GH}/{repo}/releases/tag/{esc(t)}">{esc(t)}</a>' for t in tags)
        st = "failed" if c["failed"] else ("released" if c["released"] else ("merged" if c["merged"] else "nochange"))
        rows.append((st, [f'<a href="{SITE}/projects/{esc(p)}/">{esc(p)}</a>', str(c["total"]), str(c["released"]), str(c["merged"]), str(c["failed"]), tag_html or "—"]))
    return table([("프로젝트", "primary"), ("회차", "num"), ("릴리즈", "num"), ("머지", "num"), ("실패", "num"), ("릴리즈 태그", "")], rows, filterable=True)


# ---------- 일일 보고 ----------
def write_report(day, day_runs, day_usage):
    c = counts(day_runs)
    u = usage_sum(day_usage)
    desc = summary_sentence(f"{day}에", c, day_runs, u)
    url = f"{SITE}/reports/{day}/"
    ld = {"@context": "https://schema.org", "@type": "Report", "headline": f"자율 개선 일일 보고 {day}", "name": f"자율 개선 일일 보고 {day}",
          "description": desc, "url": url, "inLanguage": "ko", "datePublished": day, "dateModified": NOW.isoformat(timespec="seconds"),
          "author": {"@type": "Person", "name": "hkjang", "url": GH}, "publisher": {"@type": "Organization", "name": "aidev", "url": SITE},
          "isPartOf": {"@type": "WebSite", "name": "aidev 자율 개선 대시보드", "url": SITE},
          "about": [{"@type": "SoftwareSourceCode", "name": p, "codeRepository": f"{GH}/{repo_name(p, r.get('result'))}"}
                    for p, r in {r.get("project"): r for r in day_runs}.items() if p]}
    crumbs = {"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": [
        {"@type": "ListItem", "position": 1, "name": "대시보드", "item": SITE + "/"},
        {"@type": "ListItem", "position": 2, "name": f"일일 보고 {day}", "item": url}]}
    lines = [front_matter(f"자율 개선 일일 보고 {day}", desc, day), jsonld(ld), jsonld(crumbs),
             f"# 자율 개선 일일 보고 — {day}\n", f'<p class="tldr"><strong>요약.</strong> {esc(desc)}</p>\n', stats_html(c, u),
             "## 회차\n", runs_table(day_runs, caption="시각은 KST. 상태 칩과 검색으로 거를 수 있습니다.")]
    if day_usage:
        lines += ["## 비용·사용량\n", usage_table(day_usage, caption="claude -p 가 보고한 추정 비용(정액제에서는 참고값)과 소요 시간")]
    seen, detail = set(), []
    for r in day_runs:
        p = r.get("project", "")
        if p in seen:
            continue
        seen.add(p)
        entry = ledger_entry(p, day)
        if entry:
            detail.append(f"### [{p}]({SITE}/projects/{p}/)\n\n{entry}\n")
    if detail:
        lines.append("## 무엇을 왜 바꿨나 (원장 발췌)\n")
        lines.extend(detail)
    lines.append(f"\n[← 대시보드]({SITE}/) · [Atom 피드]({SITE}/feed.xml) · 원본 데이터 [runs.jsonl]({SITE}/data/runs.jsonl)\n")
    write_page(os.path.join(REPORTS, f"{day}.md"), lines)
    return desc


# ---------- 주간·월간 보고 ----------
def period_key(d, kind):
    y, m, dd = map(int, d.split("-"))
    if kind == "weekly":
        iso = date(y, m, dd).isocalendar()
        return f"{iso[0]}-W{iso[1]:02d}"
    return d[:7]


def period_days(key, kind):
    if kind == "weekly":
        y, w = key.split("-W")
        start = date.fromisocalendar(int(y), int(w), 1)
        return [(start + timedelta(days=i)).isoformat() for i in range(7)]
    y, m = map(int, key.split("-"))
    nxt = date(y + (m == 12), (m % 12) + 1, 1)
    return [date(y, m, i + 1).isoformat() for i in range((nxt - date(y, m, 1)).days)]


def write_period(kind, key, by_day, by_day_usage):
    days = period_days(key, kind)
    runs_ = [r for d in days for r in by_day.get(d, [])]
    usage_ = [u for d in days for u in by_day_usage.get(d, [])]
    c = counts(runs_)
    u = usage_sum(usage_)
    label = f"{key} 주에" if kind == "weekly" else f"{key} 월에"
    title = f"자율 개선 {'주간' if kind == 'weekly' else '월간'} 보고 {key}"
    desc = summary_sentence(label, c, runs_, u)
    active_days = [d for d in days if by_day.get(d)]
    fail_rate = (c["failed"] / c["total"] * 100) if c["total"] else 0
    url = f"{SITE}/{kind}/{key}/"
    ld = {"@context": "https://schema.org", "@type": "Report", "headline": title, "name": title, "description": desc, "url": url,
          "inLanguage": "ko", "datePublished": days[0], "dateModified": NOW.isoformat(timespec="seconds"),
          "author": {"@type": "Person", "name": "hkjang", "url": GH}, "isPartOf": {"@type": "WebSite", "name": "aidev 자율 개선 대시보드", "url": SITE}}
    lines = [front_matter(title, desc, days[0], {"type": "report"}), jsonld(ld), f"# {title}\n",
             f'<p class="tldr"><strong>요약.</strong> {esc(desc)} 활동일 {len(active_days)}/{len(days)}일, 실패율 {fail_rate:.1f}%.</p>\n',
             stats_html(c, u), "## 날짜별\n", chart_svg(by_day, days, f"{key} 회차 수")]
    rows = []
    for d in days:
        if not by_day.get(d):
            continue
        cc = counts(by_day[d]); uu = usage_sum(by_day_usage.get(d, []))
        st = "failed" if cc["failed"] else ("released" if cc["released"] else ("merged" if cc["merged"] else "nochange"))
        rows.append((st, [f'<a href="{SITE}/reports/{d}/">{d}</a>', str(cc["total"]), str(cc["released"]), str(cc["merged"]), str(cc["nochange"]), str(cc["failed"]),
                          f"${uu['cost']:.2f}" if uu["sessions"] else "—", fmt_min(uu["minutes"]) if uu["sessions"] else "—"]))
    lines.append(table([("날짜", "primary"), ("회차", "num"), ("릴리즈", "num"), ("머지", "num"), ("변경 없음", "num"), ("실패", "num"), ("비용", "num"), ("시간", "num")], rows))
    lines += ["## 프로젝트별\n", project_release_table(runs_)]
    fails = [r for r in runs_ if classify(r.get("result")) == "failed" or any(k in (r.get("result") or "") for k, _ in WARN_PATTERNS)]
    if fails:
        lines += ["## 실패·경고 회차\n", runs_table(fails, with_date=True, filterable=False)]
    lines.append(f"\n[← 대시보드]({SITE}/)\n")
    write_page(os.path.join(DOCS, kind, f"{key}.md"), lines)
    return {"key": key, "url": url, "days": len(active_days), **{k: c[k] for k in ("total", "released", "merged", "nochange", "failed")},
            "cost": round(u["cost"], 2), "minutes": round(u["minutes"])}


# ---------- 프로젝트 페이지 ----------
def write_project(p, runs_p, usage_p):
    repo = repo_name(p, runs_p[-1].get("result") if runs_p else "")
    rel = release_info(p)
    c = counts(runs_p)
    u = usage_sum(usage_p)
    last = runs_p[-1] if runs_p else {}
    desc = (f"{p}: 자율 개선 회차 {c['total']}회, 릴리즈 {c['released']}건. 최근 릴리즈 {rel.get('tag') or '없음'}"
            + (f" (자산 {rel.get('assets_count')}개)" if rel.get("assets_count") is not None else "") + ".")
    ld = {"@context": "https://schema.org", "@type": "SoftwareSourceCode", "name": p, "codeRepository": f"{GH}/{repo}",
          "url": f"{SITE}/projects/{p}/", "description": desc, "inLanguage": "ko",
          "maintainer": {"@type": "Person", "name": "hkjang", "url": GH}, "dateModified": NOW.isoformat(timespec="seconds")}
    if rel.get("tag"):
        ld["version"] = rel.get("version") or rel.get("tag")
    lines = [front_matter(f"{p} — 자율 개선 이력", desc), jsonld(ld), f"# {p}\n",
             f'<p class="tldr"><strong>요약.</strong> {esc(desc)}</p>\n', stats_html(c, u), "## 현황\n", '<dl class="kv">',
             f'<dt>저장소</dt><dd><a href="{GH}/{repo}">{GH}/{repo}</a></dd>']
    if last:
        lines.append(f'<dt>마지막 회차</dt><dd>{esc((last.get("ts") or "")[:16].replace("T", " "))} KST — {pill(classify(last.get("result")))} {result_html(p, last.get("result"))}</dd>')
    if rel:
        tag = rel.get("tag") or ""
        rel_link = f'<a href="{GH}/{repo}/releases/tag/{esc(tag)}">{esc(tag)}</a>' if tag else esc(rel.get("status", ""))
        assets = rel.get("assets_count")
        a_s = "" if assets is None else (f" · 자산 {assets}개" + (f" (이전 {esc(str(rel.get('prev_tag') or ''))}: {rel.get('prev_assets_count')}개)" if rel.get("prev_tag") else ""))
        if assets == 0 and (rel.get("prev_assets_count") or 0) > 0:
            a_s += ' <span class="pill pill-failed">❌ 자산 누락</span>'
        lines.append(f"<dt>최근 릴리즈</dt><dd>{rel_link} — {esc(rel.get('status',''))}{a_s}" + (f' <a href="{GH}/{repo}/releases">전체 릴리즈 →</a>' if tag else "") + "</dd>")
        if rel.get("reason"):
            lines.append(f"<dt>사유</dt><dd>{esc(rel['reason'])}</dd>")
    for q in fix_queue():
        if q["project"] == p:
            lines.append(f"<dt>수정 과제</dt><dd>⚠️ {esc(q['note'])}</dd>")
    lines.append("</dl>\n")
    lines += ["## 회차 이력\n", runs_table(list(reversed(runs_p)), with_date=True, filterable=len(runs_p) > 8)]
    if usage_p:
        lines += ["## 비용·사용량\n", usage_table(list(reversed(usage_p))[:30], caption="최근 30세션")]
    lt = ledger_text(p)
    if lt:
        lines += ["## 원장 (에이전트가 남긴 기록)\n", re.sub(r"^# .*\n", "", lt, count=1).strip(), ""]
    lines.append(f"\n[← 대시보드]({SITE}/)\n")
    write_page(os.path.join(PROJECTS, f"{p}.md"), lines)
    return {"name": p, "repo": f"{GH}/{repo}", "page": f"{SITE}/projects/{p}/", "runs": c["total"], "released": c["released"],
            "last_ts": last.get("ts"), "last_status": classify(last.get("result")) if last else None, "last_result": last.get("result"),
            "cost": round(u["cost"], 2), "minutes": round(u["minutes"]),
            "release": {k: rel.get(k) for k in ("status", "version", "tag", "assets_count", "prev_tag", "prev_assets_count") if k in rel}}


# ---------- 피드 ----------
def write_feed(days, descs):
    entries = "\n".join(f"""  <entry>
    <title>자율 개선 일일 보고 {d}</title>
    <link href="{SITE}/reports/{d}/"/>
    <id>{SITE}/reports/{d}/</id>
    <updated>{d}T23:59:59+09:00</updated>
    <summary>{esc(descs.get(d, ''))}</summary>
  </entry>""" for d in days[:30])
    xml = f"""<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom" xml:lang="ko">
  <title>aidev 자율 개선 일일 보고</title>
  <subtitle>Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 결과</subtitle>
  <link href="{SITE}/"/>
  <link rel="self" href="{SITE}/feed.xml"/>
  <id>{SITE}/</id>
  <updated>{NOW.isoformat(timespec='seconds')}</updated>
  <author><name>hkjang</name><uri>{GH}</uri></author>
{entries}
</feed>
"""
    with open(os.path.join(DOCS, "feed.xml"), "w", encoding="utf-8") as f:
        f.write(xml)


# ---------- 대시보드 ----------
FAQ = [
    ("이 페이지는 무엇인가요?",
     "hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다."),
    ("얼마나 자주 갱신되나요?",
     "에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다. Atom 피드(/feed.xml)를 구독하면 일일 보고를 받아볼 수 있습니다."),
    ("한 회차에서 에이전트는 무엇을 하나요?",
     "저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열고 CI 통과를 확인한 뒤 머지하며, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로·첨부 자산)을 확인해 같은 방식으로 다음 버전을 냅니다."),
    ("릴리즈 워크플로가 실패하면 어떻게 되나요?",
     "러너가 한 번 자동으로 재실행합니다. 그래도 실패하면 실패 단계와 로그 요지를 수정 과제 큐에 넣고, 다음 회차에서 그 프로젝트를 우선 배정해 에이전트가 원인을 고칩니다(검증을 느슨하게 만드는 것은 금지). 큐에 있는 동안 '주의 필요'에 표시됩니다."),
    ("어떤 프로젝트가 대상인가요?",
     "최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다."),
    ("'머지(릴리즈 없음)'는 무슨 뜻인가요?",
     "개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다."),
    ("'주의 필요'에는 무엇이 뜨나요? 알림은 어디로 오나요?",
     "최근 2일 회차 중 CI 실패로 머지되지 않은 PR, 릴리즈 실패, 이전 릴리즈에는 있던 첨부 자산이 빠진 릴리즈, 수정 과제 대기처럼 사람이 확인해야 할 항목입니다. 새 경고가 생기면 GitHub Issue(라벨 alert)에 기록하고, 설정돼 있으면 Slack·이메일·Windows 알림으로도 보냅니다."),
    ("비용은 어떻게 계산되나요?",
     "각 회차의 claude -p 세션이 보고한 추정 비용(USD)과 소요 시간·턴 수·토큰을 그대로 합산합니다. 정액제 구독에서는 실제 청구가 아닌 참고값입니다."),
    ("원본 데이터는 어디서 보나요?",
     "회차 기록은 runs.jsonl, 사용량은 usage.jsonl, 요약은 data/summary.json, 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다."),
]


def write_index(by_day, days, by_day_usage, projects_info, alert_items, weeks, months):
    today = date.today().isoformat()
    tr = by_day.get(today, [])
    tu = by_day_usage.get(today, [])
    c = counts(tr)
    u = usage_sum(tu)
    total_runs = sum(len(v) for v in by_day.values())
    total_rel = sum(1 for v in by_day.values() for r in v if classify(r.get("result")) == "released")
    all_u = usage_sum([x for v in by_day_usage.values() for x in v])
    desc = (f"Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. "
            f"오늘 {c['total']}회차·릴리즈 {c['released']}건, 누적 {total_runs}회차·릴리즈 {total_rel}건, 주의 필요 {len(alert_items)}건.")
    website = {"@context": "https://schema.org", "@type": "WebSite", "name": "aidev 자율 개선 대시보드", "url": SITE + "/",
               "description": desc, "inLanguage": "ko", "author": {"@type": "Person", "name": "hkjang", "url": GH},
               "dateModified": NOW.isoformat(timespec="seconds")}
    faq = {"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": [
        {"@type": "Question", "name": q, "acceptedAnswer": {"@type": "Answer", "text": a}} for q, a in FAQ]}
    itemlist = {"@context": "https://schema.org", "@type": "ItemList", "name": "일일 보고", "itemListOrder": "Descending",
                "itemListElement": [{"@type": "ListItem", "position": i + 1, "name": f"일일 보고 {d}", "url": f"{SITE}/reports/{d}/"} for i, d in enumerate(days[:30])]}
    dataset = {"@context": "https://schema.org", "@type": "Dataset", "name": "aidev 회차 기록 (runs.jsonl)",
               "description": "자율 개선 에이전트의 회차별 기록. 한 줄에 한 회차, 필드: ts, date, project, result.",
               "url": f"{SITE}/data/runs.jsonl", "license": "https://opensource.org/license/mit", "inLanguage": "ko",
               "creator": {"@type": "Person", "name": "hkjang"}, "encodingFormat": "application/x-ndjson",
               "distribution": [{"@type": "DataDownload", "encodingFormat": "application/x-ndjson", "contentUrl": f"{SITE}/data/runs.jsonl"},
                                {"@type": "DataDownload", "encodingFormat": "application/x-ndjson", "contentUrl": f"{SITE}/data/usage.jsonl"},
                                {"@type": "DataDownload", "encodingFormat": "application/json", "contentUrl": f"{SITE}/data/summary.json"}]}
    lines = [front_matter("aidev 자율 개선 대시보드", desc), jsonld(website), jsonld(faq), jsonld(itemlist), jsonld(dataset),
             "# aidev 자율 개선 대시보드\n",
             f'<p class="tldr"><strong>한 줄 요약.</strong> {esc(desc)} 회차가 끝날 때마다 자동 갱신됩니다 '
             f'(마지막 갱신 <time datetime="{NOW.isoformat(timespec="seconds")}" data-rel>{NOW.strftime("%Y-%m-%d %H:%M")}</time> KST).</p>\n',
             alerts_html(alert_items),
             f"[운영 문서]({GH}/aidev#readme) · [원장]({GH}/aidev/tree/main/state) · [실행 이력]({GH}/aidev/commits/main) · [경고 이슈]({GH}/aidev/issues?q=label%3Aalert) · "
             f"[Atom 피드]({SITE}/feed.xml) · [summary.json]({SITE}/data/summary.json)\n",
             f"## 오늘 ({today})\n"]
    if tr:
        lines += [stats_html(c, u), f"[{today} 보고 자세히 보기 →]({SITE}/reports/{today}/)\n", runs_table(tr, caption="오늘 전체 회차 (KST)")]
    else:
        lines.append("아직 오늘 회차가 없습니다. 아래에서 최근 보고를 볼 수 있습니다.\n")
    lines += ["## 최근 14일\n", chart_svg(by_day)]
    rows = []
    for d in days[:14]:
        cc = counts(by_day[d]); uu = usage_sum(by_day_usage.get(d, []))
        st = "failed" if cc["failed"] else ("released" if cc["released"] else ("merged" if cc["merged"] else "nochange"))
        rows.append((st, [f'<a href="{SITE}/reports/{d}/">{d}</a>', str(cc["total"]), str(cc["released"]), str(cc["merged"]), str(cc["nochange"]), str(cc["failed"]),
                          f"${uu['cost']:.2f}" if uu["sessions"] else "—"]))
    lines += ["## 일일 보고\n", table([("날짜", "primary"), ("회차", "num"), ("릴리즈", "num"), ("머지", "num"), ("변경 없음", "num"), ("실패", "num"), ("비용", "num")], rows)]
    rows = [("released" if w["released"] else "nochange", [f'<a href="{w["url"]}">{w["key"]}</a>', str(w["days"]), str(w["total"]), str(w["released"]), str(w["failed"]), f"${w['cost']:.2f}", fmt_min(w["minutes"])]) for w in weeks[:8]]
    lines += ["## 주간·월간 보고\n", "### 주간\n",
              table([("주", "primary"), ("활동일", "num"), ("회차", "num"), ("릴리즈", "num"), ("실패", "num"), ("비용", "num"), ("시간", "num")], rows)]
    rows = [("released" if m["released"] else "nochange", [f'<a href="{m["url"]}">{m["key"]}</a>', str(m["days"]), str(m["total"]), str(m["released"]), str(m["failed"]), f"${m['cost']:.2f}", fmt_min(m["minutes"])]) for m in months[:6]]
    lines += ["### 월간\n", table([("월", "primary"), ("활동일", "num"), ("회차", "num"), ("릴리즈", "num"), ("실패", "num"), ("비용", "num"), ("시간", "num")], rows)]
    rows = []
    for info in sorted(projects_info, key=lambda x: x["name"].lower()):
        rel = info["release"]
        tag = rel.get("tag") or rel.get("version") or ""
        if tag and rel.get("status") == "released":
            rel_s = f'<a href="{info["repo"]}/releases/tag/{esc(tag)}">{esc(tag)}</a>'
            if rel.get("assets_count") is not None:
                rel_s += f' <span class="meta">자산 {rel["assets_count"]}</span>'
                if rel["assets_count"] == 0 and (rel.get("prev_assets_count") or 0) > 0:
                    rel_s += ' <span class="pill pill-failed">❌ 누락</span>'
        else:
            rel_s = esc(rel.get("status") or "")
        st = info["last_status"] or "other"
        rows.append((st, [f'<a href="{info["page"]}">{esc(info["name"])}</a>', esc((info["last_ts"] or "")[:16].replace("T", " ")),
                          pill(st) + " " + result_html(info["name"], info["last_result"]), rel_s, f"${info['cost']:.2f}"]))
    lines += ["## 프로젝트별 현황\n",
              table([("프로젝트", "primary"), ("마지막 회차", ""), ("결과", ""), ("최근 릴리즈", ""), ("누적 비용", "num")], rows, filterable=True,
                    caption="프로젝트 이름을 누르면 원장 전체와 회차 이력을 볼 수 있습니다.")]
    lines += ["## 비용·사용량\n",
              f'<ul class="stats"><li><b>${u["cost"]:.2f}</b><span>오늘 비용</span></li><li><b>{fmt_min(u["minutes"])}</b><span>오늘 에이전트 시간</span></li>'
              f'<li><b>{int(u["sessions"])}</b><span>오늘 세션</span></li><li><b>${all_u["cost"]:.2f}</b><span>누적 비용</span></li>'
              f'<li><b>{fmt_min(all_u["minutes"])}</b><span>누적 시간</span></li><li><b>{fmt_tok(all_u["in"])}/{fmt_tok(all_u["out"])}</b><span>누적 토큰 입력/출력</span></li></ul>\n',
              f"claude -p 가 세션마다 보고한 추정값(정액제에서는 참고값). 회차별 내역은 각 일일 보고와 프로젝트 페이지, 원본은 [usage.jsonl]({SITE}/data/usage.jsonl).\n"]
    lines.append("## FAQ\n")
    for q, a in FAQ:
        lines.append(f"<details><summary>{esc(q)}</summary><p>{esc(a)}</p></details>")
    lines.append("")
    lines.append(f"---\n러너·프롬프트·원장은 [{GH}/aidev]({GH}/aidev) 에서 관리한다. 이 페이지는 회차가 끝날 때마다 `bin/report.py` 가 다시 만든다.\n")
    write_page(os.path.join(DOCS, "index.md"), lines)
    return desc, c, u, total_runs, total_rel, all_u


def main():
    runs = load_jsonl(DATA)
    usage = load_jsonl(USAGE)
    by_day, by_project, by_day_usage, by_project_usage = defaultdict(list), defaultdict(list), defaultdict(list), defaultdict(list)
    for r in runs:
        d = r.get("date") or (r.get("ts") or "")[:10]
        if d:
            by_day[d].append(r)
        if r.get("project") and re.match(r"^[A-Za-z0-9_.-]+$", r["project"]):
            by_project[r["project"]].append(r)
    for x in usage:
        d = x.get("date") or (x.get("ts") or "")[:10]
        if d:
            by_day_usage[d].append(x)
        if x.get("project"):
            by_project_usage[x["project"]].append(x)
    days = sorted(by_day, reverse=True)
    descs = {}
    for d in by_day:
        by_day[d].sort(key=lambda r: r.get("ts") or "")
        by_day_usage[d].sort(key=lambda r: r.get("ts") or "")
        descs[d] = write_report(d, by_day[d], by_day_usage.get(d, []))
    weeks = [write_period("weekly", k, by_day, by_day_usage) for k in sorted({period_key(d, "weekly") for d in days}, reverse=True)]
    months = [write_period("monthly", k, by_day, by_day_usage) for k in sorted({period_key(d, "monthly") for d in days}, reverse=True)]
    projects_info = []
    for p in by_project:
        by_project[p].sort(key=lambda r: r.get("ts") or "")
        by_project_usage[p].sort(key=lambda r: r.get("ts") or "")
        projects_info.append(write_project(p, by_project[p], by_project_usage.get(p, [])))
    alert_items = alerts(by_day, days)
    desc, c, u, total_runs, total_rel, all_u = write_index(by_day, days, by_day_usage, projects_info, alert_items, weeks, months)
    write_feed(days, descs)
    os.makedirs(os.path.join(DOCS, "data"), exist_ok=True)
    summary = {"generated": NOW.isoformat(timespec="seconds"), "site": SITE, "description": desc,
               "today": {"date": date.today().isoformat(), **{k: c[k] for k in ("total", "projects", "released", "merged", "nochange", "failed")},
                         "cost_usd": round(u["cost"], 2), "minutes": round(u["minutes"]), "sessions": int(u["sessions"])},
               "totals": {"runs": total_runs, "released": total_rel, "days": len(days), "cost_usd": round(all_u["cost"], 2),
                          "minutes": round(all_u["minutes"]), "tokens_in": int(all_u["in"]), "tokens_out": int(all_u["out"])},
               "alerts": alert_items, "fix_queue": fix_queue(),
               "days": [{"date": d, **{k: counts(by_day[d])[k] for k in ("total", "released", "merged", "nochange", "failed")},
                         "cost_usd": round(usage_sum(by_day_usage.get(d, []))["cost"], 2)} for d in days[:30]],
               "weeks": weeks[:8], "months": months[:6],
               "projects": sorted(projects_info, key=lambda x: x["name"].lower())}
    with open(os.path.join(DOCS, "data", "summary.json"), "w", encoding="utf-8") as f:
        json.dump(summary, f, ensure_ascii=False, indent=1)
    print(f"reports: {len(by_day)} day(s), {len(weeks)} week(s), {len(months)} month(s), {len(runs)} run(s), "
          f"{len(usage)} usage row(s), {len(projects_info)} project page(s), {len(alert_items)} alert(s)")


if __name__ == "__main__":
    main()
