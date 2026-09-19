#!/usr/bin/env python3
"""실험 E2 · 지표 — 머지 뒤 수정 필요율 (post-merge corrective maintenance).

동기: 에이전트 PR 은 머지된 뒤 사람 코드보다 46~51% 더 많은 수정 유지보수를 필요로 했고, 리뷰 없이
머지되는 비율이 10%p 오를 때마다 유지보수 부담이 약 6% 늘었다는 보고가 있다 (Xia & Miller 2026;
Peralta et al. 2026). aidev 의 회귀 감시는 머지 뒤 2~48시간의 main CI 실패·되돌림만 본다. 이 스크립트는
러너가 머지한 PR 마다 그 PR 이 건드린 파일을 그 뒤 30일 안에 다시 고친 커밋(제목이 fix/revert/hotfix/
bug/수정/되돌림 류)이 있었는지를 센다 — "머지 뒤 수정 필요" 다.

출력: docs/data/postmerge.json  { generated, window_days, prs:[{pr, project, merge_sha, merged_at, files,
       corrective:[{sha, at, subject, by_runner}], days_observed, risk, approved_by}], summary:{...} }
사용: bin/postmerge.py [--no-fetch]   (헬스체크가 부른다; 저장소마다 origin 을 한 번 fetch 한다)
"""
import json, os, re, subprocess, sys
from collections import defaultdict
from datetime import datetime, timedelta, timezone

HERE = os.path.dirname(os.path.abspath(__file__)); REPO_DIR = os.path.dirname(HERE)
STATE = os.path.join(REPO_DIR, "state"); DATA = os.path.join(REPO_DIR, "docs", "data")
ROOT = os.environ.get("ROOT", "/mnt/c/Users/USER/projects")
OUT = os.path.join(DATA, "postmerge.json"); WINDOW = 30
FIX_RE = re.compile(r"\b(fix|hotfix|revert|regress|bug|broken|repair)\b|수정|되돌|고침|고쳤|회귀", re.I)
NOFETCH = "--no-fetch" in sys.argv


def git(repo, *args, timeout=120):
    try:
        return subprocess.run(["git", "-C", repo, *args], capture_output=True, text=True, timeout=timeout).stdout
    except Exception:
        return ""


def load_jsonl(p):
    rows = []
    if os.path.exists(p):
        for line in open(p, encoding="utf-8"):
            line = line.strip()
            if line:
                try:
                    rows.append(json.loads(line))
                except json.JSONDecodeError:
                    pass
    return rows


def main():
    runs = load_jsonl(os.path.join(DATA, "runs.jsonl"))
    approvals = load_jsonl(os.path.join(STATE, "approvals.jsonl"))
    appr_by = {a.get("pr"): a.get("by", "human") for a in approvals if a.get("sha")}
    prev = {}
    if os.path.exists(OUT):
        try:
            prev = {p["pr"]: p for p in json.load(open(OUT, encoding="utf-8")).get("prs", [])}
        except Exception:
            prev = {}
    now = datetime.now(timezone.utc)
    # 러너가 머지한 PR: PR 당 마지막 회차가 merged/releasing/release-ready
    last = {}
    for r in runs:
        if r.get("pr") and isinstance(r.get("project"), str) and not r["project"].startswith("("):
            last[r["pr"]] = r
    merged = {pr: r for pr, r in last.items() if r.get("outcome") in ("merged", "releasing", "release-ready")}
    by_project = defaultdict(list)
    for pr, r in merged.items():
        by_project[r["project"]].append((pr, r))
    prs = []
    for project, items in by_project.items():
        repo = os.path.join(ROOT, project)
        if not os.path.isdir(os.path.join(repo, ".git")):
            continue
        if not NOFETCH:
            git(repo, "fetch", "-q", "origin", timeout=180)
        head = git(repo, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").strip() or "origin/main"
        # 러너가 만든 커밋 집합: 러너 PR 의 머지 커밋 M 에 대해 M^1..M^2 가 그 PR 의 커밋들이다.
        # 이것이 없으면 러너의 다음 회차가 같은 파일을 'fix:' 로 고친 것까지 전부 "사람이 고쳤다" 로 읽힌다.
        runner_shas = set()
        for pr0, _ in items:
            m0 = git(repo, "log", head, "--merges", "--grep", f"Merge pull request #{pr0.rsplit('/', 1)[-1]} ", "--format=%H", "-1").strip()
            if m0:
                runner_shas.update(x for x in git(repo, "rev-list", f"{m0}^1..{m0}^2").split("\n") if x)
        for pr, r in items:
            num = pr.rsplit("/", 1)[-1]
            old = prev.get(pr)
            if old and old.get("days_observed", 0) >= WINDOW and old.get("merge_sha"):
                prs.append(old)  # 관찰 창이 닫힌 것은 다시 세지 않는다
                continue
            merge_sha = git(repo, "log", head, "--merges", "--grep", f"Merge pull request #{num} ", "--format=%H", "-1").strip()
            if not merge_sha:
                continue
            merged_at = git(repo, "log", "-1", "--format=%cI", merge_sha).strip()
            try:
                mdt = datetime.fromisoformat(merged_at)
            except Exception:
                continue
            files = [f for f in git(repo, "diff", "--name-only", f"{merge_sha}^1", merge_sha).split("\n") if f]
            corrective = []
            if files:
                log = git(repo, "log", f"{merge_sha}..{head}", "--no-merges", f"--until={(mdt + timedelta(days=WINDOW)).isoformat()}",
                          "--format=%H%x09%cI%x09%s", "--", *files[:200])
                for line in log.split("\n"):
                    if not line.strip():
                        continue
                    sha, at, subj = (line.split("\t") + ["", ""])[:3]
                    if FIX_RE.search(subj):
                        corrective.append({"sha": sha[:10], "at": at, "subject": subj[:120], "by_runner": sha in runner_shas})
            days = max(0, min(WINDOW, (now - mdt.astimezone(timezone.utc)).days))
            risk = ""
            m = re.search(r"risk=(\w+)", ((r.get("stages") or {}).get("review") or {}).get("reason", "") or "")
            if m:
                risk = m.group(1)
            prs.append({"pr": pr, "project": project, "merge_sha": merge_sha[:10], "merged_at": merged_at, "files": len(files),
                        "corrective": corrective[:20], "days_observed": days, "risk": risk,
                        "approved_by": appr_by.get(pr, "auto"), "campaign": r.get("campaign") or ""})
    # 요약: 관찰 7일 이상인 PR 중 수정 커밋이 하나라도 있던 비율, 층별
    def rate(rows):
        rows = [p for p in rows if p["days_observed"] >= 7]
        hit = sum(1 for p in rows if p["corrective"])
        human = sum(1 for p in rows if any(not c["by_runner"] for c in p["corrective"]))
        return {"n": len(rows), "with_fix": hit, "rate": round(hit / len(rows), 3) if rows else None,
                "with_human_fix": human, "human_rate": round(human / len(rows), 3) if rows else None}
    summary = {"all": rate(prs),
               "by_approval": {k: rate([p for p in prs if p["approved_by"] == k]) for k in ("auto", "human", "shepherd")},
               "by_risk": {k: rate([p for p in prs if p["risk"] == k]) for k in ("low", "medium", "high", "")},
               "by_campaign": {"campaign": rate([p for p in prs if p["campaign"]]), "exploratory": rate([p for p in prs if not p["campaign"]])},
               "median_days_to_first_fix": None}
    d2f = []
    for p in prs:
        if p["corrective"]:
            try:
                d2f.append((datetime.fromisoformat(p["corrective"][0]["at"]) - datetime.fromisoformat(p["merged_at"])).days)
            except Exception:
                pass
    if d2f:
        d2f.sort(); summary["median_days_to_first_fix"] = d2f[len(d2f) // 2]
    os.makedirs(DATA, exist_ok=True)
    json.dump({"generated": now.isoformat(timespec="seconds"), "window_days": WINDOW, "prs": prs, "summary": summary},
              open(OUT, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    print(f"postmerge: {len(prs)} merged PRs · {summary['all']} · by approval {summary['by_approval']} · median days to first fix {summary['median_days_to_first_fix']}")


if __name__ == "__main__":
    main()
