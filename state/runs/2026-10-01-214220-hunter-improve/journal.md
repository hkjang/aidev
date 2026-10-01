# 회차 노트 2026-10-01-214220-hunter-improve — hunter
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:42] base pinned — main@f0f7667
- [러너 21:42] autonomy release — 

## 구현 노트
- `web/src/workflow-navigation.ts` `sbomLabelError`가 UTF-8 200바이트만 보고 제어 문자를 놓쳐서, 탭이 섞인 SBOM 문서 이름이 최대 8MiB 업로드와 `parseSBOM`을 다 쓰고 나서야 서버 400으로 거절됐다. trim 후 값 하나에 제어 문자 검사와 바이트 검사를 모두 적용했고, `internal/app/testdata/sbom-label.json` 23벡터로 Go `sbomSafeIdentifier`와 TS `sbomLabelError`를 같은 입력으로 교차 검증했다. 프로덕션 1파일.
- **확신 없는 곳·검증 못 한 것**: (1) 탭이 실제 브라우저 붙여넣기에서 single-line input에 남는다는 것은 HTML 값 정규화 규격(CR/LF만 제거) 근거이고 실제 Chrome/Firefox 화면에서 붙여넣어 확인하지는 않았다 — 탭이 제거되는 브라우저가 있어도 다른 제어 문자(U+0085 등 프로그램 경로/자동입력)로 경로는 남지만 "표에서 붙여 넣기"라는 설명의 현실성은 미검증이다. (2) 새 한국어 오류 문구를 실제 화면에서 보지 않았다(모달 Alert·필드 error 양쪽에 같은 문자열이 그대로 들어간다는 것은 `software.tsx:331-337,536-562` 코드로만 확인). (3) `npm test`를 npm 스크립트로 돌리면 이 환경의 npm이 `--experimental-strip-types` 없는 node(v20.19.2 보고됨)를 집어 `bad option`으로 죽는다 — package.json의 같은 명령을 `~/.nvm/versions/node/v22.23.1/bin/node`로 직접 실행해 105통과/0실패를 얻었다. CI Node 26 동등성은 미검증.
- **일부러 하지 않은 것**: 서버 `importSBOM`의 검사 순서(parseSBOM 먼저, 라벨 검사 나중)는 건드리지 않았다 — 서버 계약 변경이고 DB HTTP 회귀가 필요하다. `software.tsx`도 손대지 않았다(새 오류가 기존 `error=`/`disabled` 경로로 그대로 흐른다). 문서·버전·릴리즈 파일 무변경.
- **다음 역할이 조심할 것**: `TestSBOMLabelSharedVectors`는 DSN 없이 돈다. 같은 `-run '^TestSBOM'`에 걸리는 `TestSBOMLabelsEnforceUTF8ByteLimits` 등 3개는 `HUNTER_TEST_DSN` 없으면 SKIP이며 통과로 세면 안 된다. 벡터의 `wire`는 반드시 `label.trim()`이어야 하고(웹 테스트가 단언), 웹 검사를 원문(`value`) 기준으로 되돌리면 U+000B/U+000A/U+FEFF 말단 사례가 깨진다 — 실제로 그 오답을 넣어 실패를 확인해 두었다. `go test -race ./...`(DB 필요)와 원격 CI·배포는 미실행.
- [러너 21:52] verify failed — 실패한 검증: cd web && npm test --silent (exit 9)
