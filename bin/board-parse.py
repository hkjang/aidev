#!/usr/bin/env python3
"""이사회 메모(stdin)에서 제안 목록을 뽑는다 — 마지막 줄의 JSON 이 있으면 그것을, 없으면 '## 결정 제안' 절의
"N. `동사 인자...` — 왜. 포기: ... 되돌림: `...`" 줄에서. 결과는 JSON 한 줄(stdout). 동사는 ops.sh 의 허용 목록만."""
import json, re, shlex, sys

ALLOWED = {"campaign", "shepherd", "stop", "resume", "run", "draft-campaign", "activate-campaign", "discard-draft"}
t = sys.stdin.read()
m = None
for start in [i for i, ch in enumerate(t) if ch == "{"]:
    for end in range(len(t), start, -1):
        if t[end - 1] != "}":
            continue
        try:
            d = json.loads(t[start:end])
        except Exception:
            continue
        if isinstance(d, dict) and isinstance(d.get("proposals"), list):
            m = d
            break
    if m:
        break

props, constraint, not_doing, advice = [], "", [], []
if m:
    for p in m["proposals"]:
        cmd = [str(x) for x in (p.get("cmd") or [])]
        if cmd and cmd[0] in ALLOWED:
            props.append({"cmd": cmd, "why": str(p.get("why", ""))[:400], "gives_up": str(p.get("gives_up", ""))[:200], "revert": str(p.get("revert", ""))[:200]})
    constraint = str(m.get("constraint", ""))[:300]
    not_doing = [str(x)[:200] for x in (m.get("not_doing") or [])][:8]
    advice = [str(x)[:300] for x in (m.get("role_advice") or [])][:8]
else:
    sec = ""
    for line in t.splitlines():
        h = line.strip()
        if h.startswith("## "):
            sec = h[3:].strip()
            continue
        if sec.startswith("이번 주의 제약") and h and not constraint:
            constraint = h.replace("**", "").strip()[:300]
        if sec.startswith("하지 않는 것") and h.startswith("- "):
            not_doing.append(h[2:][:200])
        if sec.startswith("역할 권고") and h.startswith("- "):
            advice.append(h[2:][:300])
        mm = re.match(r"^\s*(\d+)\.\s+`([^`]+)`\s*(?:[—–-]\s*)?(.*)$", line)
        if mm and sec.startswith("결정 제안"):
            try:
                cmd = shlex.split(mm.group(2))
            except ValueError:
                cmd = mm.group(2).split()
            if cmd and cmd[0] in ALLOWED:
                rest = mm.group(3)
                gives = re.search(r"포기[:：]\s*([^.。]*)", rest)
                rev = re.search(r"되돌림[:：]\s*`?([^`\n]*)`?", rest)
                props.append({"cmd": cmd, "why": rest[:400], "gives_up": (gives.group(1) if gives else "")[:200], "revert": (rev.group(1) if rev else "")[:200]})
for i, p in enumerate(props, 1):
    p["n"] = i
    p["applied"] = None
print(json.dumps({"constraint": constraint, "proposals": props, "not_doing": not_doing[:8], "role_advice": advice[:8]}, ensure_ascii=False))
