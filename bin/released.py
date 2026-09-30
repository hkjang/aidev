#!/usr/bin/env python3
"""러너가 만든 릴리즈 **태그**를 저장소에서 직접 센다 — 러너 기록과 별개로.

왜 따로 세나: 러너의 회차 기록(runs.jsonl)은 한동안 승인 스윕에서 성공한 회차를 골라
버렸고(2026-09-26 수정), 그래서 2026-09-27 에 세어 보니 기록상 510건인데 저장소 태그로는
615건이었다. 자기 기록이 비어도 결과는 남는다 — 태그가 그 결과다.

가려내는 방법: 러너의 릴리즈는 모양이 정해져 있다 — `auto/…` 브랜치를 머지하고, 그 위에
릴리즈 커밋 하나를 올리고, 거기에 태그를 단다. 그래서 **태그 커밋과 그 직전 커밋** 둘만 보고
`auto/` 머지가 있으면 러너 것으로 센다. 창을 6커밋으로 넓혔더니 사람이 러너 머지 직후에 직접
릴리즈한 것까지 59건 끌어왔다 (2026-10-01: GoalForge v0.32.0 은 `feature/` 브랜치 머지에 달린
태그였다). 창별 집계: 1커밋 139 · 2커밋 668 · 3커밋 690 · 6커밋 727.

세는 것은 **태그**다. GitHub Release 객체는 그보다 적을 수 있다 — 태그는 달렸지만 CI 가 막아
릴리즈 게시까지 못 간 경우가 있다(`release tag held`). 2026-09-27 기준 러너 태그 661건,
같은 방법으로 가려낸 GitHub Release 615건이었다. 차이는 그 46건이다.

출력: docs/data/released.json
  { generated, since, total, by_project: {p: {runner, human, first, last}} }
사용: bin/released.py [--since 2026-09-02]
"""
import json, os, re, subprocess, sys
from datetime import datetime, timezone

HERE = os.path.dirname(os.path.abspath(__file__)); REPO_DIR = os.path.dirname(HERE)
DATA = os.path.join(REPO_DIR, "docs", "data")
ROOT = os.environ.get("ROOT", "/mnt/c/Users/USER/projects")
OUT = os.path.join(DATA, "released.json")
SINCE = "2026-09-02"
if "--since" in sys.argv:
    SINCE = sys.argv[sys.argv.index("--since") + 1]
# 러너 브랜치는 정확히 auto/<YYYY-MM-DD>-<HHMM> 이다 (run.sh: slug="auto/$RUN_DATE-$(date +%H%M)").
# `auto/` 로 시작하는 사람 브랜치도 있다 — postra 의 auto/2026-09-30-release 가 그랬고,
# 느슨한 패턴은 그 저장소의 사람 릴리즈 3건을 러너 것으로 셌다 (2026-10-01).
AUTO = re.compile(r"from [^/\s]+/auto/\d{4}-\d{2}-\d{2}-\d{4}\b")


def git(repo, *args):
    try:
        return subprocess.run(["git", "-C", repo, *args], capture_output=True, text=True, timeout=60).stdout
    except Exception:
        return ""


def main():
    runs_path = os.path.join(DATA, "runs.jsonl")
    projects = set()
    if os.path.exists(runs_path):
        for line in open(runs_path, encoding="utf-8"):
            line = line.strip()
            if not line:
                continue
            try:
                p = json.loads(line).get("project")
            except json.JSONDecodeError:
                continue
            if isinstance(p, str) and not p.startswith("("):
                projects.add(p)
    by = {}
    tot_r = tot_h = tot_held = 0
    for p in sorted(projects):
        repo = os.path.join(ROOT, p)
        if not os.path.isdir(os.path.join(repo, ".git")):
            continue
        # 원격에 있는 태그만 센다. release_project 는 릴리즈 커밋의 CI 를 확인하지 못하면
        # 태그를 만들어 두고 push 를 보류한다("태그 보류") — 그 태그는 로컬에만 남는다.
        # 로컬 태그를 그대로 세면 게시되지 않은 릴리즈가 성과로 잡힌다 (2026-09-27: 40건).
        remote = set()
        for line in git(repo, "ls-remote", "--tags", "origin").splitlines():
            ref = line.split("\t")[-1]
            if ref.startswith("refs/tags/") and not ref.endswith("^{}"):
                remote.add(ref[len("refs/tags/"):])
        out = git(repo, "for-each-ref", "--sort=creatordate", "--format=%(refname:short)\t%(creatordate:short)", "refs/tags")
        runner, human, held, tags = 0, 0, 0, []
        for line in out.splitlines():
            parts = line.split("\t")
            if len(parts) != 2 or parts[1] < SINCE:
                continue
            tag = parts[0]
            is_runner = bool(AUTO.search(git(repo, "log", "-2", "--format=%s", tag)))
            if tag not in remote:
                if is_runner:
                    held += 1          # 태그만 만들고 push 를 보류한 것 — 반쯤 끝난 릴리즈
                continue
            if is_runner:
                runner += 1; tags.append(tag)
            else:
                human += 1
        if runner or human or held:
            by[p] = {"runner": runner, "human": human, "held": held,
                     "first": tags[0] if tags else "", "last": tags[-1] if tags else ""}
            tot_r += runner; tot_h += human; tot_held += held
    os.makedirs(DATA, exist_ok=True)
    json.dump({"generated": datetime.now(timezone.utc).isoformat(timespec="seconds"), "since": SINCE,
               "total": tot_r, "human": tot_h, "held": tot_held, "projects": len([1 for v in by.values() if v["runner"]]),
               "by_project": by}, open(OUT, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    print(f"released: 게시된 러너 릴리즈 {tot_r}건 · 사람 {tot_h}건 · 태그만 만들고 보류 {tot_held}건 · 저장소 {len(by)}개 ({SINCE} 이후)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
