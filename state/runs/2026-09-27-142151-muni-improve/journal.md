# 회차 노트 2026-09-27-142151-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:21] base pinned — main@fdbf260
- [러너 14:21] autonomy release — 

## 구현 노트
- 왜: 워크스페이스 ZIP 의 안내 파일 `목록.md` 만 항목 이름 중복 추적(`uniqueEntryName`)을 거치지 않아, 루트에 제목이 `목록` 인 문서가 있고 md 로 내보내면 ZIP 에 같은 이름 항목이 둘 들어가고 순서대로 쓰는 해제 도구에서 한쪽이 사라졌다. 이름을 상수로 두고 순회 전에 `used` 에 예약했다(프로덕션 파일 1, 테스트 파일 1, 커밋 a8f56b6).
- 확신 없는 곳: 실제로 "어느 쪽이 남는지" 는 압축 해제 도구에 달려 있고 확인하지 않았다 — 테스트가 주장하는 것은 "같은 이름 항목이 둘 존재한다" 까지이고, 그 뒤 손실은 추론이다. `archive/zip` 은 중복 이름을 거부하지 않는다는 것만 실행으로 봤다.
- 일부러 하지 않은 것: `uniqueEntryName` 시그니처와 `safeFilename`·`목록.md` 문자열 자체는 건드리지 않았다. 다른 라우트(문서 내려받기·발표자료·첨부)에는 고정 이름 항목이 없어 같은 결함이 없다. 한도 초과 안내(2000건 경계) 오탐은 별개 과제로 남겼다.
- 다음 역할이 조심할 것: 새 테스트 `TestADocumentTitledLikeTheIndexKeepsItsOwnEntry` 는 live 라 `MUNI_TEST_DSN` 없으면 SKIP 된다(단위만 돌려 "통과" 로 적지 말 것). 나는 postgres:16-alpine 컨테이너로 돌렸고 httpapi PASS 235 / SKIP 0 / FAIL 0, `go test ./...` ok, `go vet`·`gofmt -l .`·placeholder 스크립트 모두 통과. 프런트 미변경이라 npm 검사는 돌리지 않았다.
- [러너 14:27] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 구현자가 "확신 없는 곳" 으로 남긴 인과를 직접 재현했습니다 — 전용 postgres:16-alpine 로 새 live 테스트를 통과시키고, workspace_export.go:141 예약 한 줄만 되돌리자 원장의 `실패 재현` 과 바이트 단위로 같은 출력(`:114` 중복 `목록.md`, `:133` 루트 문서 항목 없음)으로 실패했습니다. `go test ./internal/httpapi` 전체·`./...`·vet·gofmt 통과. 판정 approve / risk low / 차단 없음.
- 주석 주장도 코드로 확인: `used` 키가 전체 경로라 폴더 안 `목록` 은 무관, 확장자가 항상 `format` 이라 html·txt 는 이 이름에 닿지 않음, 제목 `목록 (2)` 가 함께 있어도 suffix 루프가 `목록 (3).md` 로 비킵니다.
- 못 본 것: 실제 압축 해제 도구별로 어느 쪽이 남는지(구현자와 동일하게 미확인 — 다만 중복 항목 자체가 결함이라 판정에 영향 없음), 프런트(미변경)와 Playwright.
- 릴리즈가 알아야 할 것 둘: (1) v0.47.0 노트의 "ZIP 항목 이름 한 바이트도 안 바뀜" 약속과 달리 md 에서 루트 `목록` 문서 이름이 `목록 (2).md` 로 바뀌므로 노트에 명시할 것. (2) 이 워크트리의 `webui/dist/index.html` 이 verify 의 `npm run build` 로 더러워져 placeholder 스크립트가 exit 1 — 커밋 밖이지만 `git add -A` 금지, 먼저 `git checkout --` 할 것.
- 다음 회차 후보(기존 공백): 폴더 이름이 `목록.md` 면 디렉터리 항목 `목록.md/…` 과 파일 항목 `목록.md` 가 한 아카이브에 공존합니다.
- [러너 14:32] review approved — 리뷰 승인 (risk=low)
- [러너 14:32] pr created — https://github.com/hkjang/muni/pull/28
- [러너 14:38] ci passed — 검사 2개 모두 success
- [러너 14:38] merge done — a8f56b6
- [러너 14:46] release published — v0.48.0
- [러너 14:51] assets verified — v0.48.0 자산 1개 (이전 v0.47.0: 1)
