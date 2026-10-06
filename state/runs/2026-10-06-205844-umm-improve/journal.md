# 회차 노트 2026-10-06-205844-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:58] base pinned — main@bc051d4
- [러너 20:58] autonomy release — 

## 구현 노트
- 무엇을 왜: 지정된 실패(`Frontend dependency audit`)는 외부 권고 공개가 원인이고 이미 `4f2de72`/v0.76.7 로 수리·릴리즈됐다(지금 트리에서 audit EXIT=0). 그래서 워크플로를 느슨하게 하는 대신, 그 회차를 잃게 만든 재발 가능한 결함을 고쳤다 — `Makefile` 의 `test-web` 이 CI 프런트엔드 게이트 4개(`test:offline-queue`·`audit`·`build`·`verify:pwa`)를 돌지 않아 CI 를 떨어뜨린 게이트가 로컬에서 도달 불가였다. 1파일, 검사를 더하기만 함(임계값·범위·워크플로 불변).
- 확신 없는 곳·검증 못 한 것: ① **Go 통합 테스트는 SKIP 이다** — `POSTGRES_DSN` 없이 `make test-go` 를 돌렸다(Go 0줄 변경이라 띄우지 않았다). ② `make test-web` 이 이제 build+verify:pwa 까지 돌아 **눈에 띄게 느려졌다**(수 초 → 약 1분). 사람이 "test 타깃이 빌드까지 하는 것은 과하다" 고 볼 수 있는 설계 판단이며, 대안(별 타깃 `verify-web`)은 아무도 안 돌려 발견성을 못 고친다고 보고 고르지 않았다. ③ 재현에 쓴 GHSA 는 레지스트리에 살아 있는 실제 권고라 **재현이 네트워크에 의존**한다 — 오프라인에서는 audit 자체가 다르게 실패한다. ④ Playwright e2e 는 돌리지 않았다.
- 일부러 하지 않은 것: `npm ci` 를 `test-web` 에 넣지 않았다(설치는 `make web` 의 일이고 테스트 타깃이 node_modules 를 지우는 것은 되돌릴 수 없는 환경 변경 — 운영자 규칙 10번). `.github/workflows/*`·`Dockerfile`·`engines`·audit 임계값·`test-go` 의 `-p 1` 드리프트는 손대지 않았다(후자는 격리 DSN 으로 경합 증거를 먼저 모아야 하는 별 과제). ci.yml 과 Makefile 의 목록 일치를 기계로 강제하는 드리프트 체커는 YAML 재파싱이 필요해(운영자 규칙 3번의 '계약 다른 파서' 위험) 주석 한 줄로 대신했다 — 이것이 이 변경의 가장 약한 고리다.
- 다음 역할이 조심할 것: `make test-web` 은 **`npm ci --prefix web` 선행이 필요**하고(설치 없으면 첫 게이트부터 exit 1, 제품 결함 아님) **네트워크가 필요하다**(audit). 재현을 다시 보려면 `web/package-lock.json` 의 `source-map-js` 를 1.2.1 로 6줄 되돌린 뒤 `make test-web` → Error 1, 끝나면 `git checkout -- web/package-lock.json` 로 되돌릴 것(락파일 다운그레이드를 커밋하지 말 것 — 지금 작업 트리는 깨끗하다). `docker build` 는 통과 확인했고 검증 이미지 2개는 삭제했다.
- [러너 21:12] verify passed — 검증 17개 통과 (auto)

## 비평 노트
- 확인한 것: diff 는 `Makefile` 1파일·추가만. 새 8줄이 `ci.yml:112-137` Frontend 절과 순서까지 글자 그대로 일치함을 대조했고, 원장의 실패 재현을 독립 재현했다 — `web/package-lock.json` 의 source-map-js 를 1.2.1 로 되돌리면 `npm --prefix web audit --audit-level=high` 가 GHSA-68fv-2mgg-jv7q/high 로 EXIT=1, 되돌리니 EXIT=0(루트 락파일이 따로 있어도 `--prefix` 가 `web/` 것을 읽는다는 주장이 이 빨간 판정으로 증명됨). `make test-web` 전체 EXIT=0, 사후 `git status` 비어 있음(`web/dist` 는 .gitignore:5).
- 못 본 것: Playwright e2e, POSTGRES_DSN 통합 테스트(Go 0줄 변경이라 생략), 오프라인 상태에서의 audit 거동, `docker build`.
- 승인이어도 남는 우려: ① 새 주석의 "맨 앞 두 개는 판정이 커밋에 의존하지 않는다" 는 `test:offline-queue` 에 대해 거짓이다(`src/offline-queue.ts` 를 import 해 동작을 assert) — 동작은 맞고 설명만 과장됐으나 다음 회차가 이 전제로 순서를 재배치하면 안 된다. ② 구현 노트의 "약 1분" 은 과대평가 — 실측 10.9초이므로 릴리즈 노트에 그 숫자를 옮기지 말 것. ③ `make test` 가 이제 네트워크를 요구하고 audit 이 두 번째라 상류 권고 하나로 그날 자기 코드 판정을 못 받을 수 있다; `docs/` 에 `make test` 설명이 없어 적을 곳이 없었다.
- 보안·법무: 차단 없음. 인가·비밀값·암호·개인정보·라이선스 표면 변화 없고 공급망 게이트를 로컬에 되돌려 놓는 방향이다. revert 한 번으로 완전히 되돌아간다.
- [러너 21:15] review approved — 리뷰 승인 (risk=low)
- [러너 21:15] pr created — https://github.com/hkjang/umm/pull/168
- [러너 21:29] ci passed — 검사 1개 모두 success
- [러너 21:29] merge done — daf50ae
- [러너 21:48] release published — v0.76.8
- [러너 21:49] assets verified — v0.76.8 자산 3개 (이전 v0.76.7: 3)
