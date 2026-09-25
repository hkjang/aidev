#!/usr/bin/env python3
"""초기 형식 회차(2026-09-02~05)에 outcome 을 채운다 — 일회성 정규화, 재실행해도 안전.

러너 초기에는 runs.jsonl 이 {ts, date, project, result} 네 칸뿐이었고 `outcome` 이 없었다.
그 뒤에 들어온 분석은 전부 outcome 으로 거른다 — 대시보드 집계, 성과 쿨다운, 그리고
bin/postmerge.py 의 "러너가 머지한 PR" 목록까지. 그래서 **가장 오래된 249 회차가 통째로
빠져 있었다.** 머지 뒤 30일을 재는 논문 실험에서는 하필 관찰 기간이 가장 긴 표본이다.

결과 문장으로 outcome 을 되살린다. 문장이 애매하면 채우지 않는다.

    bin/normalize-legacy-runs.py            무엇을 채울지만 보여 준다
    bin/normalize-legacy-runs.py --apply    runs.jsonl 을 다시 쓴다 (러너가 쉴 때)
"""
import json, os, re, shutil, sys

HERE = os.path.dirname(os.path.abspath(__file__)); REPO_DIR = os.path.dirname(HERE)
RUNS = os.path.join(REPO_DIR, "docs", "data", "runs.jsonl")
APPLY = "--apply" in sys.argv


def outcome_of(result: str):
    r = result or ""
    if "merged http" in r or re.match(r"^(release-only|assets-only)", r):
        # 머지까지 갔으면 릴리즈 여부로 갈린다 — 이후 형식과 같은 뜻이 되게 맞춘다
        if "released v" in r or "release released" in r:
            return "release-ready"
        return "merged"
    if "no change" in r:
        return "no-change"
    if r.startswith("error") or "interrupted" in r:
        return "error"
    if "CI failed" in r or "verify" in r:
        return "verify-failed"
    if "PR open" in r or "PR " in r:
        return "review-pending"
    return None


def main():
    rows, filled, ambiguous = [], 0, 0
    for line in open(RUNS, encoding="utf-8"):
        line = line.strip()
        if not line:
            continue
        try:
            r = json.loads(line)
        except json.JSONDecodeError:
            rows.append(line); continue
        if r.get("outcome") is None and isinstance(r.get("project"), str):
            o = outcome_of(r.get("result", ""))
            if o:
                r["outcome"] = o
                r["normalized"] = "2026-09-25"
                # 결과 문장에 PR 이 있으면 pr 칸도 채운다 — postmerge 는 pr 로 잇는다
                if not r.get("pr"):
                    m = re.search(r"https://github\.com/[^ ,]*/pull/\d+", r.get("result", ""))
                    if m:
                        r["pr"] = m.group(0)
                filled += 1
            else:
                ambiguous += 1
        rows.append(json.dumps(r, ensure_ascii=False))
    print(f"outcome 을 채울 수 있는 회차 {filled}건 · 애매해서 건너뜀 {ambiguous}건")
    if not APPLY:
        print("--apply 를 주면 runs.jsonl 을 다시 쓴다"); return 0
    shutil.copy2(RUNS, RUNS + ".bak")
    tmp = RUNS + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        f.write("\n".join(rows) + "\n")
    os.replace(tmp, RUNS)
    print(f"runs.jsonl 갱신 ({filled}건). 백업: {os.path.basename(RUNS)}.bak")
    return 0


if __name__ == "__main__":
    sys.exit(main())
