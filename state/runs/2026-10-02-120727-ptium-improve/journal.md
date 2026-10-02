# 회차 노트 2026-10-02-120727-ptium-improve — ptium
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:07] base pinned — main@70a0c4e
- [러너 12:07] autonomy release — 

## 정찰 노트
- 닫는 `#` 시퀀스를 골랐다: 지난 회차가 1순위로 등록해 둔 것이고, 임시 테스트를 실제로 돌려 여섯 입력의 현재 출력을 눈으로 확인했고(제목·덱 이름·`!source` locator 세 곳이 함께 틀린다), 고칠 자리가 `atxHeading` 한 함수 안에서 끝난다(프로덕션 1파일 + 테스트 1파일).
- 제친 후보: 수평선(`---`/`***`)이 `- \---` 요점으로 남는 것은 가치가 더 높다고 봤지만(front matter 문서에서 junk 세 줄) `isListLine` 과의 순서·`- - -` 형태 때문에 위험이 1 더 높아 차선으로 돌렸다 — 둘 다 같은 `handle` 을 건드리므로 **같은 회차에 둘을 같이 하지 말 것**.
- 추측으로 적은 곳: `## ###` 의 **고친 뒤** 출력 문자열은 확인하지 않았다(고치기 전만 봤다). 구현자는 실행으로 확인해 박을 것. setext h2·front matter 는 범위 밖이고, `# 제목 \###` 는 현 동작 유지가 의도다.
- 조심할 것: `internal/deck/source.go`(덱 DSL)·`writer.go`·숫자 파서는 닿지 말 것. 임시로 만든 `zzrecon_test.go` 는 지웠고 작업 트리는 건드리지 않았다.
- [러너 12:14] scout done — ATX 제목의 닫는 `#` 시퀀스를 제목에서 떼기 — `## 분기 요약 ##` 이 덱 이름까지 `분기 요약 ##` 으로 만드는

## 구현 노트
- `atxHeading`(prose.go:422) 이 `text` 를 돌려주기 전 끝의 `#` 런을 뗀다 — `TrimRight` 가 실제로 줄였고 그 앞이 공백/탭이거나 남은 게 없을 때만. 제목·덱 이름·`!source` locator 가 함께 고쳐진다. 커밋 5bad44a, 프로덕션 1파일 + 테스트 1파일 + `USER_GUIDE.md:276` 한 줄.
- 확신 없는 곳: 없다고 말하진 않겠다 — 테스트가 `Read()` 의 `document.Source` 전체를 비교하므로 바뀐 경로를 실제로 지나고, 고치기 전 red 를 눈으로 봤다. 정찰이 미확인이라던 `## ###` 의 고친 뒤 출력은 실행으로 확인했고 기존 빈 제목 모양과 같았다.
- 검증 못 한 것: DB(PTIUM_TEST_DSN 없음 → Skip), 웹 단계(웹·API 변경 없어 건너뜀), 실서버·Docker·릴리즈 경로. `make test` 전체는 돌리지 않고 `go test -race ./...` + `go vet ./...` 로 대신했다.
- 일부러 하지 않은 것: 백슬래시로 지킨 닫는 시퀀스(`# 제목 \###`)는 현 동작 유지 — 커밋 메시지에 범위 밖이라고 적었다. 차선 후보였던 수평선(`---`)은 같은 `handle` 을 건드려 같은 회차에 넣지 않았다. `deck/source.go`·`writer.go`·`tables.go`·숫자 파서는 닿지 않았다.
- 다음 역할이 조심할 것: 새 테스트 3개는 DB 없이 돈다. `TestAHashInsideAHeadingIsNotAClosingSequence` 는 고치기 전에도 green 인 계약 고정용이므로 "red 를 못 봤다" 가 아니다. 미닫힘 펜스 재생 루프도 같은 `handle` 을 부르므로 새 규칙이 자동으로 적용되지만 그 자리의 테스트는 두지 않았다.
- [러너 12:23] brief accepted — 채택 — 재현 여섯 입력과 권장 형태(`TrimRight` 뒤 끝 글자가 공백/탭이거나 빈 문자열일 때만)가 현재 코드와 그대로 맞아
- [러너 12:23] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인한 것: base(70a0c4e) 를 detach 워크트리로 꺼내 새 테스트만 얹어 **red 를 직접 재현**했다 — 닫는 해시 3건과 `## ###` 가 FAIL 하고 출력이 주장한 증상(제목·덱 이름·`!source` locator 세 곳)과 같다. `TestAHashInsideAHeadingIsNotAClosingSequence` 만 base 에서 green 으로, 구현자가 밝힌 대로 계약 고정용이다.
- 경계도 새 코드에 직접 입력 14개를 넣어 봤다: `# 요약 ## ##`→`요약 ##`, `# 1 ## 2 ##`→`1 ## 2`, `# ##`·`# #`→빈 제목, `# C# 도입 ##`→`C# 도입`, `# 제목#`·전각 공백 앞 `##`→그대로. 전부 CommonMark 와 맞다. `closing[len-1]` 바이트 검사는 UTF-8 연속 바이트(0x80~0xBF)가 0x20/0x09 와 겹치지 않아 안전하다. 서버 `go vet ./...` + `go test -race ./...` 통과. 구현자가 테스트를 두지 않았다던 미닫힘 펜스 재생 경로도 손으로 확인했고 새 규칙이 적용된다.
- 못 본 것: DB(DSN 없음)·웹·Docker·릴리즈 스크립트. 웹·API·마이그레이션 변경이 없어 필요하다고 보지 않았다.
- 승인이어도 남는 우려(릴리즈가 적을 때 참고): `# 제목 \###` 는 여전히 `제목 \###` 로 백슬래시가 보인다(기존 동작, 범위 밖이라 커밋에 명시됨). 테스트 세 번째 입력의 끝 공백은 `readMarkdown` 의 TrimSpace 때문에 `atxHeading` 까지 가지 않아 중복이다 — 해롭지는 않다.
- 다음 회차: 수평선 `---`/`***` 는 같은 `handle` 을 건드리므로 이번 변경과 묶지 말라는 정찰 메모가 유효하다. 보안·법무 차단 사유 없음(인증·비밀값·의존성·개인정보에 닿지 않고, 제목 끝 문자를 덜어내는 변경이라 덱 DSL 에 새 문자 종류를 노출하지 않는다).
- [러너 12:28] review approved — 리뷰 승인 (risk=low)
- [러너 12:28] pr created — https://github.com/hkjang/ptium/pull/40
- [러너 12:33] ci passed — 검사 1개 모두 success
- [러너 12:33] merge done — 5bad44a
- [러너 12:43] release published — v1.69.54
- [러너 12:43] gh-release created — GitHub Release v1.69.54
- [러너 12:43] manifest ok — ptium-1.69.54.tar.gz ptium-1.69.54.tar.gz.sha256 docker-compose.ptium-1.69.54.yml ptium-1.69.54.env.example load-ptium-1.69.54.ps1 load-ptium-1.69.54.sh ptium-1.69.54.kubernetes.yaml 
- [러너 12:43] assets uploaded — 7개
- [러너 12:43] assets verified — v1.69.54 자산 7개 (이전 v1.69.53: 7)
