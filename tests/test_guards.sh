#!/usr/bin/env bash
# bin/run.sh 의 보호 장치 중 모의 회차로는 재현하기 어려운 것을 함수 단위로 시험한다.
#
# 왜 따로 두나: tests/test_sim.py 는 회차 전체를 돌린다. 그런데 어떤 조건은 회차 **중간**에만
# 만들어진다 — 예를 들어 "릴리즈 커밋을 push 한 뒤 태그만 보류된 상태" 는 준비 단계에서
# 만들 수 없다. 그런 것을 모의 회차로 덮으려다 2026-09-28 에 가드를 꺼도 통과하는 빈
# 테스트를 쓸 뻔했다. 함수를 직접 부르는 편이 정직하다.
#
#   bash tests/test_guards.sh
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; REPO_DIR="$(cd "$HERE/.." && pwd)"
pass=0; fail=0
ok(){ printf '  ok   %s\n' "$1"; pass=$((pass+1)); }
no(){ printf '  FAIL %s — %s\n' "$1" "$2"; fail=$((fail+1)); }

# 시험할 함수만 run.sh 에서 꺼내 온다 (run.sh 는 실행 스크립트라 source 할 수 없다)
eval "$(sed -n '/^drop_phantom_tags(){/,/^}/p' "$REPO_DIR/bin/run.sh")"
eval "$(sed -n '/^unverified_langs(){/,/^}/p' "$REPO_DIR/bin/run.sh")"

fixture(){ # 원격 + 클론 + 커밋 하나. 출력: 클론 경로
  local t; t=$(mktemp -d)
  git init -q --bare "$t/origin.git"; git --git-dir="$t/origin.git" symbolic-ref HEAD refs/heads/main
  git init -q -b main "$t/work"
  ( cd "$t/work" && echo x > a.txt && git add -A \
    && git -c user.name=t -c user.email=t@t commit -qm init \
    && git remote add origin "$t/origin.git" && git push -qu origin main ) >/dev/null 2>&1
  printf '%s' "$t/work"
}

echo "drop_phantom_tags"
# 1) 원격에 있는 태그는 그대로 돌려준다
w=$(fixture)
git -C "$w" tag -a v1.0.0 -m x >/dev/null 2>&1; git -C "$w" push -q origin v1.0.0
git -C "$w" fetch -q origin main
[ "$(drop_phantom_tags "$w" main)" = v1.0.0 ] && ok "원격에 있는 태그는 유지" || no "원격에 있는 태그는 유지" "돌려준 값=$(drop_phantom_tags "$w" main)"

# 2) 원격에 없는 태그가 base 끝을 가리키면(=보류된 태그) 지우고 그 앞 태그를 돌려준다.
#    실제 상황을 그대로 만든다: 릴리즈 커밋은 push 됐고 태그만 로컬에 남았다.
( cd "$w" && echo y >> a.txt && git add -A \
  && git -c user.name=t -c user.email=t@t commit -qm release \
  && git push -q origin main ) >/dev/null 2>&1
git -C "$w" fetch -q origin main
git -C "$w" tag -a v2.0.0 -m x >/dev/null 2>&1        # push 하지 않는다 = 보류된 태그
got=$(drop_phantom_tags "$w" main)
[ "$got" = v1.0.0 ] && ok "원격에 없는 태그는 무시" || no "원격에 없는 태그는 무시" "돌려준 값=$got"
git -C "$w" tag -l v2.0.0 | grep -q . && no "원격에 없는 태그는 삭제" "v2.0.0 이 아직 남아 있다" || ok "원격에 없는 태그는 삭제"

# 3) 태그가 하나도 없으면 빈 값 (릴리즈를 막지 않는다)
w2=$(fixture); git -C "$w2" fetch -q origin main
[ -z "$(drop_phantom_tags "$w2" main)" ] && ok "태그가 없으면 빈 값" || no "태그가 없으면 빈 값" "돌려준 값=$(drop_phantom_tags "$w2" main)"

echo "unverified_langs"
w3=$(fixture); OUT=$(mktemp -d); base=$(git -C "$w3" rev-parse HEAD)
printf 'package main\n' > "$w3/x.go"; printf 'export const a=1\n' > "$w3/y.ts"
( cd "$w3" && git add -A && git -c user.name=t -c user.email=t@t commit -qm x ) >/dev/null 2>&1
printf '{"commands":[{"cmd":"go test ./...","exit":0}]}' > "$OUT/verify.json"
got=$(unverified_langs "$w3" "$base")
[ "$got" = ts ] && ok "go 만 검증했으면 ts 를 잡아낸다" || no "go 만 검증했으면 ts 를 잡아낸다" "돌려준 값=[$got]"
printf '{"commands":[{"cmd":"go test ./...","exit":0},{"cmd":"cd web && npm test","exit":0}]}' > "$OUT/verify.json"
got=$(unverified_langs "$w3" "$base")
[ -z "$got" ] && ok "둘 다 검증했으면 빈 값" || no "둘 다 검증했으면 빈 값" "돌려준 값=[$got]"

printf '\n통과 %d · 실패 %d\n' "$pass" "$fail"
[ "$fail" -eq 0 ]
