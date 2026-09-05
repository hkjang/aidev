#!/usr/bin/env bash
# 전체 흐름 모의 실행 — 가짜 gh·claude·원격 저장소로 러너(bin/run.sh)를 실제로 돌린다.
#   사용: tests/sim/run_sim.sh <scenario.json> [--keep]   → 결과 요약 JSON 을 stdout 에 쓴다 (outcome, stages, prs, releases, gh 호출)
# 실제 상태·로그·데이터·잠금 파일은 건드리지 않는다 (AIDEV_STATE/LOGS/DATA/OWNER/WT_BASE 를 임시 경로로).
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"; REPO_DIR="$(cd "$HERE/../.." && pwd)"
SCN=${1:?scenario.json}; KEEP=${2:-}
T=$(mktemp -d /tmp/aidev-sim.XXXXXX)
export AIDEV_SIM="$T/sim" AIDEV_STATE="$T/state" AIDEV_LOGS="$T/logs" AIDEV_DATA="$T/data" AIDEV_OWNER="$T/run.owner" WT_BASE="$T/wt" ROOT="$T/projects" HOME_REAL="$HOME"
mkdir -p "$AIDEV_SIM" "$AIDEV_STATE" "$AIDEV_LOGS" "$AIDEV_DATA" "$WT_BASE" "$ROOT"
cp "$REPO_DIR/state/default.policy.json" "$REPO_DIR/state/default.guard" "$AIDEV_STATE/"
[ -f "$REPO_DIR/state/caps.env" ] && cp "$REPO_DIR/state/caps.env" "$AIDEV_STATE/"
# 모의 프로젝트: bare 원격 + 작업 클론
proj=simproj; origin="$T/origin.git"; git init -q --bare "$origin"
git init -q -b main "$ROOT/$proj"; ( cd "$ROOT/$proj" && echo "# sim" > README.md && printf 'v0.0.1\n' > VERSION && git add -A && git -c user.name=sim -c user.email=sim@x commit -qm "init" && git remote add origin "$origin" && git push -q -u origin main )
git -C "$ROOT/$proj" tag -a v0.0.1 -m v0.0.1; git -C "$ROOT/$proj" push -q origin v0.0.1
# 시나리오 → 모의 도구가 읽을 위치 (gh 는 AIDEV_SIM, claude 는 STATE)
jq --arg o "$origin" --arg p "$ROOT/$proj" '. + {origin:$o, project_repo:$p}' "$SCN" > "$AIDEV_SIM/scenario.json"; cp "$AIDEV_SIM/scenario.json" "$AIDEV_STATE/scenario.json"
# 프로젝트 정책: 러너 검증은 시나리오가 정한 명령(기본 true)
jq -n --argjson v "$(jq -c '.verify // ["true"]' "$SCN")" --argjson nc "$(jq -c '.allow_merge_without_ci // false' "$SCN")" --arg au "$(jq -r '.autonomy // "release"' "$SCN")" \
   '{verify:$v, allow_merge_without_ci:$nc, autonomy:$au}' > "$AIDEV_STATE/$proj.policy.json"
# 이전 릴리즈(자산 포함)를 원격에 있는 것처럼
for a in $(jq -r '.prev_release_assets[]? // empty' "$SCN"); do mkdir -p "$AIDEV_SIM/releases/v0.0.1/assets"; head -c 2048 /dev/urandom > "$AIDEV_SIM/releases/v0.0.1/assets/$a"; done
[ -d "$AIDEV_SIM/releases/v0.0.1" ] && jq -n '{tagName:"v0.0.1",name:"v0.0.1",publishedAt:"2026-01-01T00:00:00+09:00",isPrerelease:false,body:""}' > "$AIDEV_SIM/releases/v0.0.1/meta.json"
[ "$(jq -r '.tag_exists // false' "$SCN")" = true ] && { git -C "$ROOT/$proj" tag -a "$(jq -r '.release.tag // "v0.0.2"' "$SCN")" -m x; git -C "$ROOT/$proj" push -q origin "$(jq -r '.release.tag // "v0.0.2"' "$SCN")"; }
for st in $(jq -r '.stops[]? // empty' "$SCN"); do touch "$AIDEV_STATE/STOP$( [ "$st" = all ] && echo "" || echo "-$st")"; done
[ "$(jq -r '.duplicate_pr // false' "$SCN")" = true ] && export AIDEV_SIM_DUP=1
touch "$AIDEV_DATA/runs.jsonl" "$AIDEV_DATA/usage.jsonl"
export PATH="$HERE/bin:$PATH"
extra=$(jq -r '.args // ""' "$SCN")
# 승인 스윕·수동 큐 등은 --project 로 건너뛴다; 시간 대기는 짧게
export T_IMPROVE=120 T_REVIEW=60 T_RELEASE=120 T_ASSETS=120 CI_POLL=1 CI_MAX=4 REL_MAX=2 RETRY_DELAYS="1 1 1"
( cd "$REPO_DIR" && bash bin/run.sh --project "$proj" --no-sync $extra ) > "$T/runner.out" 2>&1
rc=$?
{ [ -s "$AIDEV_DATA/runs.jsonl" ] && tail -1 "$AIDEV_DATA/runs.jsonl" || echo '{}'; } | jq -c --arg t "$T" --argjson rc "$rc" \
  --argjson prs "$(for f in "$AIDEV_SIM"/prs/*.json; do [ -f "$f" ] && jq -c '{url,state,head}' "$f"; done | jq -s '.')" \
  --argjson rels "$(for f in "$AIDEV_SIM"/releases/*/meta.json; do [ -f "$f" ] && jq -c --arg d "$(dirname "$f")/assets" '{tag:.tagName, assets:[]}' "$f" | jq -c --arg d "$(dirname "$f")/assets" '.assets=[($d|.)]' ; done | jq -s '.')" \
  --arg tags "$(git --git-dir="$origin" tag | tr '\n' ' ')" --arg gh "$(grep -c . "$AIDEV_SIM/log/gh.log" 2>/dev/null || echo 0)" \
  '{rc:$rc, outcome, result, stages:((.stages // {})|map_values(.state)), prs:$prs, releases:$rels, remote_tags:$tags, gh_calls:($gh|tonumber), tmp:$t}'
for r in "$AIDEV_SIM"/releases/*/assets; do [ -d "$r" ] && printf '{"release_assets":"%s","names":"%s"}\n' "$(basename "$(dirname "$r")")" "$(ls "$r" | tr '\n' ' ')"; done
[ -n "$KEEP" ] || rm -rf "$T"
exit 0
