# 회차 노트 2026-09-28-145913-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:59] base pinned — main@051a309
- [러너 14:59] autonomy release — 

## 정찰 노트
- 고른 이유: 최근 네 회차가 "사용자 입력이 파일 이름·헤더가 되는 자리" 에서 나왔는데 그 계열의 **남은 한 자리**를 찾았다 — `import_html.go:66,68` 의 `imageAssetName` 이 아직 AI 프롬프트용 `truncateRunes` 를 쓴다(제어문자를 `strings.Map` 으로 지운 직후 절단이 줄바꿈을 다시 넣는 순서 결함). 고칠 함수가 같은 패키지에 이미 있어(`cutFilenameRunes`) 프로덕션 파일 하나로 끝난다. 워크스페이스 ZIP 한도 안내(2000건 live 재현이 무거움)와 240룬 제목 절단(도달 가능성 여덟 회차째 미확인)을 이 때문에 제쳤다.
- 추측으로 적은 것: **동적 재현을 전혀 못 했다** — 이 샌드박스에서 `go vet` 이 승인 대기로 막혔다(프로필이 예고한 함정 그대로). 저장된 첨부 이름이 240룬 절단에 걸리지 않는다는 계산(≈228룬)과 마크다운 인라인 파서가 201룬 alt 를 그대로 통과시키는지는 코드 읽기만 한 추정이다.
- 구현자가 조심할 것: `ai.go:238` `truncateRunes` 는 호출자가 20곳이고 AI 컨텍스트에서는 그 문구가 의도된 동작이니 건드리지 말 것. `internal/docx/import.go:724` 에 문구를 붙이지 않는 동명의 다른 함수가 있어 혼동하기 쉽다.
- 프로필은 2일 전 것이고 지금 코드와 어긋나지 않아 새로 쓰지 않았다. 다만 truncateRunes 누출 자리 목록에 `import_html.go:66,68` 이 빠져 있으니 다음 회차가 프로필을 갱신할 때 더할 것.
- [러너 15:03] scout done — 가져온 이미지 이름(`imageAssetName`)에 AI 컨텍스트 안내 문구와 줄바꿈이 들어가는 것 고치기 (가치 3 / 위험 

## 구현 노트
- `import_html.go:imageAssetName` 의 200룬 절단 두 자리를 `truncateRunes`(AI 프롬프트용, 자르면 `\n[…문서 컨텍스트가 길어…]` 를 덧붙임) → 같은 패키지의 `cutFilenameRunes` 로 바꿨다. 프로덕션 파일 하나, 테스트 파일 둘. 커밋 f5039e3.
- 확신 없는 곳: 없다시피 하다. 정찰이 미확인으로 남긴 둘을 실행으로 확인했다 — 저장된 이름은 228룬이라 `attachments.name` 의 240 절단에 걸리지 않고(live 가 DB 값을 직접 읽음), 마크다운 `![201룬](data:…)` 도 파서를 그대로 통과한다(단위 하위 테스트 markdown). 되돌리기 검증도 했다: 두 줄만 되돌리면 새 테스트 둘만 실패하고 나머지 236건은 통과.
- 굳이 보자면: 새 live 테스트는 `attachments` 를 `WHERE document_id=$1` 로 단일 행 조회한다. 이 문서에는 그림이 하나뿐이라 안전하지만 행이 둘이면 임의의 한 행을 읽는다.
- 일부러 하지 않은 것: `truncateRunes`(ai.go:238)·`safeFilename`·`extValueEscape` 본문 미변경(과제서 지시). `import_attachments.go:121`·`handoff.go:230` 의 240룬 **제목** 절단에도 같은 문구가 남아 있지만 파일 이름이 아니라 DB 데이터이고 241룬 도달 가능성이 미확인이라 범위 밖으로 뒀다 — 이번 회차로 truncateRunes 누출 자리는 그 둘만 남았다.
- 다음 역할이 조심할 것: `TestAnImportedImageNamesItselfWithoutAContextNotice` 는 live 라 `MUNI_TEST_DSN` 이 없으면 SKIP 된다. 검증은 postgres:16-alpine 컨테이너에 DSN 을 주어 `go test -count=1 -v ./internal/httpapi` PASS 238 / SKIP 0 / FAIL 0 으로 확인했고(이 워크트리엔 Chromium 이 있어 PDF 테스트도 통과), `go test ./...`·`go vet ./...`·`gofmt -l .`·`scripts/check-webui-placeholder.sh` 모두 통과. 프런트 미변경이라 npm 검사는 돌리지 않았다.
- [러너 15:07] brief accepted — 채택 — 근거(`import_html.go:66,68` 의 `truncateRunes`, 제어문자 제거 후 절단이라는 순서, 같은 패키지의 `cutFilenameRunes`, `inlineCon
- [러너 15:08] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 두 줄을 truncateRunes 로 되돌려 **실패를 직접 재현**했다(원장에 `- 실패 재현:` 줄이 없었다) — html·markdown 두 하위 테스트가 `가…가\n[…문서 컨텍스트가…].png` 로 실패하고 복원하면 통과. go vet·gofmt 통과, 첨부 내려받기가 documentRole 검사와 extValueEscape 를 지나는 것, filenameFrom 이 filename* 디코딩 값을 읽는 것까지 읽어 확인. 보안·법무 차단 사유 없음.
- 못 본 것: live 테스트(MUNI_TEST_DSN 없음)와 프런트. 컴파일만 확인했다.
- 승인이어도 남는 우려: 같은 계열의 결함이 `import_attachments.go:570` 에 남아 있다 — 손으로 올린 첨부는 `truncateRunes(filepath.Base(header.Filename), 240)` 를 쓰고 multipart 파일 이름에 상한이 없어 241룬 이름이면 안내 문구가 attachments.name 과 Content-Disposition 에 나간다. :183·:312 의 가져온 그림 이름(docx/hwp/hwpx 는 zip 항목 이름이 그대로 Asset.Name)도 같다. 구현 노트의 "남은 자리는 240룬 제목 둘뿐" 은 정확하지 않으니 다음 회차 후보로 둘 것.
- 릴리즈가 볼 것: 워크트리의 `webui/dist/index.html` 이 빌드 산출물로 더럽혀져 `scripts/check-webui-placeholder.sh` 가 지금 실패한다(커밋에는 없음). 태그 전에 `git checkout -- webui/dist/index.html`.
- 프로필 갱신 시: truncateRunes 누출 자리 목록에 `import_attachments.go:183,312,570` 을 더하고, `import_html.go:66,68` 은 이번에 닫혔으니 뺄 것.
- [러너 15:12] review approved — 리뷰 승인 (risk=low)
- [러너 15:13] pr created — https://github.com/hkjang/muni/pull/29
- [러너 15:18] ci passed — 검사 2개 모두 success
- [러너 15:18] merge done — f5039e3
- [러너 15:28] release ci-blocked — 릴리즈 커밋 CI: failed — 성공이 아닌 검사: sync=failure (태그 보류)
