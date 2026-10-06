# 회차 노트 2026-10-07-031239-AgentHub-improve — AgentHub
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:12] base pinned — main@49c4a1f
- [러너 03:12] autonomy release — 

## 정찰 노트
- 우선 과제가 지목한 "릴리즈 워크플로 2회 실패" 는 **확인할 수 없었다** — `gh run list` 가 이 환경에서 허용되지 않아 Actions 로그를 못 봤고, 저장소 작업 트리는 깨끗하며 v0.260.0 까지 정상 릴리즈됐다. 남은 단서인 TIMEOUT 에 대해 정찰이 직접 증명할 수 있는 비용을 쟀다: `go test ./cmd/... ./internal/...` 20초 중 **17.44초**가 `internal/buildinfo` 하나이고 그 전부가 `TestTheImageVersionCheckCatchesTheReleaseItWasWrittenFor` 다(`check-versions` 를 두 과거 커밋에 각각 ≈8.7초). 검증 루프를 여러 번 도는 회차가 매번 내는 요금이라 이걸 골랐다.
- 추적 상한 계열(차선 후보)을 제친 이유: 다섯 회차 연속 같은 자리였고 이번 우선 과제가 "새 아이디어 대신 실패를 고치라" 고 했다. 보호 경로(.github/workflows, CI 에 Node 회귀 추가)는 규칙대로 제외했다.
- 과제서에서 추측으로 적은 것: 8.7초가 jq/git **프로세스 생성** 비용이라는 분해는 측정이 아니라 추론이다(권한 제약으로 `time`·spawn 벤치를 못 돌렸다). 구현자는 고치기 전에 jq 가 지배적인지 먼저 확인하고, 아니면 차선으로 옮겨야 한다.
- 구현자가 조심할 것: `catalog_entry` 가 서브셸·파이프 안에서 불리는 자리가 있어 거기서 채운 캐시는 부모에 안 남는다 — 시작 시 `jq '… | @base64'` 한 번으로 전부 채우는 형태를 먼저 쓸 것. `check_versions`·`image_inputs_changed` 의 판정 논리와 테스트의 두 커밋 해시는 한 줄도 바꾸지 말 것(빨라지고 느슨해지는 것이 이번 과제의 유일한 실패 모드다).
- [러너 03:18] scout done — `catalog_entry` 의 jq 재호출을 메모이즈해 `check-versions` 를 빠르게 한다 — Go 테스트 스위트의 87% 를 차지하는

## 구현 노트
- 과제서(catalog_entry jq 메모이즈)는 측정으로 기각했다: jq 7.0ms/회 vs git show 115ms/회, check-versions 의 ~8초 중 85%가 git 72회다. 원인은 이 워크트리의 gitdir 이 /mnt/c(9p)에 있는 것 — 로컬 디스크 저장소에서 같은 git show 는 6.2ms/회(19배). 프리필 캐시를 실제로 만들어 재 봤고(jq 128→73, 벽시계 변화 없음) 되돌렸다. scripts/ 는 한 줄도 바뀌지 않았다.
- 대신 차선 후보를 구현: 추적 설정의 상한 아홉 개를 콘솔이 미리 보여 준다. 제공자 id/URL 에 maxLength, 허용 출처 목록에 사용량 카운터. 커밋 e68673c, 프로덕션 2파일 + 테스트 1파일.
- **확신 없는 곳**: (1) 브라우저 DOM 으로는 확인하지 못했다 — 실행 중인 콘솔과 관리자 세션이 필요하다. 번들(dist/assets/AdminSettings-*.js)에 maxLength 5개와 상한 아홉 값이 실린 것까지만 봤다. (2) allowedHostsUsage 는 splitHosts 를 옮긴 것이지만 Go 의 TrimSpace 와 JS 의 trim() 이 유니코드 공백 집합에서 완전히 같은지는 확인하지 않았다 — \v·\f·U+00A0 같은 문자가 주소에 섞인 경우 카운터가 서버와 1~2 다를 수 있다. 서버가 최종 판정을 하므로 틀려도 안내가 조금 어긋나는 것이고 저장 결과는 바뀌지 않는다. (3) maxLength 는 UTF-16 단위라 네 바이트 문자에서 서버의 룬 계산보다 엄하다(주석에 적었다).
- **일부러 하지 않은 것**: 스니펫 안의 출처 개수는 세지 않는다 — SnippetOrigins 파서를 TS 로 복제하면 두 곳이 같은 스니펫을 다르게 읽어 안내가 거짓이 된다. 상한만 산문으로 적었다. .github/workflows 는 보호 경로라 새 테스트를 CI 에 넣지 않았다.
- **다음 역할이 조심할 것**: 새 테스트 web/scripts/tracking-limits-check.test.mjs 는 CI 가 돌리지 않는다(ci.yaml 은 test:sso 하나만). 돌리려면 `cd web && node --test scripts/tracking-limits-check.test.mjs` — node 22 의 .ts 타입 스트리핑에 의존하므로 node 22.18+ 가 필요하다. 이 테스트는 internal/tracking/tracking.go 의 상수를 읽으므로 **Go 쪽 상한을 바꾸면 여기서 실패한다** — 그때는 web/src/pages/trackingLimits.ts 의 같은 이름 값을 같이 고치는 것이 의도된 동작이다.
- [러너 03:37] brief rejected — 기각 — 과제서의 핵심 가설("8.7초는 jq 프로세스 생성 비용")이 **측정으로 틀렸다**. 과제서가 "구현자는 먼저 jq 가 지배�
- [러너 03:38] verify passed — 검증 5개 통과 (auto)

## 비평 노트
- 거절. 수리가 먼저 볼 파일은 **web/src/pages/trackingLimits.ts:36** — `/[,\s]+/` 는 splitHosts 가 아니다(Go 는 `,`·공백·\n·\r·\t 다섯 글자만). 측정: 출처 20개를 NBSP 로 이어 붙이면 콘솔은 `20 / 64개 · 490자`(상한 안)를 보여 주는데 서버는 1항목 509룬으로 읽어 300룬 상한으로 거절한다. U+00A0·U+3000·U+2028 모두 그 한 항목이 originOf 를 통과하므로 형식 오류로 걸러지지도 않는다. 고칠 곳은 `/[, \t\r\n]+/` 한 군데와 31행 주석, 그리고 유니코드 공백 입력을 넣은 회귀(현 테스트 55행은 Go·JS 가 일치하는 구분자만 써서 거짓을 통과시킨다).
- 구현 노트의 '확신 없는 곳 (2)' 가 정확히 이 자리였다. 진단만 빗나갔다 — trim() 이 아니라 split 이고, 오차는 '1~2' 가 아니라 항목 20대1·최장 25대509 다.
- 실제로 돌린 것: 상한 아홉 개 ↔ tracking.go 선언 일치, tsc -b·lint·build 깨끗, node 회귀 144건 통과, 그리고 Go 상수를 300→299 로 일시 변조해 **가드가 실제로 실패함**을 확인(되돌린 뒤 트리 깨끗). 이 동기화 가드 자체는 좋다.
- 못 본 것: 브라우저 DOM·실제 관리자 HTTP 저장(세션·CSRF 필요), Go live/DB·클러스터. maxLength 가 UTF-16 로 더 엄한 방향이라는 주석의 주장은 추론으로 확인했고 반례는 없다.
- 승인 시에도 남는 우려(다음 회차용): `MomentoEnvironment` 는 아직 안내 없는 상한이고 그 32는 룬이 아니라 **바이트**(한국어 10자에서 걸린다) — 테스트 이름 '안내하지 않는 상한이 남아 있지 않다' 는 그만큼 과하게 약속한다. 또 신규 테스트는 CI 밖이라(test:sso 하나뿐) 동기화 보호가 로컬 실행에만 의존한다.
- [러너 03:44] review rejected — 리뷰 거절: web/src/pages/trackingLimits.ts:36 `list.split(/[,\s]+/)` 는 서버의 splitHosts 가 아니다. Go 는 FieldsFunc 로 `,`·공백·\n·\r·\t 다섯 글자에서만 나누는데 JS `\

## 수리 노트
- 지적 세 건 전부 맞았다. 임시 Go 테스트로 splitHosts 를 직접 불러 재현했다: 출처 20개를 U+00A0·U+3000·U+2028 등으로 이으면 서버는 **1개 499룬 → 300룬 초과 거절**, 콘솔은 `20개/480자`(상한 안). 비평가가 말한 509 는 출처 길이만 다른 같은 현상이다.
- 고친 방법: `HOST_SEPARATOR = /[, \t\r\n]+/` 로 다섯 글자를 못박았다. 지적 범위를 한 칸 넘겨 같은 부류의 남은 어긋남도 고쳤다 — JS `trim()` 은 U+FEFF 를 떼고 U+0085 를 남기는데 Go `TrimSpace`(unicode.IsSpace) 는 반대라, U+FEFF 가 붙은 300룬 항목을 콘솔이 '상한 안' 으로 안내하고 서버가 301 로 거절한다. `HOST_TRIM` 에 IsSpace 집합을 직접 적었다. 31행 주석은 이제 참이다.
- 테스트는 Go 소스의 splitHosts 본문에서 `letter == '…'` 를 파싱해 구분자가 다섯 뿐임을 못박고(Go 가 하나 더하면 실패), 그 밖의 유니코드 공백 9종과 TrimSpace 차이 4건을 단언한다. 13개 기대값은 전부 실제 splitHosts 출력에서 측정했다. 역전 검증: 옛 split 으로 되돌리면 2건, trim 만 되돌리면 1건 실패. 단언을 느슨하게 한 곳 없음(14→17건).
- 돌린 것: `node --test scripts/*.test.mjs` 147/147, `npm run lint`, `npm run build`, `go test ./internal/tracking ./internal/api` 모두 통과. 임시 Go 테스트는 삭제했고 트리에 두 파일만 남았다.
- 여전히 확신 없는 곳: 브라우저 DOM·실제 관리자 HTTP 저장은 이번에도 못 봤다. 앞 회차가 남긴 두 우려(`MomentoEnvironment` 32는 룬이 아니라 바이트이고 안내가 없다 / 새 회귀가 CI 의 `test:sso` 밖이라 로컬 실행에만 의존한다)는 이번 지적 범위가 아니어서 손대지 않았다 — 다음 회차 몫이다.
- [러너 03:49] repair done — 비평 세 건 모두 맞았다. `allowedHostsUsage` 가 `/[,\s]+/` 로 나눠, 서버의 splitHosts(FieldsFunc, `,`·` `·`\n`·`\r`·`\t` 다섯 글자)가 안 나누는 유니코드 공백에서 �

## 비평 노트
- 승인. 수리가 고친 자리를 직접 재검했다: Go `splitHosts` 를 임시 테스트로 호출해 얻은 기준값과 `allowedHostsUsage` 를 유니코드 공백 17종·슬래시·4바이트 문자를 섞은 무작위 입력 **4000건**으로 대조했고 (항목수, 룬수) 불일치 0건. 임시 파일은 지웠고 트리는 깨끗하다.
- 테스트가 통과만 하는 것이 아님도 확인: HOST_SEPARATOR 를 `/[,\s]+/` 로 되돌리면 2건, HOST_TRIM 을 `trim()` 으로 되돌리면 1건, Go 의 MaxAllowedHostRunes 를 299 로 바꾸면 1건 실패한다. 직접 돌린 것 — tsc -b·lint·build·`node --test scripts/*.test.mjs` 147/147.
- 승인이어도 남는 우려 ①: `maxLength` 는 상한을 넘는 **붙여넣기를 말없이 자른다** — "과대 입력은 잘라 저장하지 않고 거절한다" 는 관례와 방향이 어긋나는 유일한 자리다. 200룬 id·1024룬 URL 은 실사용에서 닿지 않아 거절로 보지 않았으나 릴리즈 노트에 적을 값은 된다. 목록 textarea 에 maxLength 를 안 붙인 판단은 옳다.
- 우려 ②(앞 회차와 같음, 아직 미해결): 테스트 이름 '안내하지 않는 상한이 남아 있지 않다' 는 실제 단언보다 넓게 약속한다 — `MomentoEnvironment` 의 32(tracking.go:279)는 여전히 안내도 maxLength 도 없고 룬이 아니라 바이트다. ③ 이 회귀는 CI 밖이라(ci.yaml 은 test:sso 하나) Go 상수를 고치는 회차가 로컬 실행을 빠뜨리면 콘솔의 거짓 숫자가 조용히 머지된다.
- 못 본 것: 브라우저 DOM, 관리자 세션+CSRF 를 쓴 실제 HTTP 저장, Go live/DB·클러스터. 보안·법무 차단 소견 없음(새 엔드포인트·인가 경로·비밀값·개인정보 흐름 없음, 서버 Validate 가 그대로 정본). web/ 는 runtime-images.json sourcePaths 밖이라 BASE_VERSION 상향 불필요.
- [러너 03:54] review approved — 리뷰 승인 (risk=low)
- [러너 03:54] pr created — https://github.com/hkjang/AgentHub/pull/44
- [러너 03:56] ci passed — 검사 1개 모두 success
- [러너 03:56] merge done — 3097118
- [러너 04:04] release published — v0.261.0
- [러너 04:10] assets verified — v0.261.0 자산 8개 (이전 v0.260.0: 8)
