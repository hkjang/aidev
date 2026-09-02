#!/usr/bin/env python3
"""docs/ 아래 GitHub Pages 용 일일 보고를 만든다.

입력  docs/data/runs.jsonl  — 러너가 회차마다 한 줄씩 남기는 기록
      state/<프로젝트>.md   — 에이전트 원장 (그날 항목을 보고에 인용)
      state/<프로젝트>.release.json
출력  docs/index.md          — 대시보드 (오늘 요약, 최근 14일, 프로젝트별 현황, FAQ)
      docs/reports/<날짜>.md — 일일 보고
러너가 회차 끝마다 호출하므로 항상 그날 보고가 최신이다. 인자 없이 실행하면 전체를 다시 만든다.

SEO/AEO: 페이지마다 front matter(title/description/date/last_modified_at) 를 넣어 jekyll-seo-tag 가
meta·OG·canonical·JSON-LD(WebPage) 를 만들게 하고, 본문 첫머리에 한 문장 요약(TL;DR) 과
구조화 데이터(Report / FAQPage / ItemList / Dataset) 를 직접 넣는다. 표는 모바일에서 가로 스크롤한다.
"""
import json, os, re, subprocess
from collections import defaultdict
from datetime import date, datetime

REPO_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
ROOT = os.environ.get("ROOT", "/mnt/c/Users/USER/projects")
DOCS = os.path.join(REPO_DIR, "docs")
DATA = os.path.join(DOCS, "data", "runs.jsonl")
STATE = os.path.join(REPO_DIR, "state")
REPORTS = os.path.join(DOCS, "reports")
SITE = "https://hkjang.github.io/aidev"
GH = "https://github.com/hkjang"

STATUS = {"released": "🚀 릴리즈", "merged": "✅ 머지", "nochange": "➖ 변경 없음", "failed": "❌ 실패", "other": "•"}


def classify(result):
    r = result or ""
    if re.search(r"released v?\d", r):
        return "released"
    if "merge failed" in r or "release push failed" in r:
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


def released_tag(result):
    m = re.search(r"released (v?[0-9][\w.\-]*)", result or "")
    return m.group(1) if m else ""


def linkify(project, result):
    out = result or ""
    out = re.sub(r"(https://github\.com/\S+/pull/(\d+))", r"[PR #\2](\1)", out)
    tag = released_tag(out)
    if tag:
        out = out.replace(f"released {tag}", f"released [{tag}]({GH}/{repo_name(project, result)}/releases/tag/{tag})")
    return out


def load_runs():
    runs = []
    if not os.path.exists(DATA):
        return runs
    with open(DATA, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                try:
                    runs.append(json.loads(line))
                except json.JSONDecodeError:
                    pass
    return runs


def ledger_entry(project, day):
    path = os.path.join(STATE, f"{project}.md")
    if not os.path.exists(path):
        return ""
    text = open(path, encoding="utf-8").read()
    m = re.search(rf"^## {re.escape(day)}\s*$(.*?)(?=^## |\Z)", text, re.M | re.S)
    return m.group(1).strip() if m else ""


def release_info(project):
    path = os.path.join(STATE, f"{project}.release.json")
    try:
        return json.load(open(path, encoding="utf-8"))
    except Exception:
        return {}


def counts(day_runs):
    c = defaultdict(int)
    for r in day_runs:
        c[classify(r.get("result"))] += 1
    c["total"] = len(day_runs)
    c["projects"] = len({r.get("project") for r in day_runs})
    return c


def yaml_str(s):
    return json.dumps(s, ensure_ascii=False)  # YAML 은 JSON 문자열을 그대로 받는다


def front_matter(title, description, day=None, modified=None, extra=None):
    fm = ["---", f"title: {yaml_str(title)}", f"description: {yaml_str(description)}"]
    if day:
        fm.append(f"date: {day}")
    fm.append(f"last_modified_at: {(modified or datetime.now().astimezone()).strftime('%Y-%m-%d %H:%M:%S %z')}")
    for k, v in (extra or {}).items():
        fm.append(f"{k}: {v}")
    fm.append("---\n")
    return "\n".join(fm)


def jsonld(obj):
    return '<script type="application/ld+json">\n' + json.dumps(obj, ensure_ascii=False, indent=1) + "\n</script>\n"


def stats_html(c):
    items = [("회차", c["total"]), ("프로젝트", c["projects"]), ("릴리즈", c["released"]),
             ("머지(릴리즈 없음)", c["merged"]), ("변경 없음", c["nochange"]), ("실패", c["failed"])]
    return '<ul class="stats">' + "".join(f"<li><b>{v}</b><span>{k}</span></li>" for k, v in items) + "</ul>\n"


def summary_sentence(day, c, day_runs):
    rel = [f"{r['project']} {released_tag(r.get('result'))}" for r in day_runs if classify(r.get("result")) == "released"]
    s = (f"{day}에 자율 개선 에이전트가 {c['projects']}개 프로젝트에서 {c['total']}회차를 돌려 "
         f"{c['released']}건을 릴리즈하고 {c['merged']}건은 머지만 했으며 {c['nochange']}건은 변경이 없었고 실패는 {c['failed']}건이다.")
    if rel:
        s += " 릴리즈: " + ", ".join(rel[:8]) + ("…" if len(rel) > 8 else "") + "."
    return s


def write_report(day, day_runs):
    os.makedirs(REPORTS, exist_ok=True)
    c = counts(day_runs)
    desc = summary_sentence(day, c, day_runs)
    url = f"{SITE}/reports/{day}/"
    ld = {"@context": "https://schema.org", "@type": "Report", "headline": f"자율 개선 일일 보고 {day}",
          "name": f"자율 개선 일일 보고 {day}", "description": desc, "url": url, "inLanguage": "ko",
          "datePublished": day, "dateModified": datetime.now().isoformat(timespec="seconds"),
          "author": {"@type": "Person", "name": "hkjang", "url": f"{GH}"},
          "publisher": {"@type": "Organization", "name": "aidev", "url": SITE},
          "isPartOf": {"@type": "WebSite", "name": "aidev 자율 개선 대시보드", "url": SITE},
          "about": [{"@type": "SoftwareSourceCode", "name": p, "codeRepository": f"{GH}/{repo_name(p, r.get('result'))}"}
                    for p, r in {r.get("project"): r for r in day_runs}.items() if p]}
    crumbs = {"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": [
        {"@type": "ListItem", "position": 1, "name": "대시보드", "item": SITE + "/"},
        {"@type": "ListItem", "position": 2, "name": f"일일 보고 {day}", "item": url}]}
    lines = [front_matter(f"자율 개선 일일 보고 {day}", desc, day), jsonld(ld), jsonld(crumbs),
             f"# 자율 개선 일일 보고 — {day}\n",
             f'<p class="tldr"><strong>요약.</strong> {desc}</p>\n', stats_html(c),
             "## 회차\n", "| 시각 | 프로젝트 | 결과 |", "|---|---|---|"]
    for r in day_runs:
        p = r.get("project", "")
        lines.append(f"| {(r.get('ts') or '')[11:16]} | [{p}]({GH}/{repo_name(p, r.get('result'))}) | "
                     f"{STATUS[classify(r.get('result'))]} {linkify(p, r.get('result'))} |")
    lines.append("")
    seen, detail = set(), []
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
    lines.append(f"\n[← 대시보드]({SITE}/) · 원본 데이터 [runs.jsonl]({SITE}/data/runs.jsonl)\n")
    with open(os.path.join(REPORTS, f"{day}.md"), "w", encoding="utf-8") as f:
        f.write("\n".join(lines))


FAQ = [
    ("이 페이지는 무엇인가요?",
     "hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다."),
    ("얼마나 자주 갱신되나요?",
     "에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다."),
    ("한 회차에서 에이전트는 무엇을 하나요?",
     "저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열어 머지하고, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로)을 확인해 같은 방식으로 다음 버전을 냅니다."),
    ("어떤 프로젝트가 대상인가요?",
     "최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다."),
    ("'머지(릴리즈 없음)'는 무슨 뜻인가요?",
     "개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다."),
    ("원본 데이터는 어디서 보나요?",
     "회차 기록은 runs.jsonl(한 줄에 한 회차, JSON), 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다."),
]


def write_index(by_day):
    days = sorted(by_day, reverse=True)
    today = date.today().isoformat()
    tr = by_day.get(today, [])
    c = counts(tr)
    total_runs = sum(len(v) for v in by_day.values())
    total_rel = sum(1 for v in by_day.values() for r in v if classify(r.get("result")) == "released")
    desc = (f"Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. "
            f"오늘 {c['total']}회차·릴리즈 {c['released']}건, 누적 {total_runs}회차·릴리즈 {total_rel}건.")
    website = {"@context": "https://schema.org", "@type": "WebSite", "name": "aidev 자율 개선 대시보드", "url": SITE + "/",
               "description": desc, "inLanguage": "ko", "author": {"@type": "Person", "name": "hkjang", "url": GH},
               "dateModified": datetime.now().isoformat(timespec="seconds")}
    faq = {"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": [
        {"@type": "Question", "name": q, "acceptedAnswer": {"@type": "Answer", "text": a}} for q, a in FAQ]}
    itemlist = {"@context": "https://schema.org", "@type": "ItemList", "name": "일일 보고", "itemListOrder": "Descending",
                "itemListElement": [{"@type": "ListItem", "position": i + 1, "name": f"일일 보고 {d}", "url": f"{SITE}/reports/{d}/"}
                                    for i, d in enumerate(days[:30])]}
    dataset = {"@context": "https://schema.org", "@type": "Dataset", "name": "aidev 회차 기록 (runs.jsonl)",
               "description": "자율 개선 에이전트의 회차별 기록. 한 줄에 한 회차, 필드: ts, date, project, result.",
               "url": f"{SITE}/data/runs.jsonl", "license": "https://opensource.org/license/mit", "inLanguage": "ko",
               "creator": {"@type": "Person", "name": "hkjang"}, "encodingFormat": "application/x-ndjson",
               "distribution": [{"@type": "DataDownload", "encodingFormat": "application/x-ndjson", "contentUrl": f"{SITE}/data/runs.jsonl"}]}
    lines = [front_matter("aidev 자율 개선 대시보드", desc), jsonld(website), jsonld(faq), jsonld(itemlist), jsonld(dataset),
             "# aidev 자율 개선 대시보드\n",
             f'<p class="tldr"><strong>한 줄 요약.</strong> {desc} 회차가 끝날 때마다 자동 갱신됩니다 (마지막 갱신 {datetime.now().strftime("%Y-%m-%d %H:%M")}).</p>\n',
             f"[운영 문서]({GH}/aidev#readme) · [원장]({GH}/aidev/tree/main/state) · [실행 이력]({GH}/aidev/commits/main) · [원본 데이터]({SITE}/data/runs.jsonl)\n",
             f"## 오늘 ({today})\n"]
    if tr:
        lines.append(stats_html(c))
        lines.append(f"[{today} 보고 자세히 보기 →]({SITE}/reports/{today}/)\n")
        lines += ["| 시각 | 프로젝트 | 결과 |", "|---|---|---|"]
        for r in tr[-12:]:
            p = r.get("project", "")
            lines.append(f"| {(r.get('ts') or '')[11:16]} | {p} | {STATUS[classify(r.get('result'))]} {linkify(p, r.get('result'))} |")
        lines.append("")
    else:
        lines.append("아직 오늘 회차가 없습니다. 아래에서 최근 보고를 볼 수 있습니다.\n")
    lines += ["## 일일 보고\n", "| 날짜 | 회차 | 릴리즈 | 머지 | 변경 없음 | 실패 |", "|---|---|---|---|---|---|"]
    for d in days[:14]:
        cc = counts(by_day[d])
        lines.append(f"| [{d}]({SITE}/reports/{d}/) | {cc['total']} | {cc['released']} | {cc['merged']} | {cc['nochange']} | {cc['failed']} |")
    lines.append("")
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
        rel_s = (f"[{tag}]({GH}/{repo_name(p, r.get('result'))}/releases/tag/{tag})"
                 if tag and rel.get("status") == "released" else (rel.get("status") or ""))
        lines.append(f"| [{p}]({GH}/{repo_name(p, r.get('result'))}) | {(r.get('ts') or '')[:16].replace('T', ' ')} | "
                     f"{STATUS[classify(r.get('result'))]} {linkify(p, r.get('result'))} | {rel_s} |")
    lines.append("")
    lines.append("## FAQ\n")
    for q, a in FAQ:
        lines.append(f"<details><summary>{q}</summary><p>{a}</p></details>")
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
