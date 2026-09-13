#!/usr/bin/env python3
"""러너가 스스로 이상하다고 느낀 것을 모은다.

러너는 오늘 하루에도 여러 번 "이상함" 을 기록해 놓고 그대로 지나쳤다. 같은 실패를
34번 되풀이했고, 대상이 0개인 캠페인을 "전부 완료" 로 닫았고, 통과할 수 없는 검사를
12번 다시 확인했다. 전부 로그에 남아 있었지만 읽는 사람이 없었다.

여기서는 판단하지 않는다. 값이 이상한 자리를 찾아 증거와 함께 내놓기만 한다.
판단은 bin/ask-claude.sh 가 클로드에게 맡긴다 — 규칙으로 적을 수 있는 것은 규칙으로
막는 편이 낫고, 규칙으로 적기 어려운 것만 물어본다.

    bin/triage.py            사람이 읽을 형태로
    bin/triage.py --json     한 줄에 하나씩 (ask-claude.sh 가 쓴다)
"""
from __future__ import annotations

import json
import re
import sys
from collections import Counter, defaultdict
from datetime import datetime, timedelta, timezone
from pathlib import Path

AIDEV = Path(__file__).resolve().parent.parent
RUNS = AIDEV / "docs" / "data" / "runs.jsonl"
USAGE = AIDEV / "docs" / "data" / "usage.jsonl"
CAMPAIGNS = AIDEV / "state" / "campaigns.json"
CRON = AIDEV / "logs" / "cron.log"
LOGS = AIDEV / "logs"

REPEAT_LIMIT = 3       # 같은 결과가 이만큼 이어지면 되풀이로 본다
STUCK_HOURS = 3        # 이만큼 회차가 없으면 멈춘 것으로 본다
OPEN_PR_LIMIT = 5      # 한 저장소에 이만큼 쌓이면 무언가 막고 있는 것이다
SKIP_DAYS = 2          # 작업 트리가 더러워 이만큼 연달아 건너뛰면 사람이 볼 일이다


def load(path: Path) -> list[dict]:
    if not path.is_file():
        return []
    out = []
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = line.strip()
        if line:
            try:
                out.append(json.loads(line))
            except json.JSONDecodeError:
                pass
    return out


def finding(kind: str, subject: str, summary: str, evidence: list[str], question: str) -> dict:
    return {"kind": kind, "subject": subject, "summary": summary,
            "evidence": evidence[:12], "question": question}


def repeated_failures(runs: list[dict]) -> list[dict]:
    """같은 프로젝트가 같은 이유로 계속 실패한다."""
    recent = runs[-400:]
    by_project: dict[str, list[dict]] = defaultdict(list)
    for r in recent:
        if r.get("project"):
            by_project[r["project"]].append(r)
    out = []
    for project, rows in by_project.items():
        tail = rows[-REPEAT_LIMIT:]
        if len(tail) < REPEAT_LIMIT:
            continue
        if not all(row.get("outcome") in ("error", "verify-failed") for row in tail):
            continue
        reasons = {re.sub(r"[0-9a-f]{7,}", "…", str(row.get("result", ""))) for row in tail}
        if len(reasons) != 1:
            continue           # 이유가 매번 다르면 되풀이가 아니라 여러 문제다
        out.append(finding(
            "repeated-failure", project,
            f"{project} 회차가 {len(tail)}번 이어서 같은 이유로 끝났습니다",
            [f"{row.get('date')} {row.get('outcome')} — {row.get('result')}" for row in rows[-6:]],
            "같은 자리에서 반복되는 원인이 무엇이고, 러너 쪽 문제인지 이 저장소 쪽 문제인지 "
            "가려 주세요. 그리고 다음에 할 일을 한 가지만 정해 주세요."))
    return out


def impossible_campaign(campaigns: dict) -> list[dict]:
    """캠페인 값이 그 자체로 앞뒤가 맞지 않는다."""
    out = []
    for c in campaigns.get("campaigns", []):
        if c.get("done"):
            continue
        cid = c.get("id", "?")
        problems = []
        if not c.get("projects"):
            problems.append("대상 목록이 비어 있습니다")
        if not c.get("until"):
            problems.append("기한이 비어 있습니다")
        if not str(c.get("goal", "")).strip():
            problems.append("목표가 비어 있습니다")
        if problems:
            out.append(finding(
                "campaign-inconsistent", cid,
                f"캠페인 {cid} 의 설정이 앞뒤가 맞지 않습니다: " + " · ".join(problems),
                [json.dumps({k: v for k, v in c.items() if k != "goal"}, ensure_ascii=False)],
                "이 값이 왜 비었는지, campaigns.json 의 문제인지 읽는 쪽(bin/run.sh 의 "
                "pick_campaign·campaign_claim 이 쓰는 TSV 열)의 문제인지 가려 주세요."))
    return out


def silent_runner(runs: list[dict]) -> list[dict]:
    """스케줄러는 도는데 회차가 없다."""
    if not runs:
        return []
    last = runs[-1].get("ts")
    try:
        when = datetime.fromisoformat(str(last))
    except ValueError:
        return []
    idle = datetime.now(timezone.utc).astimezone(when.tzinfo) - when
    if idle < timedelta(hours=STUCK_HOURS):
        return []
    tail = []
    if CRON.is_file():
        tail = CRON.read_text(encoding="utf-8", errors="replace").splitlines()[-15:]
    return [finding(
        "runner-idle", "(runner)",
        f"마지막 회차가 {int(idle.total_seconds() // 3600)}시간 전입니다",
        tail,
        "회차가 왜 시작되지 않는지 로그에서 가려 주세요. 스케줄러·잠금·상한·중지 파일 중 "
        "무엇인지 짚고, 사람이 손대야 하는 것이면 그렇게 말해 주세요.")]


def cost_spike(usage: list[dict], runs: list[dict]) -> list[dict]:
    """한 회차가 평소의 몇 배를 썼다."""
    costs = [(u.get("project"), u.get("phase"), u.get("cost_usd") or 0, u.get("ts")) for u in usage[-300:]]
    amounts = [c for _, _, c, _ in costs if c > 0]
    if len(amounts) < 20:
        return []
    amounts.sort()
    median = amounts[len(amounts) // 2]
    out = []
    for project, phase, cost, ts in costs[-40:]:
        if median > 0 and cost > median * 6:
            out.append(finding(
                "cost-spike", f"{project}/{phase}",
                f"{project} 의 {phase} 단계가 ${cost:.2f} 를 썼습니다 (보통 ${median:.2f})",
                [f"{ts} {project} {phase} ${cost:.2f}"],
                "이 단계가 왜 평소의 몇 배를 썼는지, 일감이 커서인지 같은 일을 되풀이해서인지 "
                "가려 주세요."))
    return out


def stuck_dirty(_: list[dict]) -> list[dict]:
    """작업 트리가 더러워 계속 건너뛰는 프로젝트.

    러너는 커밋되지 않은 변경이 있는 저장소를 건드리지 않는다 — 남의 작업 위에
    커밋을 얹지 않으려는 것이라 옳다. 다만 그 사실이 로그 한 줄로만 남아,
    저장소 하나가 며칠째 아무 개선도 받지 못하는 동안 아무도 알아채지 못했다
    (postra·Quantoss·DartFly 는 163회 연속으로 건너뛰어졌다).
    """
    if not LOGS.is_dir():
        return []
    cutoff = (datetime.now() - timedelta(days=SKIP_DAYS)).date()
    seen: dict[str, set] = defaultdict(set)
    for log in sorted(LOGS.glob("20*.log")):
        try:
            day = datetime.strptime(log.stem, "%Y-%m-%d").date()
        except ValueError:
            continue
        if day < cutoff:
            continue
        for line in log.read_text(encoding="utf-8", errors="replace").splitlines():
            hit = re.search(r"skip (\S+): dirty working tree", line)
            if hit:
                seen[hit.group(1)].add(day)
    out = []
    for project, days in sorted(seen.items()):
        if len(days) <= SKIP_DAYS:
            continue
        span = ", ".join(str(d) for d in sorted(days))
        out.append(finding(
            "stuck-dirty", project,
            f"{project} 은 커밋되지 않은 변경 때문에 {len(days)}일 연속 건너뛰어졌습니다",
            [f"건너뛴 날: {span}",
             f"확인: git -C {project} status --porcelain"],
            "이 저장소의 남아 있는 변경이 사람이 하던 작업인지, 예전 회차가 흘린 찌꺼기인지 "
            "가려 주세요. 찌꺼기라면 무엇을 버려도 되는지도 알려 주세요."))
    return out


def main() -> int:
    runs, usage = load(RUNS), load(USAGE)
    campaigns = {}
    if CAMPAIGNS.is_file():
        try:
            campaigns = json.loads(CAMPAIGNS.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            pass

    findings = (repeated_failures(runs) + impossible_campaign(campaigns)
                + silent_runner(runs) + cost_spike(usage, runs)
                + stuck_dirty(runs))

    if "--json" in sys.argv:
        for item in findings:
            print(json.dumps(item, ensure_ascii=False))
        return 0

    if not findings:
        print("이상 없음")
        return 0
    for item in findings:
        print(f"[{item['kind']}] {item['summary']}")
        for line in item["evidence"][:3]:
            print(f"    {line[:150]}")
    print(f"\n{len(findings)}건")
    return 0


if __name__ == "__main__":
    sys.exit(main())
