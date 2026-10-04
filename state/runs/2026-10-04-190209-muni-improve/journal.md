# 회차 노트 2026-10-04-190209-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:02] base pinned — main@b84a38b
- [러너 19:02] autonomy release — 

## 정찰 노트
- 고른 이유: 같은 계열(사용자 입력 → ZIP 항목 이름)에서 **수정 전 결정적으로 실패시킬 수 있는** 유일한 남은 자리였습니다. 차선인 정렬 동률(`, d.id`)은 PostgreSQL 이 우연히 같은 순서를 돌려주면 먼저 실패시킬 수 없어 밀었고, Windows 금지 문자(3/3/M)는 다섯 경로 공유 + 실제 Windows 재현 불가로, Content-Disposition 헬퍼 통합은 c991f0b 로 차이가 사라져 `rejected` 로 내렸습니다.
- 추측으로 적은 것: Windows/macOS 에서 실제로 뒤 항목이 앞을 덮는다는 것은 파일 시스템 계약에 기댄 추론이고 이 리눅스 워크트리에서 재현하지 못했습니다 — 과제서의 수용 기준은 "ZIP 항목 이름이 대소문자 무시로 유일하다" 까지로 좁혀 두었습니다. 폴더 이름 중복을 UI 가 허용하는지(신규 아이디어)는 확인하지 않았습니다.
- 구현자가 조심할 것: 접기는 `used` 의 **키에만** 적용하고 반환 이름·ZIP 에 쓰는 이름은 원래 대소문자를 유지할 것(`README` 가 `readme` 로 바뀌면 사용자가 보는 파일 이름이 달라짐). `:149` 의 안내 파일 예약도 같은 헬퍼를 지나게 할 것. `safeFilename` 은 열지 말 것 — 다섯 경로가 공유합니다.
- 돌려 본 것: 읽기만 했고 테스트는 돌리지 않았습니다(예산). 코드 근거는 `workspace_export.go:71,149,171,271,281`, `workspace_export_test.go:31,67,83,95,103` 을 직접 열어 확인했습니다.
- [러너 19:05] scout done — 워크스페이스 ZIP 항목 이름이 대소문자만 다를 때 압축 해제에서 한쪽이 덮이는 것 막기 (가치 3 / 위험 1 /

## 구현 노트
- 무엇·왜: `uniqueEntryName` 의 중복 추적 맵 `used` 가 이름을 바이트 그대로 비교해 `Report.md`·`report.md` 가 ZIP 안에 두 항목으로 들어갔다. 비교 키만 접는 헬퍼 `entryKey(name)=strings.ToLower(name)` 를 두고 맵의 키와 `:149` 안내 파일 예약을 그것으로 통과시켰다(커밋 a55bf5f, 프로덕션 1파일 + 테스트 2파일).
- 확신 없는 곳: 실제 Windows/macOS 에서 뒤 항목이 앞을 덮는 것은 **재현하지 못했다**(리눅스 FS 는 대소문자 구분). 증명한 것은 "ZIP 항목 이름이 대소문자 무시로 유일하다" 까지. `:149` 예약의 `entryKey` 통과는 `목록.md` 에서 `ToLower` 가 항등이므로 **동작 변화가 없고 어떤 테스트도 잡지 않는다** — 예약과 조회가 같은 규칙을 쓰게 하는 방어다(과제서 지시).
- `strings.ToLower` 는 유니코드 전체를 접는다. 한글은 영향 없고 라틴·그리스·키릴 제목은 이제 대소문자 차이로 `(2)` 를 받는다 — 의도한 동작이지만 터키어 `I`/`ı` 같은 로케일 특수 규칙까지 맞추지는 않는다(Windows 도 그렇게 비교하지 않는다).
- 일부러 안 한 것: `ORDER BY` 에 `, d.id` 추가(별 과제), `safeFilename`·`safeFolderSegment`·`workspaceManifestName` 손대기, 프런트 검사·`make build`(프런트 미변경, `make build` 는 tracked placeholder 를 덮는다).
- 다음 역할이 조심할 것: 새 live 테스트 `TestTwoTitlesDifferingOnlyInCaseStayTwoEntries` 는 **DB 가 있어야 돈다**(`MUNI_TEST_DSN`). 어느 쪽 문서가 `(2)` 를 받는지는 정렬 동률에 달려 비결정적이므로 테스트는 일부러 고정하지 않는다. `go test -count=1 -v ./internal/httpapi` = PASS 245 / SKIP 0(직전 243 + 새 2). 이 워크트리에는 Chromium 이 있어 PDF 테스트도 통과했다.
- [러너 19:11] brief accepted — 채택 — 근거(`used` 가 대소문자를 구분한다, `uniqueEntryName` 은 `:171` 한 자리만 쓴다, `:149` 예약이 같은 맵을 쓴다)가 코드�
- [러너 19:12] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: `b84a38b...HEAD` 는 프로덕션 1파일 + 테스트 2파일뿐(워크트리의 `main` 은 v0.38.0 로 낡아 `git diff main...HEAD` 는 쓸 수 없다 — 핀된 base 로 봤다). `entryKey` 는 `used` 의 키에만 쓰이고 반환 이름·`archive.Create` 인자는 원래 대소문자를 유지한다. `uniqueEntryName` 호출부는 `:171` 한 자리뿐이라 모든 ZIP 항목(문서 + `:190` 목록.md)이 같은 접힌 맵을 지나고, '항목 이름이 대소문자 무시로 유일' 불변식이 실제로 성립한다. 단위 테스트는 수정 전 `report.md` 를 돌려받아 결정적으로 실패하고(읽어서 확정), `ledger-entry.md:5` 의 live FAIL 출력도 증상과 일치한다. live 테스트를 Report/report 양쪽 정렬 순서로 손으로 따라가 어느 쪽이 `(2)` 를 받아도 네 단언이 모두 성립함을 확인 — 깜빡이지 않는다. 범위 이탈·인증·마이그레이션·비밀값 접촉 없고 revert 로 완전히 되돌아온다.
- 못 본 것: 이 워크트리에 `MUNI_TEST_DSN` 이 없어 live 테스트는 SKIP 되었다. 내가 돌려 통과한 것은 `go test ./...` 전체 PASS·`go vet`·`gofmt -l`(무출력)까지이고, 'PASS 245 / SKIP 0' 은 재현하지 않았다 — CI 에 맡긴다. 실제 Windows/macOS 에서의 덮어쓰기도 재현하지 않았다(구현자와 동일).
- **릴리즈가 가장 먼저 볼 것**: 워크트리에 커밋되지 않은 `webui/dist/index.html` 이 빌드 산출물로 남아 있고 지금 `scripts/check-webui-placeholder.sh` 가 EXIT=1 이다. 커밋 a55bf5f 에는 없으므로 심사 대상은 아니지만, `git add -A`/`git commit -a` 를 쓰면 저장소에 없는 에셋을 참조하는 index.html 이 섞인다 — 먼저 `git checkout -- webui/dist/index.html`.
- 릴리스 노트에 남는 우려: 이 변경이 지키는 것은 **항목 이름의 유일성(문서 분실 방지)**까지다. 대소문자만 다른 폴더 둘은 Windows 에서 여전히 한 디렉터리로 합쳐져 폴더 구조가 뭉개지고(안쪽 문서는 살아남는다), `목록.md` 의 각 줄은 접기 이전 이름을 적어 Windows 에서 디스크 경로와 대소문자가 어긋날 수 있다. 둘 다 기존 동작이고 분실은 아니다 — 폴더 구조 보존까지 약속하지 말 것. `live_test.go:217` 의 '두 폴더를 구별해야 한다' 주석도 이 뜻으로 읽히게 다음 회차에 다듬을 여지가 있다.
- 다음 회차용: `strings.ToLower` 는 NTFS 대문자 테이블보다 넓게 접어(켈빈 기호 U+212A→k) 충돌하지 않는 문서에 불필요한 `(2)` 가 붙을 수 있다 — 분실 방향으로는 틀리지 않으므로 수용. 보류 아이디어 중 `ORDER BY ... , d.id` 는 이번 회차가 비결정성의 live 증거를 남겨 두었으니 다음 후보로 유력하다.
- [러너 19:15] review approved — 리뷰 승인 (risk=low)
- [러너 19:16] pr created — https://github.com/hkjang/muni/pull/33
- [러너 19:21] ci passed — 검사 2개 모두 success
- [러너 19:21] merge done — a55bf5f

## 릴리즈 노트
- 한 것: VERSION `0.52.0`→`0.53.0`, `docs/releases/v0.53.0.md` 신규, 릴리즈 커밋 `docs: v0.53.0 릴리스 노트`(2a93c60), 주석 태그 `v0.53.0`. 직전 세 태그를 열어 양식을 확인했고 이전 두 릴리즈 커밋이 손댄 파일이 정확히 그 둘(VERSION + 노트)뿐임을 확인해 그대로 따랐습니다. 버전은 0.x 에서 마이너를 올려 온 패턴(v0.50→v0.51→v0.52)대로 마이너.
- 버전 증가 근거 확인: 작업 시작 시 `VERSION` 이 `0.52.0` 이었으므로 다른 세션이 사이에 릴리즈하지 않았습니다.
- 자산·GitHub Release: 둘 다 **내가 만들지 않았습니다.** `.github/workflows/release.yml` 이 `v*.*.*` 태그 푸시에 반응해 이미지 빌드 → `docker save | gzip -9` → 내부 이미지 태그 검증 → `softprops/action-gh-release` 로 Release 생성과 `muni-v0.53.0.tar.gz` 업로드까지 합니다. 그래서 `release.json` 의 `assets` 는 빈 배열, `github_release` 는 false. 노트 본문은 `sync-release-notes.yml` 이 main 의 `docs/releases/v*.md` 푸시에서 Release 에 덮어씁니다 — 노트가 Release 에 실리려면 **태그 푸시와 별도로 main 푸시가 필요**합니다.
- 돌린 검증(릴리즈 전, 이 기계): 전용 PostgreSQL 16 컨테이너(`muni-rel-053-pg`, 15953 포트)로 `MUNI_TEST_DSN` 을 주고 `go test -count=1 -json ./...` = **721 PASS / 4 SKIP / 0 FAIL**. SKIP 넷은 직전 릴리즈들과 같은 외부 코퍼스·수동 출력(`docx TestWriteSampleForInspection`, `hwp TestCorpus`, `hwpx TestCorpus`, `hwpx TestRealRoundTrip`)이고, 새 테스트 둘(`TestTwoTitlesDifferingOnlyInCaseStayTwoEntries` live, `...AreStillTwoNames` 단위)은 SKIP 없이 PASS — 비평이 못 봤다고 남긴 자리를 메웠습니다. 프런트 `npm ci`·`npm run lint`(tsc -b)·`npm test`(42파일 297건)·`npm run build`, `go vet ./...`, `gofmt -l`(무출력), placeholder 검사, `git diff --check`, `CGO_ENABLED=0 go build -trimpath`, `docker build`(VERSION=v0.53.0) 모두 통과.
- **비평이 경고한 자리 처리**: `npm run build` 가 tracked `webui/dist/index.html` 을 덮어써 placeholder 검사가 깨지는 것을 실제로 겪었고, `git add` 전에 `git checkout -- webui/` 로 되돌린 뒤 placeholder 검사를 다시 통과시켰습니다. 릴리즈 커밋에는 VERSION 과 노트 둘만 들어갔습니다(`git add` 로 경로를 명시, `-a`/`-A` 미사용).
- **비평이 '릴리스 노트에 남는 우려'로 지시한 것을 노트에 그대로 반영**: 지키는 것은 항목 이름 유일성(=문서 분실 방지)까지이고, 대소문자로만 다른 폴더 둘이 Windows 에서 한 디렉터리로 합쳐져 **폴더 구조는 뭉개진 채로 남는다**는 것, `목록.md` 의 줄과 디스크 폴더 이름의 대소문자가 어긋날 수 있다는 것을 '남아 있는 것'에 적었고 태그 주석에도 "폴더 구조 보존은 약속하지 않습니다" 를 넣었습니다. 노트 초안에는 `목록.md` 어긋남을 `(2)` 접미사 탓으로 적었는데 `workspace_export.go:184` 가 `name` 을 그대로 적는 것을 확인해 원인을 폴더 합쳐짐으로 고쳤습니다(커밋 amend, 태그 재생성 — 푸시 전이었습니다).
- 증명하지 않은 것(노트에 명시): 실제 Windows·macOS 압축 풀기, 브라우저 E2E, 실제 운영 배포. 어느 문서가 `(2)` 를 받는지 고정하지 않는 것도 노트의 '남아 있는 것'에 적었습니다.
- 푸시하지 않았습니다: 커밋·태그 모두 로컬(detached HEAD)이며 브랜치를 만들거나 옮기지 않았습니다. 검증용 PostgreSQL 컨테이너와 `muni:v0.53.0-relcheck` 이미지는 지웠습니다.
- 다음 회차용: 보류 아이디어의 `ORDER BY ... , d.id` 는 이번 노트가 "어느 쪽이 `(2)` 를 받는지 고정하지 않았다" 를 공개 약속으로 적었으므로, 그것을 결정적으로 만들 때 이 문장을 함께 고쳐야 합니다.
- [러너 19:36] release published — v0.53.0
