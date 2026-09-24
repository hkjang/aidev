# 회차 노트 2026-09-24-053427-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:34] base pinned — main@b40de35
- [러너 05:34] autonomy release — 

## 정찰 노트
- 고른 것: `safeFilename` 이 AI 컨텍스트용 `truncateRunes` 를 재사용해 101자 이상 제목의 파일 이름·ZIP 항목 이름에 줄바꿈과 「[…문서 컨텍스트가 길어 일부 생략됨…]」 을 붙이는 결함. 원인이 한 줄이고 다섯 호출 경로가 모두 한 패키지 안이라 S 로 끝난다.
- 제친 후보: ZIP 한도 경계(가치 2, 차선으로 남김), PDF 첫 줄 제목(첫 블록이 heading 인지 아직 미확인 — 근거 부족으로 세 회차째 보류), CI e2e(보호 경로·계정 시드로 한 세션 초과), 240자 제목 절단(데이터가 바뀌어 파급이 다름 — 별도 항목).
- 확신 없는 곳: 동적 재현을 하지 않았다(정찰은 코드 수정 금지). Go 가 헤더 값의 줄바꿈을 공백으로 정화할 것으로 보여 헤더 주입은 아닐 것 같으나 실제 바이트는 미확인 — 증거는 ZIP 항목 이름 쪽이 더 확실하다.
- 구현자가 조심할 것: `ai.go:238 truncateRunes` 와 `urlPathEscape` 는 넓히지 말고 `safeFilename` 안에서만 좁게 고칠 것. 이 환경의 `go test ./internal/httpapi/` 는 Chromium 이 없어 `TestDevtoolsPDFHasPageNumbers` 가 기존에도 실패한다(내 실행으로 확인).
- [러너 05:38] scout done — 내려받기 파일 이름과 ZIP 항목 이름에 AI 컨텍스트 안내 문구가 붙는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- `safeFilename`(export.go:307)이 100룬 절단에 AI 프롬프트용 `truncateRunes` 를 재사용해 101자 이상 제목의 내려받기 이름·ZIP 항목 이름에 줄바꿈과 「…생략됨…」 이 붙던 것을, 파일 이름 전용 `cutFilenameRunes` 로 바꿨습니다(자르기만 하고 문구 없음, 자른 자리의 끝 공백만 제거). 커밋 0864ae7.
- 정찰이 미확인으로 남긴 헤더 증상을 live 로 확정했습니다 — 헤더 주입은 아니고(net/http 가 줄바꿈을 공백으로 바꿈), 대신 `mime.ParseMediaType` 이 `Content-Disposition` 을 통째로 거부했습니다.
- **확신 없는 곳**: 고친 뒤에도 그 헤더는 `mime.ParseMediaType` 에 깨집니다 — `urlPathEscape` 가 `filename*` ext-value 의 한글 바이트를 그대로 두기 때문이며 **제목 길이와 무관한 별개의 기존 결함**입니다(짧은 한글 제목도 동일). 과제서가 넓히지 말라고 한 자리라 손대지 않고 live 테스트 주석과 ideas.json 에 새 항목으로 남겼습니다. 실제 브라우저가 이 헤더로 무엇을 저장하는지는 확인하지 못했습니다.
- 일부러 하지 않은 것: `ai.go:238 truncateRunes`, `urlPathEscape`, 치환기, 절단 길이 100, 빈 제목 처리, 가져오기·넘겨받기의 240자 제목 절단 — 모두 계약이 다르거나 DB 데이터가 바뀌는 자리라 범위 밖.
- 다음 역할이 조심할 것: live 테스트 2개는 `MUNI_TEST_DSN` 이 있어야 돕니다(없으면 SKIP). 검증은 postgres:16-alpine 컨테이너로 돌렸고 `go test ./...` 전체 통과, httpapi PASS 214 / SKIP 0 / FAIL 0.
- 정찰 노트 정정: 이 워크트리에는 쓸 수 있는 Chromium 이 있어 `TestDevtoolsPDFHasPageNumbers` 가 통과했습니다. "기존 실패" 로 넘기지 마세요.
- [러너 05:44] brief accepted — 채택 — 근거(`safeFilename` → `truncateRunes` 호출, 제목 240자 허용, 다섯 호출 경로가 한 패키지)가 모두 코드·실행과 맞아 지
- [러너 05:44] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 reject / security: handoff.go:181이 Fetch의 URL 포함 오류를 기록하여 claim을 노출함; 실패 전송으로 합성 토큰 노출 재현. 수리는 internal/handoff/handoff.go:275와 internal/httpapi/handoff.go:181부터 확인.
- 실제 main=1045e13으로 노트의 b40de35와 다름; 거절 결함은 요청된 main...HEAD 범위의 기존 handoff 커밋에 있고 최신 0864ae7에는 없음.
- 파일 이름 경계·호출 경로·신규 단언·SSO 및 handoff 보호·마이그레이션을 확인; 관련 단위 및 handoff 테스트와 diff --check 통과.
- 새 live 2개는 DSN 없어 SKIP; 전체 빌드·브라우저·DB 실행은 미검증. 한글 헤더 파싱 기존 문제는 별도 후속 과제로 남김.
- [러너 05:46] review rejected — 리뷰 거절: internal/httpapi/handoff.go:181 [P1] receiveHandoff가 handoff.Fetch 오류 문자열을 그대로 경고 로그에 기록한다. internal/handoff/handoff.go:275는 client.Do의 *url.Er
- [러너 05:46] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 05:46] pr created — https://github.com/hkjang/muni/pull/25
