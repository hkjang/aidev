# 회차 노트 2026-09-23-230446-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:04] base pinned — main@96f7414
- [러너 23:04] autonomy release — 
- [러너 23:10] scout done — 저장한 목록 보기·목록 주소 복사의 검색어/필터 500자 잘림을 코드포인트 단위로 고쳐 문자가 깨지지 않�

## 구현 노트
- 무엇/왜: `web/src/saved-list-views.ts`의 `savedListQuery`가 `q`·`f_<key>`를 `slice(0,500)`으로 잘라 서로게이트 쌍을 쪼개면 외톨이 서로게이트가 URL 직렬화에서 U+FFFD가 되어 저장한 보기·복사한 주소가 다른 문자열로 검색됐다. 로컬 헬퍼 `clip`으로 한도(코드유닛 500)는 그대로 두고 쌍이 갈릴 때만 한 칸 줄여 자른다. 커밋 9832f98.
- 전제 실증: 정찰이 "미확인"으로 남긴 U+FFFD 치환을 `node -e` 왕복으로 먼저 확인했다(`slice(0,500)` 끝이 0xD83D → `%EF%BF%BD`, `startsWith` false). 과제 성립.
- 확신 없는 곳/미검증: (1) 실제 브라우저에서 저장→복원 하는 end-to-end는 돌리지 않았다(Node `URLSearchParams` 왕복으로만 증명). (2) 같은 값을 읽는 다른 경로 — `list-export.ts:listSharePath`도 `savedListQuery`를 거치므로 같이 고쳐지지만, 주소 복사에 500자 한도를 적용할지의 정책 자체는 과제서 지시대로 건드리지 않았다(차선 후보로 남김). (3) 입력에 이미 외톨이 서로게이트가 있는 경우는 실제로는 `URLSearchParams` 디코딩이 먼저 U+FFFD로 바꿔 도달 불가하다고 판단해 테스트하지 않았다 — 판단 근거는 추론이다.
- 일부러 안 한 것: Go 스위트·`internal/webassets/dist` 재복사(Go 무변경, 과제서 지시), 문서·openapi.json(사용자 문구·API 계약 변화 없음), 한도 500 자체 변경(`readListPreferences`의 8192 저장 한도와 짝).
- 다음 역할 주의: `web/node_modules`가 없어 `npm --prefix web ci`를 먼저 돌려야 `tsc`가 있다. `tsc --noEmit`은 cwd에 tsconfig가 없으면 help만 찍으니 `-p web/tsconfig.json`으로 부를 것. 테스트는 DB 불필요.
- 검증: 기준선 92통과/0실패/0skip → 93통과/0실패/0skip, 되돌림 시 1실패로 인과 확인, `tsc -p web/tsconfig.json --noEmit` 통과, `npm --prefix web run build` 성공, `git diff --check` 깨끗.
- [러너 23:14] brief accepted — 채택 — 과제서가 지목한 `slice(0, 500)` 두 군데와 U+FFFD 치환 전제가 현재 코드·런타임에서 그대로 재현되어 제안대로 구�
- [러너 23:14] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 판정 approve(low, blocking 없음). clip() 경계 3종(쌍이 499/500에 걸침·498/499에 정확히 맞음·구코드 재현)을 node로 직접 돌려 off-by-one 없음을 확인했고, 새 테스트가 수정 전 코드에서 실제로 깨지는 인과 테스트임을 독립 재현했다. npm --prefix web test 93통과/0실패/0skip 직접 실행.
- 구현자가 의심한 자리 3건 모두 확인: 외톨이 서로게이트 입력은 여전히 U+FFFD를 남기지만 URLSearchParams 디코딩 때문에 도달 경로를 찾지 못했다(차기 후보, 차단 아님). dist 재복사 생략은 .gitignore:12 + CI Embed 단계로 정당.
- 못 본 것: 실브라우저 저장→복원 end-to-end, Go 스위트·DB(변경 없음이라 생략).
- 남는 우려(릴리즈 노트): list-tools.tsx:144-150이 저장 시 500자 초과를 거부하므로 이번 수정의 실효 경로는 주로 목록 주소 복사(list-export.ts:75)다. 커밋 제목보다 범위가 좁으니 노트는 그 기준으로 쓸 것. VERSION 1.12.0 미변경 — 릴리즈 단계에서 버전 동기화 판단 필요.
- [러너 23:17] review approved — 리뷰 승인 (risk=low)
- [러너 23:17] pr created — https://github.com/hkjang/hunter/pull/9
- [러너 23:36] ci timeout — 제한 시간 안에 CI 완료를 확인하지 못함
