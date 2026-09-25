#!/usr/bin/env python3
"""사라진 승인 스윕 회차를 기록에 되살린다 (2026-09-25 일회성 복구, 재실행해도 안전).

무엇이 없어졌나: bin/run.sh 의 승인 스윕은 "아무것도 달라지지 않은 승인 확인" 을 회차로
기록하지 않으려고 `[ "$OUTCOME" = merged ]` 만 통과시켰다. 그런데 머지 뒤 릴리즈까지 가면
OUTCOME 이 releasing·release-ready 로 바뀐다 — **가장 잘 끝난 회차가 골라서 지워졌다.**
usage.jsonl 에는 남아 있지만 runs.jsonl 에는 없는 `*-approve` 회차가 71건($127)이었고,
대시보드·성과 쿨다운·bin/postmerge.py(논문 E2)가 모두 그만큼 적게 세고 있었다.

되살리는 방법: usage 기록의 (프로젝트, 시각) 으로 그 저장소의 머지 커밋을 찾아 PR 번호를
복원한다. 시각 창 안에 머지 커밋이 하나일 때만 쓰고, 애매하면 건너뛴다 — 틀린 PR 을 붙이면
머지 뒤 분석이 엉뚱한 파일을 본다.

    bin/backfill-approve-runs.py            무엇을 되살릴지만 보여 준다
    bin/backfill-approve-runs.py --apply    runs.jsonl 에 덧붙인다
"""
import json, os, subprocess, sys
from datetime import datetime, timedelta

HERE = os.path.dirname(os.path.abspath(__file__)); REPO_DIR = os.path.dirname(HERE)
DATA = os.path.join(REPO_DIR, "docs", "data")
ROOT = os.environ.get("ROOT", "/mnt/c/Users/USER/projects")
RUNS, USAGE = os.path.join(DATA, "runs.jsonl"), os.path.join(DATA, "usage.jsonl")
APPLY = "--apply" in sys.argv
WINDOW_BEFORE, WINDOW_AFTER = timedelta(minutes=20), timedelta(minutes=90)


def load(p):
    out = []
    for line in open(p, encoding="utf-8"):
        line = line.strip()
        if line:
            try:
                out.append(json.loads(line))
            except json.JSONDecodeError:
                pass
    return out


def git(repo, *args):
    try:
        return subprocess.run(["git", "-C", repo, *args], capture_output=True, text=True, timeout=60).stdout
    except Exception:
        return ""


def main():
    runs = load(RUNS)
    have_ids = {r.get("run_id") for r in runs}
    # 이미 "머지됨" 으로 끝난 기록이 있는 PR 만 건너뛴다. PR 을 연 회차(review-pending)는
    # 그대로 두고 그 뒤 머지를 덧붙여야 postmerge.py 가 그 PR 을 머지된 것으로 센다.
    MERGED = ("merged", "releasing", "release-ready")
    have_prs = {r.get("pr") for r in runs if r.get("pr") and r.get("outcome") in MERGED}
    # 사라진 승인 회차: usage 에만 있는 *-approve
    lost = {}
    for u in load(USAGE):
        rid = u.get("run_id") or ""
        if not rid.endswith("-approve") or rid in have_ids:
            continue
        e = lost.setdefault(rid, {"project": u.get("project"), "ts": u.get("ts"), "cost": 0.0, "date": u.get("date")})
        e["cost"] += u.get("cost_usd") or 0
        if u.get("ts", "") > (e["ts"] or ""):
            e["ts"] = u["ts"]
    print(f"기록이 없는 승인 회차 {len(lost)}건")

    added, skipped = [], []
    for rid, e in sorted(lost.items(), key=lambda x: x[1]["ts"] or ""):
        repo = os.path.join(ROOT, e["project"] or "")
        if not os.path.isdir(os.path.join(repo, ".git")):
            skipped.append((rid, "저장소 없음")); continue
        try:
            t = datetime.fromisoformat(e["ts"])
        except Exception:
            skipped.append((rid, "시각 파싱 실패")); continue
        head = git(repo, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").strip() or "origin/main"
        log = git(repo, "log", head, "--merges", "--format=%H%x09%cI%x09%s",
                  f"--since={(t - WINDOW_BEFORE).isoformat()}", f"--until={(t + WINDOW_AFTER).isoformat()}")
        cands = []
        for line in log.splitlines():
            parts = line.split("\t")
            if len(parts) < 3 or "Merge pull request #" not in parts[2]:
                continue
            num = parts[2].split("Merge pull request #", 1)[1].split()[0]
            url = f"https://github.com/hkjang/{e['project']}/pull/{num}"
            if url in have_prs:          # 이미 다른 회차가 이 PR 을 들고 있다
                continue
            cands.append((parts[1], url))
        if len(cands) != 1:
            skipped.append((rid, f"머지 커밋 후보 {len(cands)}개 — 확실할 때만 쓴다")); continue
        at, url = cands[0]
        have_prs.add(url)
        added.append({"ts": e["ts"], "date": e["date"], "project": e["project"],
                      "result": f"merged {url} (approved)", "outcome": "release-ready",
                      "run_id": rid, "base_sha": "", "head_sha": "", "pr": url,
                      "stages": {"merge": {"state": "done", "reason": "승인 스윕(기록 복구)", "at": at}},
                      "campaign": "", "autonomy": "", "arm": "", "experiment": "",
                      "engine": "claude", "backfilled": "2026-09-25"})

    print(f"  되살릴 수 있음 {len(added)}건 · 건너뜀 {len(skipped)}건")
    for r in added[:5]:
        print(f"   {r['ts'][:16]}  {r['project']:16s} {r['pr']}")
    if skipped[:3]:
        print("  건너뛴 예:", *[f"{r} ({w})" for r, w in skipped[:3]], sep="\n   ")
    if not APPLY:
        print("\n--apply 를 주면 runs.jsonl 에 덧붙인다"); return 0
    with open(RUNS, "a", encoding="utf-8") as f:
        for r in added:
            f.write(json.dumps(r, ensure_ascii=False) + "\n")
    print(f"\nruns.jsonl 에 {len(added)}건 추가. bin/report.py 와 bin/postmerge.py 를 다시 돌리면 반영된다.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
