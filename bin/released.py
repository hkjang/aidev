#!/usr/bin/env python3
"""러너가 만든 릴리즈 **태그**를 저장소에서 직접 센다 — 러너 기록과 별개로.

왜 따로 세나: 러너의 회차 기록(runs.jsonl)은 한동안 승인 스윕에서 성공한 회차를 골라
버렸고(2026-09-26 수정), 그래서 2026-09-27 에 세어 보니 기록상 510건인데 저장소 태그로는
615건이었다. 자기 기록이 비어도 결과는 남는다 — 태그가 그 결과다.

가려내는 방법: 태그가 가리키는 커밋에서 6개 뒤까지 러너 브랜치(`auto/<날짜>-<시각>`) 머지가
있으면 러너가 만든 릴리즈로 본다. 사람이 직접 만든 브랜치(`feat/…`, `fix/…`)는 빠진다.
gh 를 부르지 않고 로컬 클론의 태그만 읽으므로 공짜다(러너가 릴리즈마다 --tags 를 받아 둔다).

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
AUTO = re.compile(r"from [^/\s]+/auto/")


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
    tot_r = tot_h = 0
    for p in sorted(projects):
        repo = os.path.join(ROOT, p)
        if not os.path.isdir(os.path.join(repo, ".git")):
            continue
        out = git(repo, "for-each-ref", "--sort=creatordate", "--format=%(refname:short)\t%(creatordate:short)", "refs/tags")
        runner, human, tags = 0, 0, []
        for line in out.splitlines():
            parts = line.split("\t")
            if len(parts) != 2 or parts[1] < SINCE:
                continue
            tag = parts[0]
            if AUTO.search(git(repo, "log", "-6", "--format=%s", tag)):
                runner += 1; tags.append(tag)
            else:
                human += 1
        if runner or human:
            by[p] = {"runner": runner, "human": human,
                     "first": tags[0] if tags else "", "last": tags[-1] if tags else ""}
            tot_r += runner; tot_h += human
    os.makedirs(DATA, exist_ok=True)
    json.dump({"generated": datetime.now(timezone.utc).isoformat(timespec="seconds"), "since": SINCE,
               "total": tot_r, "human": tot_h, "projects": len([1 for v in by.values() if v["runner"]]),
               "by_project": by}, open(OUT, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    print(f"released: 러너 태그 {tot_r}건 · 사람 태그 {tot_h}건 · 저장소 {len(by)}개 ({SINCE} 이후) → {OUT}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
