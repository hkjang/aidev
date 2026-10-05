# 회차 노트 2026-10-05-171744-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:17] base pinned — main@2a93c60
- [러너 17:17] autonomy release — 

## 정찰 노트
- 폴더 디렉터리 병합을 골랐다: 직전 회차(a55bf5f)가 항목 이름만 대소문자 무시로 유일하게 만들어 `Report/`·`report/` 두 디렉터리는 Windows 에서 여전히 한 폴더로 풀린다 — 같은 수정의 남은 절반이고 프로덕션 1파일·1함수다. 차선(문서 ORDER BY 에 d.id)은 '수정 전 실패' 를 만들 수 없어 또 미뤘고, safeFilename Windows 금지 문자는 다섯 경로 공유로 위험 3, OPERATIONS 문서 정정은 테스트로 증명할 것이 없어 제쳤다.
- 코드로 확정한 것: createFolder(workspaces.go:109)에 형제 이름 중복 검사 없음 / safeFolderSegment 는 절대 빈 문자열이 아님(safeFilename 이 "muni-document" 를 돌려줌)이라 result==prefix 는 없음 / 기존 live 테스트 `:271~291` 이 이 수정으로 반드시 깨짐(필터가 `"report/"` 를 contains 로 찾는다) / 테스트 헬퍼 네 개가 이미 있음 / splitTableRow 는 결함이 아니었음(기각).
- 추측으로 남긴 것: ORDER BY name,id 를 더해도 재귀(자식이 부모를 먼저 claim) 때문에 "먼저 만든 폴더가 원래 이름" 은 보장되지 않는다고 적었다 — 실행으로 확인하지 않았다. 그래서 수용 기준에서 어느 쪽이 ` (2)` 를 받는지 고정하지 말라고 못박았다. 이 워크트리에서 테스트를 돌리지 않았다(예산) — PASS 245/SKIP 0 은 직전 회차 기록이다.
- 구현자가 조심할 것: resolve 를 BFS 로 재작성하지 말 것(depth>32 순환 방어와 '부모가 없는 폴더는 루트' 동작이 조용히 바뀐다). claim 만 끼워 넣는 최소 수정. 마이그레이션으로 폴더 이름 유일 제약을 거는 것은 금지.
- [러너 17:22] scout done — 같은 이름(또는 대소문자만 다른 이름)의 폴더 둘이 워크스페이스 ZIP 안에서 한 디렉터리로 합쳐지는 것 �

## 구현 노트
- folderPaths 의 resolve 가 path.Join 직후 경로를 claim 하게 했다(이미 claim 된 이름이면 `기획 (2)`). claim 키는 기존 entryKey(ToLower)를 지나 대소문자만 다른 형제 폴더도 갈라지고, 반환 경로는 원래 대소문자를 보존한다. 프로덕션 1파일(workspace_export.go), 테스트 1파일.
- **과제서를 한 군데 넘어섰다 — 비평가가 먼저 볼 곳**: 지시된 SQL `ORDER BY name, id` 만으로는 효과가 **없다**(행이 map 으로 들어가 `for id := range all` 이 순서를 버린다). 읽은 순서를 슬라이스(`order`)로 남겨 순회하게 했고, 프로브로 둘을 분리 확인했다 — 슬라이스 없으면 열두 번 요청이 서로 다르고, 있으면 모두 동일.
- 확신 없는 것: 실제 Windows·macOS 폴더 병합은 리눅스 워크트리에서 재현 불가 — "ZIP 안의 어떤 두 디렉터리 경로도 ToLower 후 같지 않다" 까지만 증명했다. depth>32 와 '부모 없는 폴더의 자식은 루트' 는 코드로만 보존 확인했고(둘 다 `return ""` 이고 claim 하지 않는다), 전용 테스트는 쓰지 않았다.
- 일부러 하지 않은 것: 문서 쿼리(:71)의 `, d.id` (회차 하나에 한 조각), createFolder 의 형제 중복 검사(계약 판단 선행), 마이그레이션 유일 제약(금지), 단위 테스트(folderPaths 는 *Server·DB 를 받아 대역이 필요하고 운영자 지침이 대역 금지).
- 기존 live 테스트 `TestTwoTitlesDifferingOnlyInCaseStayTwoEntries` 의 디렉터리 블록을 갱신했다(필터를 prefix 로 넓히고, 단정을 '두 디렉터리가 ToLower 후 다르다 / 대소문자 보존 / 정확히 하나가 접미사' 로). 문서 제목 쪽 단정은 건드리지 않았고 계속 통과한다. 어느 폴더가 접미사를 받는지는 어느 테스트도 고정하지 않는다 — 자식 가진 폴더가 먼저 claim 할 수 있음을 실행으로 확인했다.
- 다음 역할이 조심할 것: 새 테스트는 `MUNI_TEST_DSN` 없으면 SKIP 된다. 검증은 postgres:16-alpine(포트 55601, 55432 는 다른 세션이 점유) 으로 돌렸고 httpapi PASS 246 / SKIP 0 / FAIL 0, `go test ./...` exit 0, vet·gofmt·placeholder 모두 통과. 프런트 미변경, `make build` 미실행.
- [러너 17:29] brief accepted — 채택 — 근거(`resolve` 가 중복을 보지 않는다, `createFolder` 에 형제 중복 검사가 없다, `entryKey` 재사용, 기존 live 테스트 `:271
- [러너 17:30] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다. postgres:16-alpine(55611)로 claim 루프(`workspace_export.go:271~274`)만 되돌리니 두 테스트가 주장된 증상 그대로 실패(`both 기획 folders became the one directory "기획/"`, `report/회의록.md` 와 `Report/회의록 (2).md` 가 한 디렉터리). 복원 후 httpapi PASS 246 / SKIP 0 / FAIL 0, `go test ./...` exit 0, vet·gofmt 통과 — 구현 노트 수치와 일치. **approve / risk low / blocking 없음.**
- 구현자가 고정하지 않았다고 밝힌 결정론 절반도 프로브로 확인했다(같은 이름 폴더 3개, 20회 내보내기: `order` 슬라이스만 되돌리면 2회차부터 어긋나고 복원하면 20회 동일). 커밋 메시지의 두 주장 모두 사실. 단 **그 절반에는 커밋된 테스트가 없다** — 다음 회차가 `range order` 를 `range all` 로 되돌려도 아무 테스트도 잡지 못한다.
- 읽어서 확인한 것: 접미사 루프는 후보가 단조 증가해 반드시 끝난다 / claim 이 memoize 직전이라 폴더당 최대 한 번 claim / depth>32·부모 없음 경로는 `return ""` 으로 claim 하지 않아 보존 / `" (N)"` 은 구분자나 `.`·`..` 를 만들 수 없어 탈출 방어 유지 / `claimed` 와 `used` 는 확장자 유무로 충돌하지 않음 / 기존 테스트의 `strings.Index(name,"/")` 는 `Contains` 로 가드됨. 보안·법무 차단 사유 없음(새 라우트·인가·비밀값·개인정보 변화 없음, `ORDER BY name, id` 는 리터럴, 마이그레이션 없어 revert 로 완전 복구).
- 못 본 것: 실제 Windows·macOS 압축 해제(리눅스에서 불가 — ZIP 항목 경로 비교까지만), depth>32·순환 전용 테스트, 프런트(미변경), Playwright.
- 남는 우려: `folders` 쿼리(:206)에 LIMIT 이 없어 같은 이름 폴더 n개에 O(n²) 맵 조회(공격 경로로는 약함, 참고) / 빈 폴더도 claim 하므로 문서 든 쪽이 `기획 (2)/` 를 받으면 ZIP 에 평범한 `기획/` 이 아예 없을 수 있음 — **릴리즈 노트는 어느 쪽이 접미사를 받는지 쓰지 말 것** / 다음 회차 후보: 라이브 루트 폴더 이름이 `휴지통` 이면 :169 하드코딩 접두사와 합쳐짐(이번 변경이 만들지도 악화시키지도 않음), safeFilename 의 `: * ? " < > |` / 워크트리 `webui/dist/index.html` 이 빌드 산출물로 더러움(커밋에는 없음 — 릴리즈 역할은 `git commit -a` 금지).
- [러너 17:35] review approved — 리뷰 승인 (risk=low)
- [러너 17:36] pr created — https://github.com/hkjang/muni/pull/34
- [러너 17:41] ci passed — 검사 2개 모두 success
- [러너 17:41] merge done — 17a7cd6
- [러너 17:53] release published — v0.54.0
- [러너 17:58] assets verified — v0.54.0 자산 1개 (이전 v0.53.0: 1)
