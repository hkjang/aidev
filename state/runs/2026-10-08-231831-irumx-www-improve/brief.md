- 과제: 서비스 HTML의 사이트맵 누락을 빌드 검사에서 동적으로 감지 (가치 3 / 위험 1 / 작업량 S)
- 왜: scripts/verify-build.mjs는 사이트맵의 17개 고정 주소만 검사하여 목록 밖의 새 서비스 HTML이 사이트맵에서 빠져도 성공한다. 실제로 생성된 서비스 HTML에서 검사 대상을 추가하면 서비스 추가 시 수동 목록 갱신을 빠뜨려도 누락을 발견한다.
- 수용 기준: 1) 기존 17개 필수 주소 및 404 배제 검사를 그대로 유지하고, dist/services/<slug>.html마다 https://www.irumx.app/services/<slug>의 정확한 loc를 요구한다(확장자·끝 슬래시 없음). 빠진 주소를 포함한 한국어 오류와 exit 1을 반환한다. 2) 현재 정상 빌드와 모든 서비스 loc가 있는 경우는 통과한다. 비서비스 HTML·이미지는 새 검사 대상에 넣지 않는다. 3) 회귀 테스트가 실제 verify-build.mjs를 자식 프로세스로 실행하여 ‘고정 목록에 없는 추가 서비스 HTML + loc 없음’만 실패하고 해당 loc 추가 후 성공함을 증명한다. 기존 필수 주소 /about 누락과 404 포함도 계속 실패해야 한다.
- 건드릴 파일: scripts/verify-build.mjs: walk 결과인 htmlFiles 및 robots.txt·sitemap 검사 블록(현재 약 216~225행) — relative(dist, f)를 슬래시 경로로 정규화하고 /^services\/[^/]+\.html$/에 해당하는 파일의 URL을 추가 검사; tests/verify-build-sitemap.test.mjs(신규) — node:test로 실제 빌드 검사 CLI 회귀. 프로덕션 파일 1개, 테스트 1개로 제한한다. 새 의존성·package.json 변경은 필요 없다.
- 검증 명령: 사전 준비 npm ci → npm run build. 이후 node --check scripts/verify-build.mjs → node --test tests/verify-build-sitemap.test.mjs → node scripts/verify-build.mjs. 최종 전체 확인은 npm run test:build → npx playwright install chromium → npm test --silent(기존 러너 명령은 그대로 유지). Node v22.23.1/npm 10.9.8에서 정찰의 구문 검사와 아래 재현 명령은 실제 실행했다. 새 테스트 명령은 파일 구현 후 실행하며, npm ci/build/Playwright는 이번 정찰에서 실행하지 않았다.
- 위험과 피할 것: tests/site.spec.ts의 PAGES와 서비스 콘텐츠를 통합하거나 고정 필수 목록을 삭제하지 않는다. Worker·문의·auth·migrations·workflows·배포·의존성·포트·pretest를 같이 고치지 않는다. 과거 반복 회차를 체리픽하지 않는다. 현재 sitemap-0.xml 단일 파일 및 Astro build.format:file 계약에 맞춘 작은 검사만 추가한다. XML 파서 도입·사이트맵 분할 지원·모든 페이지 색인 정책 변경은 범위 밖이다.
- 차선 후보: README·docs/deploy.md의 Worker 라우팅과 MAIL 관문 설명 정합성(가치 2 / 위험 1 / S) — 구현 시작 시 1순위 검사가 이미 추가되어 있을 때만 사용. README 구성 표와 docs/deploy.md 구조 초반의 ‘/api만 Worker’는 실제 serveHtml 및 run_worker_first와 다르다.

실제 확인한 근거와 한계

- HEAD 94eb01f. 최근 git log -30은 전체 18건을 반환했다. 최신 커밋은 MAIL 관문 도입, 직전은 HTML gzip이다. 작업 트리는 깨끗하고 의존성·dist·dist-draft·dist-email이 없다.
- src/pages/services/[slug].astro:getStaticPaths는 PUBLISHED_SERVICES.map으로 경로를 만든다. src/content/services.ts:PUBLISHED_SERVICES는 published 필터를 적용한다. astro.config.mjs는 format:file/trailingSlash:never이며 sitemap에서 404를 제외한다.
- scripts/verify-build.mjs:walk는 실제 HTML을 이미 수집한다. 동일 파일의 FAQPage 검사도 /^services\/[^/]+\.html$/을 사용한다. 신규 검사는 그 범위를 따른다.
- 저장소 루트에서 `node --test /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-231831-irumx-www-improve/assets/sitemap-gap.test.mjs` 실행: 2 통과·1 실패(exit 1). 실제 사이트맵 검사 블록을 읽어 VM에서 실행했고, 메모리상 htmlFiles에 /virtual/dist/services/scout-future.html을 추가해도 failures=[]였다. /about 누락은 감지한다. 이 파일은 정찰 증거이며 소스 문자열 추출 방식 자체를 영구 테스트로 복사하지 말 것.
- 실제 생성물의 사이트맵 누락이나 운영 장애는 미확인이다. 이번 과제는 재현된 검증 공백 보강이다.
- package.json에 pretest가 없고 Playwright 서버는 8820~8823 고정·재사용 금지다. 전체 시험 준비물을 먼저 만든다. 포트 점유·Turnstile 외부 연결 실패는 기존 환경 문제로 기록하고 이 과제에 끌어들이지 않는다. harfbuzzjs/fontverter 직접 의존성 미선언 역시 별도 보류 사항이다.

구현 순서와 검토 지점(모든 단계 미착수)

1. npm ci 및 npm run build로 기준 산출물을 확인한다. 증명: 기존 빌드 exit 0. 검토: 구현자가 결과를 기록하고 진행하며 사람 승인 대기는 없다. 기반 빌드 실패 시 먼저 원인을 기록하고 검사 기능 결함과 구분한다.
2. tests/verify-build-sitemap.test.mjs에 CLI 회귀를 작성한다. node:test에서 임시 루트를 만들고 scripts/verify-build.mjs와 dist를 복사하며, 설치된 node_modules를 임시 루트에서 참조하게 한다(이 Linux 환경에서는 디렉터리 심볼릭 링크 가능). 실제 스크립트가 import.meta.url 기준으로 루트를 잡는 점을 지킨다. dist/services/yeopjari.html을 같은 폴더의 고정 목록 밖 이름 scout-future.html로 복사하여 기존 HTML·글꼴 검사를 통과하는 fixture로 쓴다. 임시 파일의 canonical/og:url 등 해당 서비스 URL도 새 경로에 맞춘다. 정상 → 새 HTML만 추가(실패해야 함) → loc 추가(성공) → /about loc 제거(실패) → 복원 후 404 loc 추가(실패)를 독립 fixture 또는 순차 복원으로 검사한다. 임시 폴더는 finally로 정리하고 원래 dist는 수정하지 않는다. 증명: node --test tests/verify-build-sitemap.test.mjs에서 수정 전 추가 서비스 누락 단정만 실패. 검토: 의도한 이유의 실패인지 stderr를 확인한다.
3. verify-build.mjs에 동적 주소 검사만 추가한다. 기존 sitemap 문자열의 정확한 <loc> 비교 관례를 재사용해 더 큰 파서 변경을 피한다. 증명: node --check scripts/verify-build.mjs 및 node --test tests/verify-build-sitemap.test.mjs 통과. 검토: 새 루프를 잠시 제거했을 때 추가 서비스 누락 단정이 다시 실패하는지 확인 후 복원한다.
4. node scripts/verify-build.mjs로 원래 dist가 정상인지 확인하고, 위 최종 전체 명령을 수행해 결과를 남긴다. 검토: 기존 테스트 기대값/skip/timeout을 변경하지 않았는지 diff로 확인하고 다음 비평 단계에 넘긴다.

접근 대안과 선택

- 선택: 기존 htmlFiles를 이용한 동적 추가 검사. 파일 추가 때 손으로 목록을 맞추는 부담 없이 현재 필수 경로 보장도 남긴다.
- 수동 목록만 갱신: 지금 운영 누락은 확인되지 않아 당장 무변경도 가능하지만 다음 서비스 추가의 공백을 없애지 못한다.
- 콘텐츠·Playwright·검증 목록을 공통 manifest로 통합: 향후 규모가 커지면 검토할 수 있으나 이번에는 다수 파일과 런타임 결합을 늘린다.
- 핵심 전제: services 폴더의 직계 HTML은 모두 공개 서비스 상세 페이지다. getStaticPaths/published 필터에서 현재 확인했다. noindex 전용 서비스 파일이나 분할 사이트맵이 발견되면 범위를 다시 기록하고 무리하게 일반화하지 않는다.

작업량 근거와 예비 시간

- 상향식 추정(정찰자의 판단, 실측 아님): 기준 준비 5~8분, fixture/회귀 10~12분, 검사 추가 3~5분, 검증/기록 7~10분 = 기본 25~35분. 알려진 변동(콜드 설치·빌드 및 fixture 준비)에 별도 contingency 5분을 두어 30~40분을 예상한다. 신뢰도 중간이며 통계적 확률이나 완료 약속이 아니다.
- 교차 확인: 과거 동일 저장소 유형 검증 회차는 전체 Playwright 31.8초, 다른 회차는 콜드 빌드 수분 이상을 보고했다. 전체 45분 추정치를 직접 뒷받침하는 유사 과제 실측은 없어 정밀한 비율 추정은 하지 않는다. 첫 빌드 후 다시 추정한다.
- 관리 예비 시간(management reserve)은 이번 과제에 배정하지 않는다. 새 범위 발견 시 끼워 넣지 않고 보류 기록으로 남긴다. 운영 배포·외부 API 검증·기반 테스트 환경 수리는 추정에서 제외했다.
- 적용 스킬: /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md. 전용 Skill 도구가 노출되지 않아 로컬 파일을 읽었다. pmo의 references/sources.md도 확인했으며 외부 비용 산정 수치/방법론 주장은 사용하지 않았다.
