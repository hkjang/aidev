#!/usr/bin/env python3
"""기본 브랜치 CI 가 며칠째 빨간 저장소를 찾아 알리고, 원하면 복구를 작업 큐에 넣는다.

왜: madi 의 main CI 는 2026-09-24 13:41 부터 09-27 13:45 까지 사흘을 빨간 채 있었다.
그동안 그 저장소의 PR 은 전부 CI 실패로 막혔고 릴리즈도 멈췄는데, 아무도 알아채지 못했다.
러너는 자기 PR 의 CI 만 보고 기본 브랜치의 상태는 보지 않는다 — 회차마다 "CI failed" 를
찍으면서도 그 원인이 저장소 전체에 있다는 것은 세지 않았다.

이 스크립트는 기본 브랜치의 마지막 결론만 본다(워크플로별). 실패가 STALE_DAYS(기본 1)일
넘게 이어지면 알리고, --queue 를 주면 실패한 잡 이름과 로그 앞부분을 명세로 넣어
state/run-queue.tsv 에 복구를 배정한다. 같은 저장소는 QUIET_DAYS(기본 3)일에 한 번만.

사용: bin/ci-watch.py [--queue] [--quiet]
"""
import json, os, re, subprocess, sys
from datetime import datetime, timedelta, timezone

HERE = os.path.dirname(os.path.abspath(__file__)); REPO_DIR = os.path.dirname(HERE)
STATE = os.path.join(REPO_DIR, "state"); DATA = os.path.join(REPO_DIR, "docs", "data")
ROOT = os.environ.get("ROOT", "/mnt/c/Users/USER/projects")
SEEN = os.path.join(STATE, ".ci-watch-seen")
STALE_DAYS = int(os.environ.get("CI_STALE_DAYS", "1"))
QUIET_DAYS = int(os.environ.get("CI_QUIET_DAYS", "3"))
QUEUE = "--queue" in sys.argv
QUIET = "--quiet" in sys.argv


def sh(*a, **kw):
    try:
        return subprocess.run(a, capture_output=True, text=True, timeout=90, **kw).stdout
    except Exception:
        return ""


def excluded(name):
    f = os.path.join(STATE, "exclude.txt")
    if not os.path.exists(f):
        return False
    for line in open(f, encoding="utf-8"):
        pat = line.split("#", 1)[0].strip()
        if pat and re.fullmatch(pat, name):
            return True
    return False


def main():
    projects = set()
    runs = os.path.join(DATA, "runs.jsonl")
    if os.path.exists(runs):
        for line in open(runs, encoding="utf-8"):
            line = line.strip()
            if not line:
                continue
            try:
                p = json.loads(line).get("project")
            except json.JSONDecodeError:
                continue
            if isinstance(p, str) and not p.startswith("(") and not excluded(p):
                projects.add(p)
    seen = {}
    if os.path.exists(SEEN):
        try:
            seen = json.load(open(SEEN, encoding="utf-8"))
        except Exception:
            seen = {}
    now = datetime.now(timezone.utc)
    bad = []
    for p in sorted(projects):
        repo = os.path.join(ROOT, p)
        if not os.path.isdir(os.path.join(repo, ".git")):
            continue
        base = (sh("git", "-C", repo, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").strip() or "origin/main").split("/", 1)[-1]
        out = sh("gh", "run", "list", "-R", f"hkjang/{p}", "--branch", base, "--limit", "30",
                 "--json", "name,conclusion,createdAt,databaseId")
        try:
            rows = [r for r in json.loads(out or "[]") if r.get("conclusion")]
        except Exception:
            continue
        # 워크플로마다 가장 최근 결론만 본다
        latest = {}
        for r in rows:
            latest.setdefault(r["name"], r)
        for name, r in latest.items():
            if r["conclusion"] != "failure":
                continue
            try:
                when = datetime.fromisoformat(r["createdAt"].replace("Z", "+00:00"))
            except Exception:
                continue
            days = (now - when).days
            if days < STALE_DAYS:
                continue
            bad.append({"project": p, "base": base, "workflow": name, "days": days,
                        "run_id": r.get("databaseId"), "at": r["createdAt"][:16]})
    print(f"ci-watch: 기본 브랜치 CI 가 {STALE_DAYS}일 넘게 실패인 저장소·워크플로 {len(bad)}건")
    fresh = []
    for b in bad:
        key = f"{b['project']}|{b['workflow']}"
        last = seen.get(key)
        if last:
            try:
                if (now - datetime.fromisoformat(last)).days < QUIET_DAYS:
                    continue
            except Exception:
                pass
        seen[key] = now.isoformat(timespec="seconds")
        fresh.append(b)
    for b in fresh:
        # 실패한 잡과 로그 앞부분을 붙여 두면 복구 회차가 재현부터 시작할 수 있다
        job = step = detail = ""
        if b["run_id"]:
            jj = sh("gh", "run", "view", str(b["run_id"]), "-R", f"hkjang/{b['project']}", "--json", "jobs")
            try:
                jobs = json.loads(jj or "{}").get("jobs", [])
            except Exception:
                jobs = []
            for j in jobs:
                if j.get("conclusion") == "failure":
                    job = j.get("name", "")
                    step = next((s.get("name", "") for s in j.get("steps", []) if s.get("conclusion") == "failure"), "")
                    log = sh("gh", "api", f"repos/hkjang/{b['project']}/actions/jobs/{j.get('databaseId')}/logs")
                    detail = " / ".join(l.split("Z ", 1)[-1][:110] for l in log.splitlines()
                                        if re.search(r"error|failed|cannot|not found", l, re.I))[:600]
                    break
        b["job"], b["step"], b["detail"] = job, step, detail
        print(f"  {b['project']} [{b['base']}] {b['workflow']} — {b['days']}일째 실패 ({b['at']}) · {job}/{step}")
        # 30일 넘게 커밋이 없는 저장소는 회차 후보가 아니다 — 큐에 넣어도 잡히지 않고 줄만 남는다.
        last = sh("git", "-C", os.path.join(ROOT, b["project"]), "log", "-1", "--format=%ct").strip()
        stale_repo = (not last) or (now - datetime.fromtimestamp(int(last), timezone.utc)).days > 30
        if stale_repo:
            print("    (30일 넘게 커밋이 없는 저장소 — 큐에 넣지 않는다)")
        if QUEUE and not stale_repo:
            q = os.path.join(STATE, "run-queue.tsv")
            have = os.path.exists(q) and any(l.split("\t")[0] == b["project"] for l in open(q, encoding="utf-8"))
            if not have:
                spec = (f"{b['project']} 의 기본 브랜치({b['base']}) CI 가 {b['days']}일째 실패입니다. 이번 회차는 이것만 고치세요.\\n\\n"
                        f"실패: 워크플로 '{b['workflow']}' · 잡 '{job}' · 단계 '{step}'\\n"
                        f"로그에서 뽑은 줄: {detail or '(없음)'}\\n\\n"
                        "할 일: 1) 먼저 로컬에서 같은 실패를 재현하고 그 출력을 원장의 '실패 재현' 에 붙이세요. "
                        "2) 원인을 고치세요. 3) 같은 명령으로 통과를 확인하세요.\\n"
                        "제약: 실패를 없애려고 검사를 끄거나 느슨하게 만들지 마세요(테스트 skip·게이트 제거·버전 고정 해제 금지). "
                        "원인을 고치는 변경만 하고, 무관한 파일은 건드리지 마세요.")
                with open(q, "a", encoding="utf-8") as f:
                    f.write(f"{b['project']}\tCI 복구(자동 배정): {b['workflow']}\t0\tnormal\t{spec}\n")
                print(f"    → 작업 큐에 배정")
    json.dump(seen, open(SEEN, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    if fresh and not QUIET:
        text = "🚨 기본 브랜치 CI 가 며칠째 빨갛습니다\n" + "\n".join(
            f"• {b['project']} [{b['workflow']}] {b['days']}일째 — {b['job']}/{b['step']}" for b in fresh[:5])
        if QUEUE:
            text += "\n복구를 작업 큐에 넣었습니다 — 다음 회차가 잡습니다."
        subprocess.run([os.path.join(HERE, "tg.sh"), text], capture_output=True, timeout=30)
    return 0


if __name__ == "__main__":
    sys.exit(main())
