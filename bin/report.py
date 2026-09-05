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
    ("review rejected", "리뷰 에이전트가 머지 보류 — PR 열림, 사람 판단 필요"),
    ("guarded files", "보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요"),
    ("rollback PR", "자동 롤백 PR 생성 — 사람 머지 필요"),
    ("rollback failed", "롤백 충돌 — 사람 개입 필요"),
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
OUTCOME_LABEL = {"no-change": "변경 없음", "review-pending": "검토 대기", "verify-failed": "검증 실패", "merged": "병합 완료",
                 "releasing": "릴리즈 진행 중", "release-ready": "배포 준비 완료", "error": "실행 오류"}


def outcome_of(r):
    """구조화 outcome 이 있으면 그것을, 없으면 결과 문장으로 추정한다 (구 기록 호환)."""
    o = r.get("outcome") if isinstance(r, dict) else None
    if o in OUTCOME_LABEL:
        return o
    res = (r.get("result") or "") if isinstance(r, dict) else ""
    if "ASSETS MISSING" in res or "release failed" in res or "release missing" in res:
        return "releasing" if re.search(r"released v?\d", res) else "merged"
    if re.search(r"released v?\d", res):
        return "release-ready"
    if "PR open" in res or "review" in res or "guarded" in res:
        return "review-pending"
    if "merge failed" in res or "CI failed" in res or "verify failed" in res:
        return "verify-failed"
    if "merged" in res:
        return "merged"
    if "no change" in res:
        return "no-change"
    if "error" in res or "daily cap" in res:
        return "error"
    return "no-change"


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


OUTCOME_PILL = {"no-change": "nochange", "review-pending": "merged", "verify-failed": "failed", "merged": "merged",
                "releasing": "merged", "release-ready": "released", "error": "failed"}


def outcome_pill(r):
    o = outcome_of(r)
    return f'<span class="pill pill-{OUTCOME_PILL[o]}" title="outcome={o}">{OUTCOME_LABEL[o]}</span>'


def meta_html(r):
    """runs.jsonl 의 변경 요약(files/additions/deletions/tests/title)을 한 줄로."""
    if not r.get("files"):
        return ""
    t = f' — {esc(r["title"])}' if r.get("title") else ""
    tests = f' · 테스트 {r.get("tests", 0)}' if r.get("tests") else " · <em>테스트 없음</em>"
    return f'<div class="meta">{r["files"]}파일 <span style="color:var(--good)">+{r.get("additions",0)}</span>/<span style="color:var(--bad)">−{r.get("deletions",0)}</span>{tests}{t}</div>'


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
                parts = line.rstrip("\n").split("\t")
                out.append({"project": parts[0], "note": parts[1], "merge_sha": parts[2] if len(parts) > 2 else ""})
    return out


def ideas(project):
    try:
        return json.load(open(os.path.join(STATE, f"{project}.ideas.json"), encoding="utf-8"))
    except Exception:
        return []


def caps():
    out = {}
    path = os.path.join(STATE, "caps.env")
    if os.path.exists(path):
        for line in open(path, encoding="utf-8"):
            m = re.match(r"^\s*([A-Z_]+)=(\d+)", line)
            if m:
                out[m.group(1)] = int(m.group(2))
    return out


def lessons_all():
    return load_jsonl(os.path.join(STATE, "lessons.jsonl"))


def health():
    try:
        return json.load(open(os.path.join(DOCS, "data", "health.json"), encoding="utf-8"))
    except Exception:
        return {}


def health_grade(runs_p, lessons_p):
    """최근 14일: 실패·경고·회귀로 A~D. (등급, 설명)"""
    cutoff = (date.today() - timedelta(days=14)).isoformat()
    recent = [r for r in runs_p if (r.get("date") or "") >= cutoff]
    rel = sum(1 for r in recent if classify(r.get("result")) == "released")
    fails = sum(1 for r in recent if classify(r.get("result")) == "failed")
    warns = sum(1 for r in recent if any(k in (r.get("result") or "") for k, _ in WARN_PATTERNS))
    regress = sum(1 for l in lessons_p if (l.get("date") or "") >= cutoff)
    if regress >= 2 or warns >= 4 or fails >= 3:
        g = "D"
    elif regress == 1 or warns >= 2 or fails >= 1:
        g = "C"
    elif warns == 1 or rel == 0:
        g = "B"
    else:
        g = "A"
    return g, f"14일: 릴리즈 {rel}, 실패 {fails}, 경고 {warns}, 회귀 {regress}"


GRADE_PILL = {"A": "pill-released", "B": "pill-merged", "C": "pill-merged", "D": "pill-failed"}


def grade_html(g, why):
    return f'<span class="pill {GRADE_PILL[g]}" title="{esc(why)}">건강 {g}</span>'


def health_html(h, lessons):
    if not h:
        return ""
    since = int(h.get("seconds_since_last") or 0)
    ago = f"{since//3600}시간 {since%3600//60}분 전" if since >= 3600 else f"{since//60}분 전"
    ok = h.get("ok", True)
    cls = "alerts ok" if ok else "alerts"
    items = "".join(f"<li>{esc(x)}</li>" for x in (h.get("problems") or []) + [f"조치: {a}" for a in (h.get("actions") or [])])
    cutoff = (date.today() - timedelta(days=7)).isoformat()
    reg7 = sum(1 for l in lessons if (l.get("date") or "") >= cutoff)
    nxt = f' · 다음 실행 {esc(h["scheduler_next_run"])}' if h.get("scheduler_next_run") else ""
    return (f'<div class="{cls}" role="status"><strong>{"🩺 러너 정상" if ok else "🩺 러너 점검 필요"}</strong> '
            f'<span class="meta">— 마지막 회차 {ago} · 스케줄러 {esc(h.get("scheduler_status") or "?")}{nxt} · 디스크 {h.get("disk_percent", "?")}% · '
            f'최근 7일 회귀 {reg7}건 · 점검 {esc((h.get("checked") or "")[11:16])}</span>{("<ul>" + items + "</ul>") if items else ""}</div>\n')


def counts(day_runs):
    c = defaultdict(int)
    for r in day_runs:
        c[classify(r.get("result"))] += 1
        c["o:" + outcome_of(r)] += 1
    c["total"] = len(day_runs)
    c["projects"] = len({r.get("project") for r in day_runs if r.get("project") != "(runner)"})
    return c


def quality_metrics(runs, usage, lessons, alert_hist, days_window=14):
    """대시보드 품질 지표. 관찰 기간(24h)이 지나지 않은 머지는 '검증된 개선' 분모에서 뺀다."""
    cutoff = (date.today() - timedelta(days=days_window)).isoformat()
    recent = [r for r in runs if (r.get("date") or "") >= cutoff and r.get("project") != "(runner)"]
    merged = [r for r in recent if outcome_of(r) in ("merged", "releasing", "release-ready")]
    observed = [r for r in merged if (NOW - datetime.fromisoformat(r["ts"])).total_seconds() >= 86400] if merged else []
    prs = {r.get("pr") for r in observed if r.get("pr")}
    regressed_prs = {l.get("pr") for l in lessons if l.get("kind") in ("ci-broken-after-merge", "reverted", "rolled-back") and (l.get("date") or "") >= cutoff}
    reworked_prs = {l.get("pr") for l in lessons if l.get("kind") in ("reverted", "rolled-back") and (l.get("date") or "") >= cutoff}
    verified = [r for r in observed if r.get("pr") not in regressed_prs]
    rel_attempts = [r for r in recent if outcome_of(r) in ("releasing", "release-ready")]
    rel_ready = [r for r in rel_attempts if outcome_of(r) == "release-ready"]
    costed_ids = {u.get("run_id") for u in usage if u.get("cost_usd") is not None and u.get("run_id")}
    cost_total = sum(float(u.get("cost_usd") or 0) for u in usage if (u.get("date") or "") >= cutoff and u.get("cost_usd") is not None)
    verified_costed = [r for r in verified if r.get("run_id") in costed_ids]
    unknown_cost = sum(1 for u in usage if (u.get("date") or "") >= cutoff and u.get("cost_usd") is None)
    # 예외 처리 소요 시간: alerts-history 의 open→close 짝
    durations = []
    opened = {}
    for e in alert_hist:
        k = e.get("key")
        if e.get("event") == "open":
            opened[k] = e.get("ts")
        elif e.get("event") == "close" and k in opened:
            try:
                durations.append((datetime.fromisoformat(e["ts"]) - datetime.fromisoformat(opened.pop(k))).total_seconds() / 3600)
            except Exception:
                pass
    durations.sort()
    med = durations[len(durations) // 2] if durations else None

    def pct(a, b):
        return f"{a / b * 100:.0f}%" if b else "—"
    return [
        ("검증된 개선 완료율", pct(len(verified), len(observed)), f"관찰 24h 지난 머지 {len(observed)}건 중 회귀 없음 {len(verified)}건"),
        ("완전한 릴리즈 비율", pct(len(rel_ready), len(rel_attempts)), f"릴리즈 시도 {len(rel_attempts)}건 중 자산 검증까지 {len(rel_ready)}건"),
        ("사람의 재작업률", pct(len(reworked_prs & prs), len(observed)), f"되돌림·롤백 {len(reworked_prs & prs)}건 / 관찰 머지 {len(observed)}건"),
        ("변경 후 회귀율", pct(len(regressed_prs & prs), len(observed)), f"회귀 {len(regressed_prs & prs)}건 / 관찰 머지 {len(observed)}건"),
        ("유효 개선당 비용", (f"${cost_total / len(verified_costed):.2f}" if verified_costed else "—"), f"비용 확인된 유효 개선 {len(verified_costed)}건, 미확인 세션 {unknown_cost}"),
        ("예외 처리 소요 시간(중앙값)", (f"{med:.1f}시간" if med is not None else "—"), f"해결된 경고 {len(durations)}건"),
        ("실행 오류", str(sum(1 for r in recent if outcome_of(r) == "error")), f"최근 {days_window}일"),
    ]


def alert_history():
    return load_jsonl(os.path.join(STATE, "alerts-history.jsonl"))


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
    items = [("회차", c["total"]), ("프로젝트", c["projects"]), ("배포 준비 완료", c["o:release-ready"]), ("릴리즈 진행 중", c["o:releasing"]),
             ("병합 완료", c["o:merged"]), ("검토 대기", c["o:review-pending"]), ("검증 실패", c["o:verify-failed"]), ("변경 없음", c["o:no-change"]), ("실행 오류", c["o:error"])]
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
                      outcome_pill(r) + " " + result_html(p, r.get("result")) + meta_html(r)]))
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
    lessons_p = [l for l in lessons_all() if l.get("project") == p]
    grade, gwhy = health_grade(runs_p, lessons_p)
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
             f'<p class="tldr"><strong>요약.</strong> {esc(desc)} {grade_html(grade, gwhy)} <span class="meta">{esc(gwhy)}</span></p>\n', stats_html(c, u), "## 현황\n", '<dl class="kv">',
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
    ideas_p = ideas(p)
    pend = [i for i in ideas_p if i.get("status") == "pending"]
    if ideas_p:
        lines += [f"## 아이디어 백로그 — 대기 {len(pend)} / 전체 {len(ideas_p)}\n"]
        rows = [("nochange" if i.get("status") == "pending" else ("released" if i.get("status") == "done" else "failed"),
                 [esc(i.get("title", "")), esc(f"{i.get('value','?')}/{i.get('risk','?')}/{i.get('size','?')}"),
                  esc({"pending": "대기", "done": "완료", "rejected": "기각"}.get(i.get("status"), i.get("status", ""))), esc(i.get("note", "")), esc(i.get("updated", ""))])
                for i in sorted(ideas_p, key=lambda x: (x.get("status") != "pending", -(x.get("value") or 0), x.get("risk") or 0))]
        lines.append(table([("아이디어", "primary"), ("가치/위험/크기", ""), ("상태", ""), ("메모", ""), ("갱신", "")], rows, filterable=len(rows) > 8,
                           caption="에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다."))
    if lessons_p:
        lines += ["## 교훈 (깨졌던 변경)\n"] + [f"- {esc(l.get('date',''))} **{esc(l.get('kind',''))}** — {esc(l.get('detail',''))}" + (f" ([링크]({l['pr']}))" if (l.get("pr") or "").startswith("http") else "") for l in reversed(lessons_p)] + [""]
    lt = ledger_text(p)
    if lt:
        lines += ["## 원장 (에이전트가 남긴 기록)\n", re.sub(r"^# .*\n", "", lt, count=1).strip(), ""]
    lines.append(f"\n[← 대시보드]({SITE}/) · [교훈 모음]({SITE}/lessons/)\n")
    write_page(os.path.join(PROJECTS, f"{p}.md"), lines)
    return {"name": p, "repo": f"{GH}/{repo}", "page": f"{SITE}/projects/{p}/", "runs": c["total"], "released": c["released"],
            "last_ts": last.get("ts"), "last_status": classify(last.get("result")) if last else None, "last_result": last.get("result"),
            "cost": round(u["cost"], 2), "minutes": round(u["minutes"]), "grade": grade, "grade_why": gwhy, "lessons": len(lessons_p),
            "ideas_pending": len(pend), "ideas_total": len(ideas_p),
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


# ---------- 교훈 페이지 ----------
def write_lessons(lessons):
    desc = f"자율 개선 에이전트의 변경이 깨뜨린 것들 {len(lessons)}건 — 머지 뒤 CI 실패, 되돌림, 릴리즈 워크플로 반복 실패, 롤백. 다음 회차 프롬프트에 프로젝트별로 주입된다."
    lines = [front_matter("교훈 — 깨졌던 변경 모음", desc, None, {"type": "report"}), "# 교훈 — 깨졌던 변경 모음\n",
             f'<p class="tldr"><strong>요약.</strong> {esc(desc)}</p>\n']
    byk = defaultdict(int)
    for l in lessons:
        byk[l.get("kind", "")] += 1
    if byk:
        lines.append('<ul class="stats">' + "".join(f"<li><b>{v}</b><span>{esc(k)}</span></li>" for k, v in sorted(byk.items(), key=lambda x: -x[1])) + "</ul>\n")
    rows = [("failed", [esc(l.get("date", "")), f'<a href="{SITE}/projects/{esc(l.get("project",""))}/">{esc(l.get("project",""))}</a>', esc(l.get("kind", "")),
                        esc(l.get("detail", "")) + (f' <a href="{l["pr"]}">링크</a>' if (l.get("pr") or "").startswith("http") else "")]) for l in reversed(lessons)]
    lines.append(table([("날짜", ""), ("프로젝트", "primary"), ("종류", ""), ("내용", "")], rows, filterable=True) if rows else "아직 기록된 교훈이 없습니다.\n")
    lines.append(f"\n[← 대시보드]({SITE}/)\n")
    write_page(os.path.join(DOCS, "lessons.md"), lines)


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
    ("머지 전에 검토는 없나요?",
     "있습니다. 구현과 다른 세션의 리뷰 에이전트가 diff 만 읽고 '머지하면 안 되는 이유'(검증하지 않는 테스트, 논리 오류, 설명과 다른 동작, 위험한 변경)를 찾습니다. 거절하면 PR 에 사유를 달고 열어 둡니다. 또 워크플로·마이그레이션·인증·결제·배포 파일(보호 파일)을 건드린 PR 은 자동 머지하지 않습니다."),
    ("깨진 변경은 어떻게 되나요?",
     "머지 2시간 뒤부터 main CI 실패와 되돌림 커밋을 확인해 '교훈'으로 기록하고, 그 프로젝트의 다음 회차 프롬프트에 주입해 같은 실수를 피하게 합니다. 릴리즈 워크플로가 반복 실패하고 수정 회차도 실패하면 원래 머지를 되돌리는 롤백 PR 을 자동으로 엽니다(머지는 사람). 프로젝트마다 최근 14일의 실패·경고·회귀로 건강 등급 A~D 를 매깁니다."),
    ("비용은 어떻게 계산되나요?",
     "각 회차의 claude -p 세션이 보고한 추정 비용(USD)과 소요 시간·턴 수·토큰을 그대로 합산합니다. 정액제 구독에서는 실제 청구가 아닌 참고값입니다."),
    ("원본 데이터는 어디서 보나요?",
     "회차 기록은 runs.jsonl, 사용량은 usage.jsonl, 요약은 data/summary.json, 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다."),
]


def write_index(by_day, days, by_day_usage, projects_info, alert_items, weeks, months, lessons):
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
             alerts_html(alert_items), health_html(health(), lessons),
             f"[운영 문서]({GH}/aidev#readme) · [원장]({GH}/aidev/tree/main/state) · [실행 이력]({GH}/aidev/commits/main) · [경고 이슈]({GH}/aidev/issues?q=label%3Aalert) · "
             f"[교훈 {len(lessons)}건]({SITE}/lessons/) · [Atom 피드]({SITE}/feed.xml) · [summary.json]({SITE}/data/summary.json)\n",
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
        rows.append((st, [f'<a href="{info["page"]}">{esc(info["name"])}</a> ' + grade_html(info["grade"], info["grade_why"]), esc((info["last_ts"] or "")[:16].replace("T", " ")),
                          pill(st) + " " + result_html(info["name"], info["last_result"]), rel_s, f"${info['cost']:.2f}"]))
    lines += ["## 프로젝트별 현황\n",
              table([("프로젝트 · 건강", "primary"), ("마지막 회차", ""), ("결과", ""), ("최근 릴리즈", ""), ("누적 비용", "num")], rows, filterable=True,
                    caption="프로젝트 이름을 누르면 원장 전체와 회차 이력을 볼 수 있습니다.")]
    cp = caps()
    if cp:
        def bar(v, m):
            pct = min(100, int(v / m * 100)) if m else 0
            col = "var(--bad)" if pct >= 100 else ("var(--warn)" if pct >= 80 else "var(--good)")
            return f'<span class="meta">{v} / {m}</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:{pct}%;height:100%;background:{col};border-radius:4px"></span></span>'
        lines += ["## 오늘 상한\n", '<dl class="kv">',
                  f'<dt>비용</dt><dd>{bar(round(u["cost"], 2), cp.get("MAX_DAILY_COST", 0))}</dd>',
                  f'<dt>회차</dt><dd>{bar(c["total"], cp.get("MAX_DAILY_ROUNDS", 0))}</dd>',
                  f'<dt>릴리즈</dt><dd>{bar(c["released"], cp.get("MAX_DAILY_RELEASES", 0))}</dd>',
                  f'<dt>휴면 규칙</dt><dd>변경 없음 {cp.get("DORMANT_AFTER", "?")}회 연속이면 {cp.get("DORMANT_DAYS", "?")}일 제외</dd>',
                  f'<dt>수동 실행</dt><dd><a href="{GH}/aidev/issues/new?labels=run&title=run%3A+%3C%ED%94%84%EB%A1%9C%EC%A0%9D%ED%8A%B8%3E">이슈 만들기</a> — 제목 <code>run: &lt;프로젝트&gt;</code>, 라벨 <code>run</code> → 다음 회차 우선 실행</dd>',
                  "</dl>\n", "설정: `state/caps.env`. 상한에 닿으면 그날은 새 회차를 시작하지 않고 알린다.\n"]
    qm = quality_metrics([r for v in by_day.values() for r in v], [x for v in by_day_usage.values() for x in v], lessons, alert_history())
    lines += ["## 품질 지표 (최근 14일)\n",
              '<ul class="stats">' + "".join(f'<li title="{esc(w)}"><b>{esc(v)}</b><span>{esc(k)}</span></li>' for k, v, w in qm) + "</ul>\n",
              "'검증된 개선'은 머지 후 24시간 관찰에서 회귀(main CI 실패·되돌림·롤백)가 없는 변경. '완전한 릴리즈'는 태그·Release·필수 자산 검증까지 끝난 것. 비용이 확인되지 않은 세션은 0이 아니라 '미확인'으로 뺀다.\n"]
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
    lessons = lessons_all()
    write_lessons(lessons)
    desc, c, u, total_runs, total_rel, all_u = write_index(by_day, days, by_day_usage, projects_info, alert_items, weeks, months, lessons)
    write_feed(days, descs)
    os.makedirs(os.path.join(DOCS, "data"), exist_ok=True)
    summary = {"generated": NOW.isoformat(timespec="seconds"), "site": SITE, "description": desc,
               "today": {"date": date.today().isoformat(), **{k: c[k] for k in ("total", "projects", "released", "merged", "nochange", "failed")},
                         "cost_usd": round(u["cost"], 2), "minutes": round(u["minutes"]), "sessions": int(u["sessions"])},
               "totals": {"runs": total_runs, "released": total_rel, "days": len(days), "cost_usd": round(all_u["cost"], 2),
                          "minutes": round(all_u["minutes"]), "tokens_in": int(all_u["in"]), "tokens_out": int(all_u["out"])},
               "alerts": alert_items, "fix_queue": fix_queue(), "health": health(), "lessons": lessons[-50:], "caps": caps(),
               "quality": [{"metric": k, "value": v, "detail": w} for k, v, w in quality_metrics([r for v in by_day.values() for r in v], [x for v in by_day_usage.values() for x in v], lessons, alert_history())],
               "outcomes_today": {k[2:]: v for k, v in c.items() if k.startswith("o:")},
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
