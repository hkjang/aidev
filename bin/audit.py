#!/usr/bin/env python3
"""측정 자기 감사 — 독립된 두 출처를 맞춰 보고 어긋나면 적는다.

왜: 2026-09-25~27 사이에 이 러너의 측정 결함 여섯 개가 사람 손으로 발견됐다.

  1. 승인 스윕이 머지+릴리즈까지 간 회차만 골라 기록에서 지웠다 (71건)
  2. 초기 형식 회차 249건에 outcome 이 없어 모든 집계에서 빠졌다
  3. 릴리즈 커밋이 "사람이 고친 것" 으로 집계돼 재수정률이 8%p 부풀었다
  4. 크기별 재수정률이 캠페인 PR(생존 편향)에 가려져 2.4배로 낮게 보였다
  5. 실험 arm 둘이 노출 0·7% 인데 null 결과를 내고 있었다
  6. 게시되지 않은 로컬 태그가 성과로 잡히고 그 저장소의 릴리즈를 막았다

여섯 개의 공통점은 "두 출처가 어긋나는데 아무도 맞춰 보지 않았다" 다. 그래서 맞춰 보는 일을
사람이 아니라 6시간마다 도는 스크립트에 맡긴다. 고치지는 않는다 — 어긋난 사실만 적고 알린다.

출력: docs/data/audit.json  { generated, findings: [{id, severity, summary, detail}] }
사용: bin/audit.py [--quiet]   (헬스체크가 부른다; 새 findings 만 알림)
"""
import json, os, re, subprocess, sys
from datetime import datetime, timezone

HERE = os.path.dirname(os.path.abspath(__file__)); REPO_DIR = os.path.dirname(HERE)
DATA = os.path.join(REPO_DIR, "docs", "data"); STATE = os.path.join(REPO_DIR, "state")
OUT = os.path.join(DATA, "audit.json")
QUIET = "--quiet" in sys.argv


def jsonl(p):
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


def load(p, default=None):
    try:
        return json.load(open(p, encoding="utf-8"))
    except Exception:
        return default if default is not None else {}


def main():
    runs = jsonl(os.path.join(DATA, "runs.jsonl"))
    usage = jsonl(os.path.join(DATA, "usage.jsonl"))
    rel = load(os.path.join(DATA, "released.json"))
    pm = load(os.path.join(DATA, "postmerge.json"))
    exp = load(os.path.join(STATE, "experiment.json"))
    f = []

    def add(fid, sev, summary, detail=""):
        f.append({"id": fid, "severity": sev, "summary": summary, "detail": detail})

    # 1) 게시된 릴리즈(원격 태그) vs 회차 기록. 기록 경로가 빠지면 여기가 벌어진다.
    if rel.get("total"):
        recorded = sum(1 for r in runs if re.search(r"released v", r.get("result") or ""))
        gap = rel["total"] - recorded
        # 과거 잔재(2026-09-26 승인 스윕 결함 이전)는 줄지 않는다 — 그 수를 기준선으로 두고
        # **늘어날 때만** 알린다. 안 그러면 고칠 수 없는 항목이 영구히 high 로 떠 있어
        # 다른 알림을 덮는다 (2026-10-01: 기준선 64건).
        baseline = 0
        bf = os.path.join(STATE, ".audit-release-gap-baseline")
        if os.path.exists(bf):
            try:
                baseline = int(open(bf, encoding="utf-8").read().strip() or 0)
            except Exception:
                baseline = 0
        else:
            open(bf, "w", encoding="utf-8").write(str(gap)); baseline = gap
        if gap > baseline + 5:
            add("release-gap", "high",
                f"게시 릴리즈 {rel['total']}건인데 회차 기록은 {recorded}건 — {gap}건이 빠졌다 (기준선 {baseline}건보다 {gap - baseline}건 늘었다)",
                "회차를 기록하는 경로 어딘가가 결말에 따라 기록을 건너뛴다. 기준선을 다시 잡으려면 state/.audit-release-gap-baseline 을 지운다.")
        elif gap > baseline:
            add("release-gap", "low", f"기록 누락이 기준선({baseline}건)보다 {gap - baseline}건 늘었다 — 아직 작지만 지켜볼 것")

    # 2) 게시하지 못한 로컬 태그. 남아 있으면 그 저장소가 영구히 '릴리즈할 것 없음' 이 된다.
    if rel.get("held"):
        add("held-tags", "high", f"원격에 없는 릴리즈 태그 {rel['held']}건 — 그 저장소의 다음 릴리즈를 막는다",
            "release_project 의 describe 검사가 로컬 태그를 보므로, 보류된 태그가 base 끝을 가리키면 릴리즈가 영구히 멈춘다.")

    # 3) outcome 이 빈 회차. 이후 모든 집계에서 빠진다.
    nulls = [r for r in runs if r.get("outcome") is None and isinstance(r.get("project"), str)]
    if len(nulls) > 5:
        add("null-outcome", "medium", f"outcome 이 빈 회차 {len(nulls)}건 — 대시보드·쿨다운·머지 뒤 분석에서 빠진다",
            f"가장 최근: {nulls[-1].get('ts','?')[:16]} {nulls[-1].get('project','?')}. bin/normalize-legacy-runs.py 로 결과 문장에서 복원할 수 있다.")

    # 4) 비용 기록 vs 토큰. 정액제 추정이라 완벽하지 않지만, 크게 어긋나면 상한 판단이 흔들린다.
    pairs = []
    for u in usage:
        c = u.get("cost_usd")
        if not c:
            continue
        tok = (u.get("cache_read") or 0) + (u.get("cache_create") or 0) * 3 + (u.get("output_tokens") or 0) * 12
        if tok > 1000:
            pairs.append(c / tok * 1e6)
    if len(pairs) > 50:
        pairs.sort(); med = pairs[len(pairs) // 2]
        odd = sum(1 for x in pairs if x > med * 4 or x < med / 4)
        if odd / len(pairs) > 0.05:
            add("cost-drift", "medium", f"비용/토큰 비율 이상치가 {odd / len(pairs) * 100:.0f}% — 일일 비용 상한 판단이 흔들릴 수 있다")

    # 5) 실험 arm 의 처치 노출. 돌지 않는 기능을 끄는 arm 은 null 결과를 만들어 낸다.
    if exp.get("enabled"):
        feat = {"no-scout": "scout", "no-repair": "repair", "no-arbiter": "arbiter", "codex-critic": "review"}
        base = [r for r in runs if r.get("arm") == "baseline"]
        for arm, key in feat.items():
            a = exp.get("arms", {}).get(arm)
            if not a or (a.get("weight", 1) == 0) or not base:
                continue
            seen = sum(1 for r in base if ((r.get("stages") or {}).get(key) or {}).get("state"))
            share = seen / len(base)
            if share < 0.15:
                add(f"arm-exposure-{arm}", "medium",
                    f"실험 arm {arm} 의 처치 노출이 {share * 100:.0f}% — 그 arm 의 null 결과는 '효과 없음' 이 아니다",
                    f"baseline {len(base)}회차 중 {key} 단계가 돈 것은 {seen}회다.")

    # 6) 머지 뒤 분석의 러너 커밋 귀속. 릴리즈 커밋이 '사람 수정' 으로 새면 여기가 튄다.
    s = (pm.get("summary") or {}).get("all") or {}
    if s.get("n", 0) >= 50 and (s.get("human_rate") or 0) > 0.45:
        add("postmerge-attribution", "medium",
            f"머지 뒤 30일 사람 수정률이 {s['human_rate'] * 100:.0f}% — 러너 커밋이 사람 수정으로 새는지 확인할 것",
            "2026-09-27 에 릴리즈 커밋(태그가 가리키는 커밋)이 사람 수정으로 집계되던 결함을 고쳤다. 같은 종류가 또 생기면 이 값이 먼저 튄다.")

    # 7) 캠페인 '완료' 와 실제 적용. 완료 알림이 성공으로 읽히지 않게.
    camps = load(os.path.join(STATE, "campaigns.json")).get("campaigns", [])
    bad = [c for c in camps if c.get("landed") and c["landed"].get("prs", 0) >= 10
           and c["landed"].get("merged", 0) / max(c["landed"]["prs"], 1) < 0.35]
    if bad:
        add("campaign-landing", "low",
            f"닫힌 캠페인 {len(bad)}개의 적용률이 35% 미만 — PR 은 열렸지만 표준이 들어가지 않았다",
            " · ".join(f"{c['id']} {c['landed']['merged']}/{c['landed']['prs']}" for c in bad[:4]))

    os.makedirs(DATA, exist_ok=True)
    prev = {x["id"] for x in (load(OUT).get("findings") or [])}
    json.dump({"generated": datetime.now(timezone.utc).isoformat(timespec="seconds"), "findings": f},
              open(OUT, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    print(f"audit: {len(f)}건" + ("" if not f else " — " + " · ".join(x["id"] for x in f)))
    new = [x for x in f if x["id"] not in prev]
    if new and not QUIET:
        text = "🔍 측정 자기 감사 — 새로 어긋난 것 %d건\n" % len(new)
        text += "\n".join(f"[{x['severity']}] {x['summary']}" for x in new[:4])
        subprocess.run([os.path.join(HERE, "tg.sh"), text], capture_output=True, timeout=30)
    return 0


if __name__ == "__main__":
    sys.exit(main())
