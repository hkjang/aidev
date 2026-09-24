#!/usr/bin/env python3
"""비교 실험 분석 — state/experiment.json 의 arm 별로 회차 지표를 모아 표와 신뢰구간을 낸다.

사용: bin/exp-analyze.py [실험 id]   → docs/paper/experiments/ab-<id>.json, ab-<id>.md (논문 표 그대로)
지표(회차 단위): 검증 통과율, 비평 거절(1회 이상)률, PR 도달률, 머지율(merged/release-ready), 검토 대기율,
변경 없음률, 회차 비용(usage.jsonl run_id 합), 머지당 비용, 과제서 채택률, 수리 성공률, 중재 결과,
머지 뒤 30일 사람 수정률(postmerge.json 조인). 비율에는 부트스트랩 95% 신뢰구간, baseline 과의 차이도 함께.
"""
import json, os, random, sys
from collections import defaultdict

HERE = os.path.dirname(os.path.abspath(__file__)); REPO_DIR = os.path.dirname(HERE)
STATE = os.path.join(REPO_DIR, "state"); DATA = os.path.join(REPO_DIR, "docs", "data")
OUTD = os.path.join(REPO_DIR, "docs", "paper", "experiments")


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


def st(r, k):
    return ((r.get("stages") or {}).get(k) or {}).get("state")


def boot_ci(vals, iters=2000, seed=7):
    if not vals:
        return (None, None)
    rnd = random.Random(seed); n = len(vals); means = []
    for _ in range(iters):
        s = [vals[rnd.randrange(n)] for _ in range(n)]
        means.append(sum(s) / n)
    means.sort()
    return (round(means[int(0.025 * iters)], 3), round(means[int(0.975 * iters)], 3))


def rate(vals):
    return round(sum(vals) / len(vals), 3) if vals else None


def main():
    exp = json.load(open(os.path.join(STATE, "experiment.json"), encoding="utf-8"))
    exp_id = sys.argv[1] if len(sys.argv) > 1 else exp["id"]
    runs = [r for r in load_jsonl(os.path.join(DATA, "runs.jsonl")) if r.get("experiment") == exp_id and r.get("arm")]
    usage = load_jsonl(os.path.join(DATA, "usage.jsonl"))
    cost_by_run = defaultdict(float)
    for u in usage:
        if u.get("run_id"):
            cost_by_run[u["run_id"]] += float(u.get("cost_usd") or 0)
    # 9-24 이전에는 회차에 engine 이 없다. 구현 단계 사용 기록의 비용이 비어 있으면 코덱스 대체다
    # (run_codex 가 합성하는 결과에는 total_cost_usd 가 없다) — 그걸로 과거 회차의 엔진을 되살린다.
    codex_runs = {u["run_id"] for u in usage
                  if u.get("phase") == "improve" and u.get("cost_usd") is None and u.get("run_id")}
    for r in runs:
        if not r.get("engine"):
            r["engine"] = "codex" if r.get("run_id") in codex_runs else "claude"
    pm = {}
    try:
        pm = {p["pr"]: p for p in json.load(open(os.path.join(DATA, "postmerge.json"), encoding="utf-8")).get("prs", [])}
    except Exception:
        pass
    arms = list(exp["arms"].keys())
    by_arm = {a: [r for r in runs if r["arm"] == a] for a in arms}
    merged_out = ("merged", "releasing", "release-ready")

    def metrics(rows):
        committed = [r for r in rows if st(r, "verify")]
        reviewed = [r for r in rows if st(r, "review") in ("approved", "rejected")]
        scouted = [r for r in rows if st(r, "scout") == "done"]
        briefed = [r for r in scouted if st(r, "brief") in ("accepted", "fallback", "rejected")]
        repaired = [r for r in rows if st(r, "repair")]
        arb = [r for r in rows if st(r, "arbiter")]
        merged = [r for r in rows if r.get("outcome") in merged_out]
        costs = [cost_by_run.get(r.get("run_id"), 0.0) for r in rows]
        pm_rows = [pm[r["pr"]] for r in merged if r.get("pr") in pm and pm[r["pr"]].get("days_observed", 0) >= 7]
        # 관찰 3일 기준 잠정치 — 실험 초기에는 7일 기준이 전부 비어 "측정 실패" 로 오독된다 (2026-09-24)
        pm_rows3 = [pm[r["pr"]] for r in merged if r.get("pr") in pm and pm[r["pr"]].get("days_observed", 0) >= 3]
        m = {
            "n": len(rows),
            "verify_pass": {"n": len(committed), "rate": rate([st(r, "verify") == "passed" for r in committed]), "ci": boot_ci([1 if st(r, "verify") == "passed" else 0 for r in committed])},
            "critic_rejected_ever": {"n": len(reviewed), "rate": rate([1 if (st(r, "review") == "rejected" or st(r, "repair")) else 0 for r in reviewed]), "ci": boot_ci([1 if (st(r, "review") == "rejected" or st(r, "repair")) else 0 for r in reviewed])},
            "pr_reached": {"n": len(rows), "rate": rate([1 if r.get("pr") else 0 for r in rows]), "ci": boot_ci([1 if r.get("pr") else 0 for r in rows])},
            "merged": {"n": len(rows), "rate": rate([1 if r.get("outcome") in merged_out else 0 for r in rows]), "ci": boot_ci([1 if r.get("outcome") in merged_out else 0 for r in rows])},
            "review_pending": {"n": len(rows), "rate": rate([1 if r.get("outcome") == "review-pending" else 0 for r in rows])},
            "no_change": {"n": len(rows), "rate": rate([1 if r.get("outcome") == "no-change" else 0 for r in rows])},
            "cost_per_round": {"n": len(rows), "mean": round(sum(costs) / len(costs), 2) if costs else None, "ci": boot_ci(costs)},
            "cost_per_merge": round(sum(costs) / len(merged), 2) if merged else None,
            "brief_accepted": {"n": len(briefed), "rate": rate([1 if st(r, "brief") == "accepted" else 0 for r in briefed])},
            "repair_success": {"n": len(repaired), "rate": rate([1 if st(r, "repair") == "done" else 0 for r in repaired])},
            "arbiter": {"n": len(arb), "approved": sum(1 for r in arb if st(r, "arbiter") == "approved")},
            "postmerge_human_fix_30d": {"n": len(pm_rows), "rate": rate([1 if any(not c["by_runner"] for c in p["corrective"]) else 0 for p in pm_rows])},
            "postmerge_fix_3d": {"n": len(pm_rows3), "rate": rate([1 if p["corrective"] else 0 for p in pm_rows3])},
            "pm_observed": {"n": len([r for r in merged if r.get("pr") in pm]),
                            "max_days": max([pm[r["pr"]].get("days_observed", 0) for r in merged if r.get("pr") in pm], default=0)},
        }
        return m

    by_arm_claude = {a: [r for r in by_arm[a] if r.get("engine") != "codex"] for a in arms}
    result = {"experiment": exp_id, "questions": exp.get("questions", {}), "arms": {a: metrics(by_arm[a]) for a in arms},
              # 엔진을 섞지 않은 비교 — 역할의 효과를 보려면 같은 엔진끼리 봐야 한다
              "arms_claude": {a: metrics(by_arm_claude[a]) for a in arms},
              "campaign_share": {a: rate([1 if r.get("campaign") else 0 for r in by_arm[a]]) for a in arms},
              # 클로드가 한도에 걸린 날은 같은 프롬프트를 코덱스가 돌았다(2026-09-20~22 에 163회차).
              # 엔진이 섞인 비율이 arm 마다 다르면 그 차이는 역할이 아니라 엔진의 차이다.
              "codex_share": {a: rate([1 if (r.get("engine") == "codex") else 0 for r in by_arm[a]]) for a in arms},
              "projects": {a: len({r["project"] for r in by_arm[a]}) for a in arms}, "total_runs": len(runs)}
    base = result["arms"].get("baseline", {})

    def cell(a, key, sub="rate"):
        v = result["arms"][a].get(key)
        if isinstance(v, dict):
            x = v.get(sub); n = v.get("n")
            if x is None:
                return "—"
            s = f"{x:.0%}" if sub == "rate" else f"${x:.2f}"
            ci = v.get("ci")
            if ci and ci[0] is not None:
                s += f" [{ci[0]:.0%}, {ci[1]:.0%}]" if sub == "rate" else f" [{ci[0]:.2f}, {ci[1]:.2f}]"
            return f"{s} (n={n})"
        return "—" if v is None else str(v)

    lines = [f"# 비교 실험 {exp_id} — arm 별 지표", "", f"회차 {len(runs)}건 · 생성 {__import__('datetime').datetime.now().isoformat(timespec='minutes')}", "",
             "| arm | 회차 | 검증 통과 | 비평 거절(1회↑) | PR 도달 | 머지 | 검토 대기 | 회차 비용 | 머지당 비용 | 과제서 채택 | 수리 성공 | 중재(승인/전체) | 머지 뒤 3일 수정(잠정) | 머지 뒤 30일 사람 수정 |",
             "|---|---|---|---|---|---|---|---|---|---|---|---|---|"]
    for a in arms:
        m = result["arms"][a]
        lines.append(f"| {a} | {m['n']} | {cell(a, 'verify_pass')} | {cell(a, 'critic_rejected_ever')} | {cell(a, 'pr_reached')} | {cell(a, 'merged')} | {cell(a, 'review_pending')} | {cell(a, 'cost_per_round', 'mean')} | {'—' if m['cost_per_merge'] is None else '$' + str(m['cost_per_merge'])} | {cell(a, 'brief_accepted')} | {cell(a, 'repair_success')} | {m['arbiter']['approved']}/{m['arbiter']['n']} | {cell(a, 'postmerge_fix_3d')} | {cell(a, 'postmerge_human_fix_30d')} |")
    lines += ["", "## 클로드 회차만 (엔진 혼입 제거)", "",
              "같은 프롬프트라도 클로드가 한도에 걸린 날은 코덱스가 대신 돌았고, 그 회차는 절반이 변경 없이 끝난다. "
              "역할의 효과를 보려면 엔진을 섞지 말아야 한다 — 아래는 클로드가 구현한 회차만 다시 센 것이다.", "",
              "| arm | 회차 | 검증 통과 | PR 도달 | 머지 | 검토 대기 | 회차 비용 | 머지당 비용 |",
              "|---|---|---|---|---|---|---|---|"]
    for a in arms:
        m = result["arms_claude"][a]
        def c2(key, field="rate"):
            v = m.get(key) or {}
            x = v.get(field)
            if x is None: return "—"
            return (f"{round(x*100)}% (n={v.get('n')})" if field == "rate" else f"${x} (n={v.get('n')})")
        lines.append(f"| {a} | {m['n']} | {c2('verify_pass')} | {c2('pr_reached')} | {c2('merged')} | {c2('review_pending')} | {c2('cost_per_round','mean')} | {'—' if m['cost_per_merge'] is None else '$' + str(m['cost_per_merge'])} |")
    lines += ["", "RQ 매핑: " + " · ".join(f"{k}: {v}" for k, v in exp.get("questions", {}).items()), "",
              f"품질 축 성숙도: 머지된 실험 PR {sum(result['arms'][a]['pm_observed']['n'] for a in arms)}건, 관찰 최대 "
             f"{max([result['arms'][a]['pm_observed']['max_days'] for a in arms], default=0)}일 — 30일 열은 관찰 7일이 차야 값이 생긴다(빈 값은 측정 실패가 아니다).",
             "",
             "코덱스 대체 회차 비중(엔진 혼입): " + ", ".join(f"{a} {result['codex_share'][a] if result['codex_share'][a] is not None else '—'}" for a in arms)
             + " — 회차에 engine 이 없던 기간은 구현 단계 비용이 비어 있는 것(코덱스 합성 결과)으로 판정한다.",
             "",
             "비율의 대괄호는 부트스트랩 95% 신뢰구간. 캠페인 회차 비중: " + ", ".join(f"{a} {result['campaign_share'][a] if result['campaign_share'][a] is not None else '—'}" for a in arms) + "."]
    os.makedirs(OUTD, exist_ok=True)
    json.dump(result, open(os.path.join(OUTD, f"ab-{exp_id}.json"), "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    open(os.path.join(OUTD, f"ab-{exp_id}.md"), "w", encoding="utf-8").write("\n".join(lines) + "\n")
    print("\n".join(lines))


if __name__ == "__main__":
    main()
