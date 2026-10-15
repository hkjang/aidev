## 2026-10-15
- 선택: MP4 판독에 회전 행렬 반영(세로 촬영 영상의 표시 해상도) (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `src/worker/lib/mp4.ts` 가 tkhd 의 변환 행렬을 무시하고 저장된 너비·높이를 그대로 써서, 90°·270° 회전이 붙은 세로 영상(휴대전화 촬영)을 1920×1080 으로 판독했다. 그 결과 `spec.ts` 의 16:9·해상도 검사가 세로 영상을 통과시키고(AT-19) 관리자·고객 화면과 작품 올리기 스크립트에도 틀린 해상도가 떴다. 행렬의 a·b 를 부호 있는 16.16 고정소수점으로 읽어 90°·270° 면 너비·높이를 바꿔 돌려주도록 좁게 고쳤고(프로덕션 파일 1개), 단위 시험과 업로드 HTTP 경로 시험을 더했다. 검증은 `npm run build`(빌드 점검 통과)와 `npm test` 78개 전부 통과(종전 76 + 신규 2).
- 실패 재현: `✘ [unit] MP4 판독: 90·270° 회전된 트랙은 표시 해상도(세로)로 읽어 규격 검사에서 걸린다(AT-19)` — `- "height": 1920, - "width": 1080 / + "height": 1080, + "width": 1920`. 고친 코드를 되돌린 뒤 업로드 경로 시험도 같은 값으로 실패했다: `✘ [api] tests/api-files.spec.ts:66 회전 행렬이 붙은 세로 영상은 표시 해상도(1080×1920)로 판독한다(AT-19)`.
- 보류 아이디어: ① `topLevel` 의 최상위 상자 64개 제한 — 상자가 더 많은 파일에서 moov 를 못 찾고 "영상 정보 없음" 으로 거절 ② mvhd 길이만 쓰므로 트랙(mdhd)이 더 긴 파일의 길이 판정이 어긋날 수 있음 ③ `scripts/portfolio-import.mjs` 에 자동 시험 없음(드라이런 모드 + 단위 시험) ④ `seo.ts`·`shell.html` 짝 — 해당 없는 메타의 `element.remove()` 와 `noscript#seo` 이스케이프를 고정하는 시험 ⑤ `parseRange` 경계(`bytes=0-0`, 끝 초과, 여러 범위) 시험 보강
- 과제서: 기각 — 정찰이 예산 초과로 중단되어 과제서가 없었고, 구현자가 직접 골랐다.

## 2026-10-15
- 선택: GitHub Actions CI 워크플로 추가 — 저장소의 검증 명령을 그대로 CI 로 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 저장소에 워크플로 파일이 하나도 없어 러너 PR 이 전부 'CI 검사 없음' 으로 막혔다. 노트북·러너가 쓰는 관문(`npm ci` → `npm run build` → `npx playwright install --with-deps chromium` → `npm test`)을 그대로 `.github/workflows/ci.yml` 한 파일로 옮겼다. push(main)·pull_request 에서 돌고, Node 22(engines `>=22.12`)·npm 캐시를 쓰며, 비밀 값은 쓰지 않는다(시험 설정은 저장소 안 `tests/worker-test.env`, D1·R2 는 `wrangler dev` 로컬, 메일은 가짜 Resend). 검증은 이 워크트리에서 같은 순서를 실제로 돌려서 했다 — `npm run build` 통과("빌드 점검 통과"), `npm test` **76개 전부 통과(57.7s)**, 워크플로 YAML 은 `yaml.safe_load` 로 파싱 확인. 프로덕션 코드·스크립트·문서는 건드리지 않았다(변경 파일 1개).
- 실패 재현: 해당 없음 — 테스트를 더하거나 고치는 회차가 아니라 지금 통과하는 명령을 CI 로 옮기는 작업이다(명세가 새 테스트 작성을 금지). 대신 CI 가 돌릴 명령을 이 워크트리에서 그대로 실행해 통과를 확인했다: `npm run build` → "빌드 점검 통과", `npm test` → "76 passed (57.7s)".
- 보류 아이디어: ① `scripts/portfolio-import.mjs` 드라이런 + 자동 시험(수동 확인만 있음) ② `seo.ts`·`shell.html` 짝 고정 시험(빈 메타 `element.remove()` 와 `noscript#seo` 이스케이프) ③ `parseRange` 경계 시험 보강(`bytes=0-0`, 끝 초과, 여러 범위, 음수) ④ 길이 판정을 mvhd 외에 트랙 mdhd 와 교차 확인 ⑤ `topLevel` 의 최상위 상자 64개 제한 ⑥ (신규) CI 에 `npm run check` 를 build 와 별도 단계로 두어 타입 오류를 빌드 전에 빠르게 드러내기 ⑦ (신규) CI 에 의존성 감사(`npm audit --omit=dev`)를 경고 단계로 추가
- 과제서: 채택 — 요청된 작업(이슈 #0)을 그대로 했다. 파일 하나(.github/workflows/ci.yml)만 바꿨다.

## 2026-10-15
- 선택: 해석되지 않는 Range 헤더에 416 대신 전체(200)로 응답 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `src/worker/services/parseRange()` 가 `/^bytes=(\d*)-(\d*)$/` 에 걸리지 않는 모든 Range 를 `'invalid'` 로 돌려주고 `serveObject()` 가 그것을 416 으로 내보내, 여러 구간(`bytes=0-1,5-6`)·모르는 단위(`items=0-5`)·깨진 값을 보내는 클라이언트는 공개 작품 영상·홈 대표 영상·시안·최종본의 본문을 하나도 받지 못했다. 해석 불가는 헤더가 없는 것처럼 `null`(=200 전체)로 분류하고, `'invalid'`(416)는 문법이 맞는데 만족시킬 수 없는 경우(start>end · start>=size)만 남기도록 한 줄을 좁게 바꿨다(프로덕션 1파일, 반환 타입·`serveObject` 헤더 조립은 그대로). 검증은 `npm run build`("빌드 점검 통과")와 `npm test` **76개 전부 통과(58.3s)**, 그리고 고치기 전에 단위·HTTP 시험이 416 으로 실패하는 것을 먼저 확인했다.
- 실패 재현: `✘ [unit] tests/unit-media.spec.ts:60 범위 요청 해석 — ... Expected: null / Received: "invalid"` (`parseRange('bytes=0-1,5-6', 1000)`), 그리고 프로덕션 배선을 지나는 HTTP 시험 `✘ [api] tests/api-portfolio.spec.ts:25 ...(AT-03) Expected: 200 / Received: 416` (공개 영상 `mine.videoUrl` + `Range: bytes=0-1,5-6`).
- 보류 아이디어: ① `seo.ts`·`shell.html` 짝 고정 시험(private 경로 noindex · canonical · `noscript#seo` 의 `esc()`) ② `scripts/portfolio-import.mjs` 드라이런 + 자동 시험 ③ 길이 판정을 mvhd 외에 트랙 mdhd 와 교차 확인 ④ `topLevel` 의 최상위 상자 64개 제한 ⑤ 공유 미리보기용 `twitter:card`·`og:image:alt`(shell.html·seo.ts·verify-build 3파일이 함께 걸린다)
- 과제서: 채택 — 과제서의 근거가 지금 코드와 정확히 맞았고(파일·행 번호 포함), 수용 기준 1~4 를 그대로 구현·검증했다. 다만 종전 통과 수는 78 이 아니라 **76** 이었다(이 base `b00d53c` 에 MP4 회전 회차가 들어 있지 않다).

## 2026-10-15
- 선택: Cron 보관 만료 안내가 이미 만료된 프로젝트에 영구히 막힌다 — 안내 조회에 하한(retentionUntil > now) 추가 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `src/worker/services/cron.ts` 의 `retention()` 이 안내 대상을 상한(`retentionUntil < now + 7일`)만으로 골라 이미 만료된 프로젝트까지 담고, 루프에서 건너뛴 그 프로젝트들은 `retentionNoticeAt` 이 NULL 로 남아 다음 Cron 에서도 같은 조회에 다시 잡혔다 — 그런 찌꺼기가 20건 쌓이면 `limit(20)` 창이 전부 막혀 7일 뒤 만료될 프로젝트가 안내를 못 받고 파일이 삭제됐다. 조회에 `gt(retentionUntil, now)` 하한을 더해(운영 화면의 같은 조회 `routes/admin.ts:50` 이 이미 쓰는 형태) 창을 되찾았고, 루프의 `p.retentionUntil < now` 가드는 이중 방어로 남겼다. 삭제 시점·보관 정책은 건드리지 않았다(프로덕션 파일 1개 + 시험 1개). 검증은 `npm run build`("빌드 점검 통과")와 `npm test` **77개 전부 통과(1.2m)** — 종전 기준선 76 + 신규 1.
- 실패 재현: `✘ 2 [api] › tests/api-retention.spec.ts:59:1 › 이미 만료된 프로젝트가 안내 창(20건)을 넘겨도 7일 안에 만료될 프로젝트는 안내를 받는다(AT-24) (36.7s)` — `Error: 메일 없음: starved-mv9e1zr0f9a0@customer.test /보관 만료 안내/ (at api.ts:43)`. 같은 시험이 고친 뒤 21.7s 로 통과한다.
- 보류 아이디어: ① `seo.ts`·`shell.html` 짝 고정 시험(private 경로 noindex·canonical·`noscript#seo` 의 `esc()`) ② `scripts/portfolio-import.mjs` 드라이런 + 자동 시험 ③ `admin.ts:50` 이 실제 하루(`7 * DAY`)를 쓰고 Cron 은 `dayMs(env)` 를 쓰는 단위 불일치(운영에서는 같아 영향 없음, 시험 서버에서만 창이 다르다) ④ `retention()` 의 `soon`·`expired` 조회에 `orderBy(asc(retentionUntil))` — 창이 꽉 찼을 때의 우선순위를 정한다 ⑤ 길이 판정을 mvhd 외에 트랙 mdhd 와 교차 확인
- 과제서: 채택 — 과제서의 근거(파일·행 번호·수용 기준)가 지금 코드와 정확히 맞았고, 지시한 변경(하한 추가, 가드 유지, `admin.ts` 미수정)과 시험을 그대로 구현·검증했다. 다만 `cron()` 헬퍼는 `POST /__cron` 이 아니라 `GET /cdn-cgi/handler/scheduled` 이고, 찌꺼기 21건은 고객 1명으로 만들어(로그인 1회) 신규 시험 1개가 21.7s 로 끝난다.

