#!/usr/bin/env python3
"""러너의 판정 함수 — 자동 진행을 막아야 하는 모든 경우를 성공과 구분한다.

원칙: 확인하지 못한 것은 성공이 아니다. API 오류, 파싱 오류, 빈 응답, 미완료, 시간 초과, 알 수 없는 값은 전부 '차단'이다.
순수 함수라 tests/test_gate.py 로 회귀 테스트한다. 러너(run.sh)는 서브커맨드로 부른다:

  gate.py ci       <check-runs.json> --sha <커밋> [--required A,B] [--allow-no-ci]      → merge_ok
  gate.py review   <review.json>                                                          → approve 만 통과
  gate.py release  <release.json> --out-dir <허용 디렉터리>                                → 스키마·자산 경로·존재 확인
  gate.py ideas    <ideas.json>                                                            → 스키마 확인
  gate.py secrets  <파일 또는 - (stdin)>                                                   → 비밀정보 패턴이 있으면 차단
  gate.py verify   <verify.json>                                                           → 러너 검증 결과 종합
종료 코드 0 = 진행 허용, 1 = 차단. stdout 에 JSON {"ok":bool,"state":str,"reason":str,...} 를 쓴다.
"""
import argparse
import json
import os
import re
import sys

ALLOWED_VERDICTS = {"approve", "reject"}
ALLOWED_RELEASE_STATUS = {"released", "skipped", "failed"}
ALLOWED_IDEA_STATUS = {"pending", "done", "rejected"}
ALLOWED_SIZES = {"S", "M", "L"}
GOOD_CONCLUSIONS = {"success", "neutral", "skipped"}
SECRET_PATTERNS = [  # 확실한 토큰 형식 — 문맥 없이도 차단
    (r"gh[pousr]_[A-Za-z0-9]{20,}", "GitHub 토큰"),
    (r"github_pat_[A-Za-z0-9_]{20,}", "GitHub fine-grained 토큰"),
    (r"AKIA[0-9A-Z]{16}", "AWS 액세스 키"),
    (r"xox[abprs]-[A-Za-z0-9-]{10,}", "Slack 토큰"),
    (r"sk-(proj-|ant-)?[A-Za-z0-9_-]{24,}", "API 키(sk-)"),
    (r"-----BEGIN (RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----", "개인 키"),
    (r"eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{40,}\.[A-Za-z0-9_-]{20,}", "JWT"),
]
# 값을 봐야 하는 패턴 — 자리표시자·예시·로컬 DSN 은 걸지 않는다 (docs/compose/.env.example 의 postgres://app:app@db 같은 것)
VALUE_PATTERNS = [
    (r"(?i)(password|passwd|secret|token|api[_-]?key)\s*[:=]\s*['\"]([^'\"\s]{8,})['\"]", "비밀값 대입"),
    (r"[a-z][a-z0-9+.-]*://([^/\s:@]+):([^/\s:@]+)@([^/\s:]+)", "URL 내 자격증명"),
]
PLACEHOLDER = re.compile(r"(?i)^(.*(password|passwd|secret|changeme|change[_-]?me|example|sample|dummy|placeholder|your[_-]|xxx|todo|redacted|test|dev|local|admin|user|guest|root|\*{3,}|\.{3,}).*|<[^>]*>|\$\{?[A-Za-z_][A-Za-z0-9_]*\}?|\{\{.*\}\}|%[A-Za-z_]+%|[a-z]{1,8})$")
LOCAL_HOST = re.compile(r"(?i)^(localhost|127\.\d+\.\d+\.\d+|0\.0\.0\.0|host\.docker\.internal|[a-z0-9_-]+)(:\d+)?$")  # 점 없는 호스트 = compose 서비스명
INTERNAL_PATTERNS = [
    (r"\b(10\.\d{1,3}|192\.168|172\.(1[6-9]|2\d|3[01]))\.\d{1,3}\.\d{1,3}\b", "사설 IP"),
    (r"\b[a-z0-9-]+\.(internal|corp|lan)\b", "내부 호스트명"),
]
SKIP_FILES = re.compile(r"(^|/)(docs?/|test/|tests/|testdata/|fixtures?/|examples?/|CHANGELOG|README|.*\.(md|example|sample|template|dist)$|docker-compose.*\.ya?ml$|compose.*\.ya?ml$|.*_test\.go$|.*\.test\.[jt]sx?$|.*\.spec\.[jt]sx?$|test_.*\.py$|.*\.env\.example$)")


def looks_real(value):
    """자리표시자가 아니고 실제 비밀처럼 보이는 값인가: 12자 이상, 글자와 숫자/기호 섞임."""
    if PLACEHOLDER.match(value):
        return False
    if len(value) < 12:
        return False
    return bool(re.search(r"[A-Za-z]", value)) and bool(re.search(r"[0-9!@#$%^&*+=/_-]", value))


def out(ok, state, reason, **extra):
    print(json.dumps({"ok": ok, "state": state, "reason": reason, **extra}, ensure_ascii=False))
    return 0 if ok else 1


def load_json(path):
    """(data, error). 파일 없음·빈 파일·파싱 오류를 구분해 돌려준다."""
    if path == "-":
        raw = sys.stdin.read()
    else:
        if not os.path.exists(path):
            return None, "missing"
        raw = open(path, encoding="utf-8").read()
    if not raw.strip():
        return None, "empty"
    try:
        return json.loads(raw), None
    except json.JSONDecodeError as e:
        return None, f"invalid-json: {e.msg} at {e.pos}"


# ---------- CI ----------
def evaluate_ci(data, sha, required=None, allow_no_ci=False):
    """check-runs API 응답(dict 또는 --paginate 로 이어붙인 list)을 판정한다.
    반환 (ok, state, reason, details). ok 는 '모든 필수 검사가 이 커밋에서 success 로 완료' 일 때만 True."""
    runs = []
    pages = data if isinstance(data, list) else [data]
    for page in pages:
        if not isinstance(page, dict):
            return False, "api-error", "응답이 객체가 아님", {}
        if "message" in page and "check_runs" not in page:
            return False, "api-error", f"API 오류: {page.get('message')}", {}
        if "check_runs" not in page:
            return False, "parse-error", "check_runs 필드 없음", {}
        if not isinstance(page["check_runs"], list):
            return False, "parse-error", "check_runs 가 배열이 아님", {}
        runs.extend(page["check_runs"])
    # 커밋 일치 — 다른 커밋의 검사로 판정하지 않는다
    wrong = [r.get("name") for r in runs if sha and r.get("head_sha") and r["head_sha"] != sha]
    if wrong:
        return False, "sha-mismatch", f"다른 커밋의 검사 포함: {', '.join(map(str, wrong[:3]))}", {}
    # 같은 이름의 검사가 여러 번이면(재실행) 가장 최근 것만 본다
    latest = {}
    for r in runs:
        name = r.get("name") or f"#{r.get('id')}"
        key = (r.get("started_at") or "", r.get("id") or 0)
        if name not in latest or key > latest[name][0]:
            latest[name] = (key, r)
    runs = [v[1] for v in latest.values()]
    if not runs:
        if allow_no_ci:
            return True, "no-ci-allowed", "검사 없음 — 정책으로 허용", {"checks": 0}
        return False, "no-ci", "이 커밋에 검사가 없음 (정책 allow_merge_without_ci 가 없으면 차단)", {"checks": 0}
    required = [x for x in (required or []) if x]
    names = {r.get("name") for r in runs}
    missing = [n for n in required if n not in names]
    if missing:
        return False, "required-missing", f"필수 검사가 없음: {', '.join(missing)}", {"checks": len(runs)}
    subject = [r for r in runs if (not required) or r.get("name") in required]
    pending = [r.get("name") for r in subject if r.get("status") != "completed"]
    if pending:
        return False, "pending", f"완료되지 않은 검사: {', '.join(map(str, pending))}", {"checks": len(runs), "pending": len(pending)}
    bad = [(r.get("name"), r.get("conclusion")) for r in subject if r.get("conclusion") not in GOOD_CONCLUSIONS]
    if bad:
        kinds = {c for _, c in bad}
        state = "failed" if "failure" in kinds else ("timed_out" if "timed_out" in kinds else ("cancelled" if "cancelled" in kinds else "not-success"))
        return False, state, "성공이 아닌 검사: " + ", ".join(f"{n}={c}" for n, c in bad), {"checks": len(runs), "bad": len(bad)}
    return True, "success", f"검사 {len(subject)}개 모두 success", {"checks": len(runs)}


def cmd_ci(a):
    data, err = load_json(a.path)
    if err:
        return out(False, "api-error" if err == "missing" else err.split(":")[0], f"check-runs 를 읽지 못함: {err}")
    required = [x.strip() for x in (a.required or "").split(",") if x.strip()]
    ok, state, reason, details = evaluate_ci(data, a.sha, required, a.allow_no_ci)
    return out(ok, state, reason, **details)


# ---------- 리뷰 ----------
def evaluate_review(data):
    if not isinstance(data, dict):
        return False, "invalid", "리뷰 결과가 객체가 아님"
    v = data.get("verdict")
    if v not in ALLOWED_VERDICTS:
        return False, "unknown-verdict", f"허용되지 않은 판정값: {v!r}"
    if v == "reject":
        reasons = data.get("reasons") or []
        return False, "rejected", "리뷰 거절: " + (" / ".join(map(str, reasons[:3])) or "(사유 없음)")
    return True, "approved", f"리뷰 승인 (risk={data.get('risk', '?')})"


def cmd_review(a):
    data, err = load_json(a.path)
    if err:
        return out(False, "missing" if err == "missing" else "invalid", f"리뷰 결과 없음/손상: {err} — 보류")
    ok, state, reason = evaluate_review(data)
    return out(ok, state, reason, risk=data.get("risk") if isinstance(data, dict) else None)


# ---------- 릴리즈 결과 ----------
def evaluate_release(data, out_dir):
    if not isinstance(data, dict):
        return False, "invalid", "릴리즈 결과가 객체가 아님", {}
    st = data.get("status")
    if st not in ALLOWED_RELEASE_STATUS:
        return False, "unknown-status", f"허용되지 않은 status: {st!r}", {}
    if st != "released":
        return False, st, f"릴리즈 안 함: {data.get('reason') or st}", {}
    tag = data.get("tag") or ""
    if tag and not re.match(r"^[A-Za-z0-9][A-Za-z0-9._\-]{0,99}$", tag):
        return False, "bad-tag", f"태그 형식 이상: {tag!r}", {}
    if not isinstance(data.get("github_release", False), bool):
        return False, "invalid", "github_release 는 true/false 여야 함", {}
    assets = data.get("assets", [])
    if not isinstance(assets, list) or not all(isinstance(x, str) for x in assets):
        return False, "invalid", "assets 는 문자열 배열이어야 함", {}
    root = os.path.realpath(out_dir) if out_dir else None
    good, problems = [], []
    for p in assets:
        rp = os.path.realpath(p)
        if root and not (rp == root or rp.startswith(root + os.sep)):
            problems.append(f"허용 디렉터리 밖: {p}")
        elif not os.path.isfile(rp):
            problems.append(f"파일 없음: {p}")
        elif os.path.getsize(rp) == 0:
            problems.append(f"빈 파일: {p}")
        else:
            good.append(rp)
    notes = data.get("notes_file") or ""
    if notes:
        rn = os.path.realpath(notes)
        if root and not rn.startswith(root + os.sep):
            problems.append(f"notes_file 이 허용 디렉터리 밖: {notes}")
        elif not os.path.isfile(rn):
            problems.append(f"notes_file 없음: {notes}")
    if problems:
        return False, "bad-assets", "; ".join(problems), {"assets_ok": good}
    return True, "released", f"tag={tag or '(없음)'} assets={len(good)}", {"assets_ok": good, "tag": tag}


def cmd_release(a):
    data, err = load_json(a.path)
    if err:
        return out(False, "missing" if err == "missing" else "invalid", f"릴리즈 결과 없음/손상: {err}")
    ok, state, reason, details = evaluate_release(data, a.out_dir)
    return out(ok, state, reason, **details)


# ---------- 아이디어 ----------
def evaluate_ideas(data):
    if not isinstance(data, list):
        return False, "invalid", "배열이 아님"
    for i, it in enumerate(data):
        if not isinstance(it, dict) or not it.get("title"):
            return False, "invalid", f"[{i}] title 없음"
        if it.get("status") not in ALLOWED_IDEA_STATUS:
            return False, "invalid", f"[{i}] status 이상: {it.get('status')!r}"
        if it.get("size") not in ALLOWED_SIZES:
            return False, "invalid", f"[{i}] size 이상: {it.get('size')!r}"
        for k in ("value", "risk"):
            if not isinstance(it.get(k), int) or not 1 <= it[k] <= 5:
                return False, "invalid", f"[{i}] {k} 는 1~5 정수"
    return True, "valid", f"{len(data)}건"


def cmd_ideas(a):
    data, err = load_json(a.path)
    if err:
        return out(False, "missing" if err == "missing" else "invalid", f"아이디어 파일 없음/손상: {err}")
    ok, state, reason = evaluate_ideas(data)
    return out(ok, state, reason)


# ---------- 비밀정보 ----------
def _hits_in(text, internal):
    hits = []
    for pat, label in SECRET_PATTERNS + (INTERNAL_PATTERNS if internal else []):
        for m in re.finditer(pat, text):
            s = m.group(0)
            hits.append(f"{label}: {s[:6]}…{s[-3:]}" if len(s) > 12 else f"{label}: {s[:4]}…")
    for pat, label in VALUE_PATTERNS:
        for m in re.finditer(pat, text):
            if label == "URL 내 자격증명":
                user, pw, host = m.group(1), m.group(2), m.group(3)
                if LOCAL_HOST.match(host) or not looks_real(pw) or pw == user:
                    continue
            else:
                if not looks_real(m.group(2)):
                    continue
            s = m.group(0)
            hits.append(f"{label}: {s[:6]}…{s[-3:]}")
    return hits


def scan_secrets(text, internal=False):
    """unified diff 면 추가된 줄만, 문서·예시·테스트·compose 파일은 건너뛴다. 그 외 텍스트는 통째로 본다."""
    if "+++ b/" in text or "+++ " in text and "\n@@" in text:
        hits, current, skip = [], "", False
        for line in text.splitlines():
            if line.startswith("+++ "):
                current = line[4:].strip()
                current = current[2:] if current.startswith("b/") else current
                skip = bool(SKIP_FILES.search(current))
                continue
            if skip or not line.startswith("+") or line.startswith("+++"):
                continue
            for h in _hits_in(line[1:], internal):
                hits.append(f"{current}: {h}")
                if len(hits) >= 10:
                    return hits
        return hits
    return _hits_in(text, internal)[:10]


def cmd_secrets(a):
    text = sys.stdin.read() if a.path == "-" else (open(a.path, encoding="utf-8", errors="replace").read() if os.path.exists(a.path) else "")
    hits = scan_secrets(text, a.internal)
    if hits:
        return out(False, "secrets-found", "비밀정보/내부 정보 의심: " + "; ".join(hits), hits=len(hits))
    return out(True, "clean", "비밀정보 없음")


# ---------- 러너 검증 결과 ----------
def evaluate_verify(data):
    """verify.json: {"commands":[{"cmd":..,"exit":int,"seconds":..}], "source":"policy|auto|none"}"""
    if not isinstance(data, dict):
        return False, "invalid", "검증 결과가 객체가 아님"
    cmds = data.get("commands")
    if data.get("source") == "none" or not cmds:
        return False, "no-verify", "실행한 검증 명령이 없음 — 정책(verify) 또는 자동 감지 필요"
    bad = [c for c in cmds if not isinstance(c.get("exit"), int) or c["exit"] != 0]
    if bad:
        return False, "verify-failed", "실패한 검증: " + "; ".join(f"{c.get('cmd')} (exit {c.get('exit')})" for c in bad[:3])
    return True, "verified", f"검증 {len(cmds)}개 통과 ({data.get('source')})"


def cmd_verify(a):
    data, err = load_json(a.path)
    if err:
        return out(False, "missing" if err == "missing" else "invalid", f"검증 결과 없음/손상: {err}")
    ok, state, reason = evaluate_verify(data)
    return out(ok, state, reason)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("ci"); p.add_argument("path"); p.add_argument("--sha", default=""); p.add_argument("--required", default=""); p.add_argument("--allow-no-ci", action="store_true"); p.set_defaults(fn=cmd_ci)
    p = sub.add_parser("review"); p.add_argument("path"); p.set_defaults(fn=cmd_review)
    p = sub.add_parser("release"); p.add_argument("path"); p.add_argument("--out-dir", default=""); p.set_defaults(fn=cmd_release)
    p = sub.add_parser("ideas"); p.add_argument("path"); p.set_defaults(fn=cmd_ideas)
    p = sub.add_parser("secrets"); p.add_argument("path"); p.add_argument("--internal", action="store_true"); p.set_defaults(fn=cmd_secrets)
    p = sub.add_parser("verify"); p.add_argument("path"); p.set_defaults(fn=cmd_verify)
    a = ap.parse_args(argv)
    return a.fn(a)


if __name__ == "__main__":
    sys.exit(main())
