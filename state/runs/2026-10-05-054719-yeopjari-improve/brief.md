- 과제: 사진 정제의 EXIF·GPS 제거를 실제 JPEG 회귀 테스트로 보장 (가치 4 / 위험 1 / 작업량 M)
- 왜: README는 사진에서 촬영 위치가 제거된다고 약속하지만 `packages/core/src/media/image.ts:sanitise`의 메타데이터 제거를 검증하는 테스트가 없고 `scripts/e2e.mjs`의 사진 검사도 업로드·크기·형식에 한정된다. 실제 GPS 메타데이터가 있는 JPEG를 정제한 뒤 본문과 썸네일을 검사하면 원본 재사용이나 코덱 변경이 개인정보를 다시 남기는 회귀를 잡을 수 있다.
- 수용 기준:
  1) 테스트가 작은 정상 JPEG와 유효한 EXIF APP1(TIFF GPS IFD, 합성 위도·경도), 식별 가능한 COM 주석을 구성한다. 정제 전 입력이 실제 `jpeg-js.decode`로 디코딩되고 EXIF/GPS 구조와 주석이 존재함을 먼저 assert하여 빈 fixture로 거짓 통과하지 않게 한다. 실사용자 사진·좌표는 쓰지 않는다.
  2) 실제 `sanitise(input, limits)`를 호출하여 full·thumb 양쪽을 검증한다. 정상 JPEG로 다시 디코딩되고 예상 크기를 가지며 EXIF APP1과 COM 주석이 없어야 한다. JPEG 세그먼트 길이에 따라 SOS 이전 헤더를 검사하고, 단순 바이트 문자열 검색이나 소스 grep만으로 EXIF 제거를 증명하지 않는다.
  3) 같은 32×16 입력에 `maxLongEdge=32`(축소 없음)와 `maxLongEdge=16`(축소 있음)을 각각 적용한다. full은 각각 32×16·16×8, `thumbLongEdge=8`인 thumb는 둘 다 8×4이며 두 분기 모두 메타데이터 제거 조건을 만족해야 한다. 픽셀의 JPEG 손실 압축 오차나 인코딩 결과 전체 바이트를 스냅샷으로 고정하지 않는다.
- 건드릴 파일: `packages/core/src/__tests__/image.test.ts`(신규): 실제 `sanitise`와 `jpeg-js.encode/decode`를 호출하는 테스트, 작은 합성 JPEG/EXIF 생성 및 헤더 확인 helper를 같은 파일에 둔다. 기존 `packages/core/src/media/image.ts:sanitise/encodeJpeg/fitWithin/resize`와 `packages/core/src/routes/media.ts:mediaRoutes`는 읽기 참고이며 변경 대상이 아니다. 프로덕션 변경 0개, 테스트 파일 1개.
- 검증 명령: 저장소 루트에서 의존성이 없으면 `npm ci`, 이어서 `npm test -- packages/core/src/__tests__/image.test.ts`, 마지막으로 `npm run check`(lint → 세 workspace typecheck → 전체 Vitest). Node >=22. 새 테스트 파일은 아직 없으므로 대상 명령은 구현 후 실행한다. 현재 정찰의 `npm test`는 exit 127 / `vitest: not found`였으며 전체 테스트·lint·typecheck 통과 여부는 미확인이다.
- 위험과 피할 것: auth/session, db/migrations, .github/workflows, 빌드·릴리즈·의존성·API 계약은 범위 밖이다. DB/Ctx/코덱을 대역으로 바꾸지 말고 실제 공개 함수와 실제 JPEG를 쓴다. `jpeg-js`는 런타임에 EXIF 버퍼를 노출하지만 확인한 `.d.ts`에는 해당 속성이 없다. 숨은 타입에 의존하거나 any를 붙이는 대신 JPEG 헤더를 확인하고, 입력 생성도 `encode`의 문서화된 픽셀·comments 입력 + SOI 다음 APP1 삽입을 이용한다. 모든 APP 마커를 금지하면 정상 JFIF APP0까지 거부하므로 EXIF APP1과 COM을 구별한다. `targetBytes`는 목표치이지 엄격한 상한이 아니고 품질 루프 정책 변경은 이번 범위가 아니다. 실제 오류를 발견하면 원인과 과제 범위를 먼저 갱신하고 조용히 프로덕션 수정을 추가하지 않는다.
- 차선 후보: Node 정적 파일 서빙의 정상·거부 경로 회귀 테스트 (가치 3 / 위험 1 / 작업량 S) — `apps/server/src/static.ts:StaticFiles.serve/shell/resolve`를 실제 Request와 임시 파일로 호출한다. 신규 `apps/server/src/__tests__/static.test.ts` 한 파일에서 HEAD 무본문, index/SPA의 no-store·보안 헤더, 잘못된 인코딩·경로 입력에 외부 파일 내용이 노출되지 않음을 검사한다. 경로 입력은 `serve(request, pathname)`의 두 번째 인자로도 전달하여 URL의 사전 정규화로 시험할 입력이 사라지지 않게 한다. 현재 거부 시 SPA shell로 돌아가는 계약을 400/404로 바꾸지 않는다. 검증은 `npm test -- apps/server/src/__tests__/static.test.ts`와 `npm run check`.

범위·판단 근거
- 기준 커밋은 `32869c9`. 저장소에 CLAUDE.md/AGENTS.md 및 별도 ROADMAP/TODO 파일은 찾지 못했고, 주요 소스 TODO/FIXME 검색 결과는 없었다. 내부 설계와 남은 일은 `engineering/{architecture,security,performance,design}.md`, 공개 가이드는 `docs/`에 있다.
- `mediaRoutes`는 `sanitise` 반환값 full·thumb를 각각 저장한다(같은 프라이버시 보장이 두 출력 모두에 필요). 이 회차의 증명 범위는 저장 직전 실제 정제 함수이며 HTTP 업로드/DB/저장소까지의 E2E 보장이라고 주장하지 않는다.
- 정찰의 읽기 전용 실행: 현재 작업 트리 `image.ts`를 별도 체크아웃 `/mnt/c/Users/USER/projects/yeopjari/node_modules`의 esbuild로 메모리에서 번들하여 실제 jpeg-js와 실행했다(write:false). EXIF TIFF의 GPS IFD 포인터와 위도/경도 RATIONAL을 넣은 32×16 JPEG를 사용했다. 입력 EXIF 버퍼 129 bytes 및 주석 1개를 확인했고, 두 maxLongEdge 분기의 full·thumb 모두 정상 디코딩 및 EXIF/주석 없음, 위 수용 기준의 크기를 확인했다. 이 실행은 Vitest 통과나 HTTP E2E를 대신하지 않는다. 별도 체크아웃의 정확한 잠금파일 일치는 미확인이므로 구현자는 이 저장소의 `npm ci`로 재검증한다.
- `scripts/e2e.mjs:1090~1216`에는 선등록·최소 표본·사분위수·목록/상세 우회 차단·철회 후 재잠금 테스트가 이미 있다. 기존 연봉 후보를 중복 구현하지 않는다. 반면 같은 파일 `salaryLeak/visibleRows` 검사는 쿼리 오류와 파싱 실패의 undefined도 성공 처리하므로 별도 신규 후보로 남긴다(DB 필요).
- 이전 쿠키 회차 성공 기록과 달리 이 기준 커밋의 `readCookies`에는 보호 없는 decodeURIComponent가 남아 있다. 머지 여부는 미확인이고 이전 과제를 다시 선택하거나 이번 과제에 섞지 않는다.

접근 비교와 선택
- 선택: 코어 실제 함수에 작은 합성 JPEG를 넣는 Vitest 테스트. 새 의존성·DB 없이 두 출력과 두 분기를 고정하고 1파일에 끝난다. 가장 중요한 전제는 유효한 GPS fixture를 테스트 스스로 입증할 수 있다는 것인데 정찰 실행으로 구성 가능성을 확인했다.
- 대안: 기존 e2e.mjs에서 업로드 후 원본/썸네일 다운로드까지 검사. 배선 보장은 넓지만 서버·DB·회원/매물 권한 준비가 필요하므로 이번 45분 회차보다 후속 회차에 적합하다.
- 현상 유지: 현재 정제 동작은 정상이고 코드 수정은 필요 없다. 다만 README의 프라이버시 약속을 검증하지 못하므로 회귀 테스트를 추가하는 쪽을 선택했다. 큰 메타데이터 라이브러리 도입·이미지 파이프라인 개편은 요구 범위를 넘는다.

구현 순서와 확인 지점 (현재 모두 미착수, 사람 확인 없이 실행 가능)
1. `image.test.ts`에 작은 JPEG 및 합성 EXIF/GPS/COM fixture와 fixture 유효성 테스트를 작성한다. `npm test -- packages/core/src/__tests__/image.test.ts` 통과 후 진행한다. 체크포인트: 입력에 실제 메타데이터가 있고 decode가 성공해야 한다.
2. 같은 파일에 축소 유무 × full/thumb 결과 검사를 추가한다. 같은 대상 테스트 명령으로 검증한다. 체크포인트: 메타데이터를 지우지 않은 입력을 검사 helper에 넣으면 거부되는 대조 assertion이 있어야 한다. 프로덕션 파일을 임시로 바꾸는 방식은 필수가 아니다.
3. `npm run check`로 기존 테스트·타입·린트를 확인한다. 실패하면 이번 테스트의 원인과 기존 환경 문제를 구분하여 기록하고, 실제 명령/결과를 구현 노트에 남긴다. 별도 CI/릴리즈 변경이나 외부 서비스 호출은 하지 않는다.

작업량 근거와 예비 시간
- 방법: 파일·함수 확인을 바탕으로 한 bottom-up 추정. fixture/입력 입증 10분 + 두 분기/두 출력 테스트 12분 + 검증/기록 8분 = 기본 30분.
- 알려진 불확실성 예비: APP1/TIFF 오프셋·엄격한 타입 처리 5~10분, 의존성 설치 지연 0~5분. 총 35~45분을 중간 확신의 범위로 본다(실측 확률로 보정한 수치는 아님). 일반 개발이 이미 각 항목에 포함되어 있어 중복 여유를 더하지 않는다.
- 유사 회차는 쿠키 테스트 추가 1파일 사례지만 소요 시간 기록이 없어 독립된 시간 추정으로 사용하지 않았다. 새 라이브러리/DB가 필요해지면 이 추정은 무효이며 위 차선 후보로 전환하거나 과제 범위를 재평가한다. 미지의 범위 확대용 관리 예비는 0분이며 기능 확대를 승인한 것으로 해석하지 않는다.
- 적용 스킬: 로컬 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`의 SKILL.md를 읽고 적용했다. Skill 호출 도구가 없어 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/`에서 직접 읽었다. 추정 스킬의 `references/sources.md`도 확인했으며 외부 원문에 기반한 확률/비용 주장은 하지 않았다.
