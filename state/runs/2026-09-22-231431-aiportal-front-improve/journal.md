# 회차 노트 2026-09-22-231431-aiportal-front-improve — aiportal-front
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:14] base pinned — main@e938e8e
- [러너 23:14] autonomy release — 

## 정찰 노트
- 릴리즈 교착을 원문으로 재확인했다(release-prompt.md:21,22 는 없는 증가 패턴을 요구하고 :24 의 skipped 는 package.json:3 의 "0.0.0" 때문에 불가). 외부 파일은 허용 디렉터리 밖이라 읽기만 되고 고칠 수 없어, 같은 무변경 회차를 7번째 반복하는 대신 1순위에 진입 조건을 달고 저장소 안에서 실제로 고칠 수 있는 건을 기본 실행 대상으로 지정했다.
- globalLoading 을 골랐다: 원인(boolean 단일 상태)과 동시 실행 경로(ChatStorageList.vue:55/:90)를 실물로 확인했고, 순수 JS 라 SFC 플러그인 없는 현재 vitest 로 Red→Green 을 증명할 수 있다. manualChunks·Header 폴링·serviceCode 통일은 각각 빌드 설정 위험·계약 미확인·.vue 테스트 불가로 제쳤다.
- 추측으로 적은 것: 수용 기준 3의 loadingTxt 겹침 정책은 현재 동작(무조건 덮어쓰기)만 확인했을 뿐 기대 계약은 미확인이다. 110개 호출부 중 실제로 열어 본 것은 ChatStorageList.vue 와 Chat/Index.vue 뿐이고 나머지 27개 파일의 try/finally 짝은 미확인이다.
- 구현자가 조심할 것: floor-at-0 를 빼면 스피너가 영구히 안 꺼지는, 지금보다 나쁜 회귀가 난다. 모듈 전역 싱글톤이라 테스트 간 카운터 누수도 주의(vi.resetModules + 동적 import 권장).
- node_modules 가 없어 이번 회차에 npm ci/test/build 를 실행하지 않았다. Red 기준선은 구현자가 직접 잡아야 한다.
- [러너 23:18] scout done — [수정 과제] 릴리즈 실패 원인 고정 — 진입 조건이 충족되지 않으면 이번 회차 기본 실행 대상은 globalLoadi

## 구현 노트
- 과제서의 기본 실행 대상(globalLoading 참조 카운트)은 착수하지 않았다. 31개 호출 파일 전수 감사에서 과제서가 정한 중단 조건에 걸리는 누수 2건을 실물 확인했다: ChatStorageDetail.vue:886-902 openLawViewer 는 stopLoading 이 아예 없고, SupportOcr.vue:203-223 은 stopLoading 이 setTimeout 안에만 있어 catch 경로에서 누락된다. 카운터를 넣으면 0 복귀 불가 → 스피너 영구 유지(현재보다 나쁜 회귀)라 지정된 차선(문서 명령·링크 정정)을 구현했다.
- 확신 없는 곳: 감사는 파일별 start/stop 개수 + 이상치 2건 정독으로 했다. 개수가 같은 파일 안에서 start 와 stop 이 서로 다른 제어 흐름에 있는 경우(예: Chat/Main.vue 3 start / 4 stop)는 정독하지 않았으므로 누수가 더 있을 수 있다. 즉 "누수는 2건뿐"은 확인된 것이 아니다.
- docs/09 표의 12~14 테스트케이스 문서는 실제로 없어 링크를 지우고 "(미작성)" 평문으로 바꿨다. 원래 계획된 문서인지 폐기된 것인지는 확인하지 못했다.
- 일부러 하지 않은 것: docs/06:424 의 "GitLab CI 예제" 블록에 남은 npm run build:prod. 같은 블록이 없는 .env.prod 도 참조하는 가상 예시라 한 줄만 고치면 반쯤 실제인 설정이 된다. .vue 호출부는 과제서 지시대로 건드리지 않았고, 버전 파일·태그·docs/RELEASE.md 도 손대지 않았다(1순위 진입 조건 미충족 유지).
- 다음 역할이 조심할 것: 이번 변경은 문서 4개뿐이라 런타임 영향이 없다(npm test 19파일 424 통과, build:dev 통과 후 dist/ 삭제). globalLoading 참조 카운트를 다음에 넣으려면 위 누수 2건 수정이 반드시 선행되어야 하고, 그걸 증명하려면 vitest.config.js 에 @vitejs/plugin-vue 를 먼저 붙여야 한다(현재 .vue 테스트 불가).
- [러너 23:23] brief fallback — 차선 — 과제서가 명시한 중단 조건("`finally` 없이 start 하는 호출부를 하나라도 발견하면 범위를 줄여 보고하라")에 해당�
- [러너 23:23] verify passed — 검증 2개 통과 (auto)

## 비평 노트
- 확인함: diff 4개 문서 전부를 package.json scripts·docs/ 실파일 목록과 대조했다. build:core/ofc/dev/int·test 는 실존, build:prod·lint 는 부재, docs/07-개선사항-권장사항.md 와 docs/09 표의 10·11 대상 파일은 실존, 12~14 는 부재로 '(미작성)' 평문 처리가 맞다. README·docs 의 모든 상대 링크 목적지를 전수 대조했고 깨진 것은 없다.
- 구현자가 의심한 자리를 직접 시험했다: 중단 사유였던 globalLoading 누수 2건(ChatStorageDetail openLawViewer 에 stopLoading 없음, SupportOcr.vue:203-223 stopLoading 이 setTimeout 안에만 있어 catch 경로 누락)은 실물로 성립한다. 따라서 기본 과제 → 차선 전환은 정당한 판단이다.
- 못 본 것: npm ci/test/build 재실행(문서 전용이라 생략), 구현 노트의 '424 통과' 독립 재현, start/stop 개수가 같은 파일 내부의 제어 흐름(누수가 2건뿐이라는 보장은 여전히 없음).
- 승인이어도 남는 우려: (1) 커밋 메시지는 docs/06 의 build:prod 를 교체했다고 적었지만 CI 예제 블록 :424 에는 남아 있다 — 릴리즈 노트에는 '일부'로 적을 것. (2) docs/.ipynb_checkpoints/ 의 tracked stale 사본에 npm run build/lint/build:prod 오정보가 그대로 남아 있다.
- 보안·법무 차단 없음: 인증·식별자·비밀값·의존성·개인정보 경로를 건드리지 않는 마크다운 전용 변경이며 revert 가 무해하다.
- [러너 23:25] review approved — 리뷰 승인 (risk=low)
- [러너 23:25] pr created — https://github.com/hkjang/aiportal-front/pull/21
- [러너 23:25] ci passed — 검사 없음 — 정책으로 허용
- [러너 23:25] merge done — 0eb9a8d
- [러너 23:27] release failed — 릴리즈 안 함: 다음 버전·태그·릴리즈 노트 관례를 확정할 근거가 없어 릴리즈를 수행하지 못했다. HEAD 66e6086 에서 재조회한 실제 근거: git tag 0개(refs/t
