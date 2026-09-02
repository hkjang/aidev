#!/usr/bin/env python3
"""docs/ 아래 GitHub Pages 용 일일 보고를 만든다.

입력  docs/data/runs.jsonl  — 러너가 회차마다 한 줄씩 남기는 기록
      state/<프로젝트>.md   — 에이전트 원장 (그날 항목을 보고에 인용)
      state/<프로젝트>.release.json
출력  docs/index.md          — 대시보드 (오늘 요약, 최근 14일, 프로젝트별 현황)
      docs/reports/<날짜>.md — 일일 보고
러너가 회차 끝마다 호출하므로 항상 그날 보고가 최신이다. 인자 없이 실행하면 전체를 다시 만든다.
"""
import json, os, re, subprocess, sys
from collections import defaultdict
from datetime import date, datetime, timedelta

REPO_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
ROOT = os.environ.get("ROOT", "/mnt/c/Users/USER/projects")
DOCS = os.path.join(REPO_DIR, "docs")
DATA = os.path.join(DOCS, "data", "runs.jsonl")
STATE = os.path.join(REPO_DIR, "state")
REPORTS = os.path.join(DOCS, "reports")
PAGES = "https://hkjang.github.io/aidev"
GH = "https://github.com/hkjang"

STATUS = {  # 결과 문자열 → (상태 키, 표시)
    "released": "🚀 릴리즈",
    "merged": "✅ 머지",
    "nochange": "➖ 변경 없음",
    "failed": "❌ 실패",
    "other": "•",
}


def classify(result):
    r = result or ""
    if re.search(r"released v?\d", r) or "released" in r and "release released" not in r:
        return "released"
    if "merge failed" in r or "release push failed" in r or "failed" in r and "merged" not in r:
        return "failed"
    if "merged" in r:
        return "merged"
    if "no change" in r or "변경없음" in r:
        return "nochange"
    return "other"


_repo_cache = {}


def repo_name(project, result):
    m = re.search(r"github\.com/hkjang/([^/]+)/pull", result or "")
    if m:
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


def linkify(project, result):
    """결과 문자열의 PR URL 과 태그를 링크로 바꾼다."""
    out = result or ""
    out = re.sub(r"(https://github\.com/\S+/pull/(\d+))", r"[PR #\2](\1)", out)
    m = re.search(r"released (v?[0-9][\w.\-]*)", out)
    if m:
        tag = m.group(1)
        out = out.replace(f"released {tag}", f"released [{tag}]({GH}/{repo_name(project, result)}/releases/tag/{tag})")
    return out


def load_runs():
    runs = []
    if not os.path.exists(DATA):
        return runs
    with open(DATA, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            try:
                runs.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    return runs


def ledger_entry(project, day):
    """원장에서 그날 항목만 뽑는다 ('## 날짜' 부터 다음 '## ' 전까지)."""
    path = os.path.join(STATE, f"{project}.md")
    if not os.path.exists(path):
        return ""
    text = open(path, encoding="utf-8").read()
    m = re.search(rf"^## {re.escape(day)}\s*$(.*?)(?=^## |\Z)", text, re.M | re.S)
    return m.group(1).strip() if m else ""


def release_info(project):
    path = os.path.join(STATE, f"{project}.release.json")
    if not os.path.exists(path):
        return {}
    try:
        return json.load(open(path, encoding="utf-8"))
    except Exception:
        return {}


def counts(day_runs):
    c = defaultdict(int)
    for r in day_runs:
        c[classify(r.get("result"))] += 1
    c["total"] = len(day_runs)
    return c


def fm(title):
    return f"---\nlayout: default\ntitle: {title}\n---\n\n"


def write_report(day, day_runs):
    os.makedirs(REPORTS, exist_ok=True)
    c = counts(day_runs)
    lines = [fm(f"일일 보고 {day}"), f"# 일일 보고 — {day}\n",
             f"[← 대시보드](../) · 회차 **{c['total']}** · {STATUS['released']} **{c['released']}** · "
             f"{STATUS['merged']}(릴리즈 없음) **{c['merged']}** · {STATUS['nochange']} **{c['nochange']}** · "
             f"{STATUS['failed']} **{c['failed']}**\n",
             "## 회차\n", "| 시각 | 프로젝트 | 결과 |", "|---|---|---|"]
    for r in day_runs:
        t = (r.get("ts") or "")[11:16]
        p = r.get("project", "")
        lines.append(f"| {t} | [{p}]({GH}/{repo_name(p, r.get('result'))}) | {STATUS[classify(r.get('result'))]} {linkify(p, r.get('result'))} |")
    lines.append("")
    seen = set()
    detail = []
    for r in day_runs:
        p = r.get("project", "")
        if p in seen:
            continue
        seen.add(p)
        entry = ledger_entry(p, day)
        if entry:
            detail.append(f"### {p}\n\n{entry}\n")
    if detail:
        lines.append("## 무엇을 왜 바꿨나 (원장 발췌)\n")
        lines.extend(detail)
    with open(os.path.join(REPORTS, f"{day}.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(lines))


def write_index(by_day):
    days = sorted(by_day, reverse=True)
    today = date.today().isoformat()
    lines = [fm("aidev 자율 개선 대시보드"), "# aidev 자율 개선 대시보드\n",
             f"갱신 {datetime.now().strftime('%Y-%m-%d %H:%M')} · "
             f"[운영 문서]({GH}/aidev#readme) · [원장]({GH}/aidev/tree/main/state) · [실행 이력]({GH}/aidev/commits/main)\n"]
    # 오늘
    tr = by_day.get(today, [])
    c = counts(tr)
    lines.append(f"## 오늘 ({today})\n")
    if tr:
        lines.append(f"회차 **{c['total']}** · {STATUS['released']} **{c['released']}** · {STATUS['merged']} **{c['merged']}** · "
                     f"{STATUS['nochange']} **{c['nochange']}** · {STATUS['failed']} **{c['failed']}** — [자세히](reports/{today})\n")
        lines += ["| 시각 | 프로젝트 | 결과 |", "|---|---|---|"]
        for r in tr[-12:]:
            p = r.get("project", "")
            lines.append(f"| {(r.get('ts') or '')[11:16]} | {p} | {STATUS[classify(r.get('result'))]} {linkify(p, r.get('result'))} |")
        lines.append("")
    else:
        lines.append("아직 회차가 없습니다.\n")
    # 최근 14일
    lines += ["## 일일 보고\n", "| 날짜 | 회차 | 릴리즈 | 머지 | 변경 없음 | 실패 |", "|---|---|---|---|---|---|"]
    for d in days[:14]:
        c = counts(by_day[d])
        lines.append(f"| [{d}](reports/{d}) | {c['total']} | {c['released']} | {c['merged']} | {c['nochange']} | {c['failed']} |")
    lines.append("")
    # 프로젝트별 현황
    latest = {}
    for d in days:
        for r in by_day[d]:
            p = r.get("project")
            if p and p not in latest:
                latest[p] = r
    lines += ["## 프로젝트별 현황\n", "| 프로젝트 | 마지막 회차 | 결과 | 최근 릴리즈 |", "|---|---|---|---|"]
    for p in sorted(latest, key=str.lower):
        r = latest[p]
        rel = release_info(p)
        tag = rel.get("tag") or rel.get("version") or ""
        rel_s = f"[{tag}]({GH}/{repo_name(p, r.get('result'))}/releases/tag/{tag})" if tag and rel.get("status") == "released" else (rel.get("status") or "")
        lines.append(f"| [{p}]({GH}/{repo_name(p, r.get('result'))}) | {(r.get('ts') or '')[:16].replace('T', ' ')} | "
                     f"{STATUS[classify(r.get('result'))]} {linkify(p, r.get('result'))} | {rel_s} |")
    lines.append("")
    lines.append(f"---\n러너·프롬프트·원장은 [{GH}/aidev]({GH}/aidev) 에서 관리한다. 이 페이지는 회차가 끝날 때마다 `bin/report.py` 가 다시 만든다.\n")
    with open(os.path.join(DOCS, "index.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(lines))


def main():
    runs = load_runs()
    by_day = defaultdict(list)
    for r in runs:
        d = r.get("date") or (r.get("ts") or "")[:10]
        if d:
            by_day[d].append(r)
    for d in by_day:
        by_day[d].sort(key=lambda r: r.get("ts") or "")
        write_report(d, by_day[d])
    write_index(by_day)
    print(f"reports: {len(by_day)} day(s), {len(runs)} run(s)")


if __name__ == "__main__":
    main()
