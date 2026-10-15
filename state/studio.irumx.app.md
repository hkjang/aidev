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

