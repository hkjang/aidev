# 회차 노트 2026-09-20-185409-DartFly-improve — DartFly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:54] base pinned — main@0c256cc
- [러너 18:54] autonomy release — 

## 정찰 노트
- 선택: saved.js의 상세 선택/응답 경합 수정(4/1/M). Node 대역으로 이전 current/삭제 단추 잔존과 늦은 응답 덮어쓰기를 재현해 로그인 문구 개선·근거 약한 스모크 재시도보다 우선했다.
- 기존 12개 후보 유지·재평가(sso=none 거절 안내만 rejected), 신규 2개 추가. 전체 go test -race ./... 통과; 실제 브라우저·삭제 API 실행은 미확인.
- 구현 주의: 최신 요청 성공·실패 모두 검사하고 삭제 ID 고정/후속 선택 보존까지 검증. auth·migrations·workflows·목록 페이지네이션은 범위 밖.
- 요청된 회사 스킬 3개는 도구/로컬 검색에서 찾지 못해 적용 불가를 brief에 명시했다. 코드·커밋 변경 없이 지정 산출물만 작성했다.
- [러너 18:58] scout done — 저장 결과 상세의 늦은 응답과 이전 선택의 작업 단추가 남는 문제 수정 (가치 4 / 위험 1 / 작업량 M)

## 구현 노트
- 5cd5ca0: saved.js에 선택 즉시 상세 초기화·요청 순번 검사·선택 ID 기반 강조를 추가해 이전 결과 작업과 늦은 응답 반영을 막았습니다.
- 삭제 ID를 고정하고 중복 삭제를 막으며, 완료 시 같은 선택만 초기화하고 진행 중인 삭제 대상 조회를 무효화합니다. 후속 B 선택은 보존합니다.
- 회귀: 실제 saved.js 이벤트를 VM+DOM/API 대역으로 실행, 원본 8개 중 7개 실패→최종 11개 통과. 전체 race·vet·gofmt·스모크 빌드 통과.
- 브라우저: Chromium 설치 후 필수 스모크 31페이지·편집기 및 별도 /saved 클릭 경합 검증 통과. 초기 별도 대역의 data 포장 누락을 수정해 재검증했습니다(same run 폴더의 saved-browser.py/log, smoke-final.log).
- 확신 없는 곳·검증 못 한 것: 저장 API는 제어 가능한 대역으로 검증했으며 실제 저장 데이터를 삭제하는 종단 검증은 하지 않았습니다. technology:completion-verification/systematic-debugging/test-driven-development는 도구·로컬 목록에 없어 절차·반환 형식 미확인입니다.
- 일부러 제외: 서버 API·auth·session·migrations·workflows·layout.js 및 목록 페이지네이션은 이번 경합 수정에 불필요해 변경하지 않았습니다.
- 다음 역할 주의: Node 테스트는 외부 의존성 없음; 스모크는 Docker·Python Playwright·Chromium 필요. --keep으로 추가 클릭 검증 후 --down 정리 완료; 푸시·릴리즈 미실행.
- [러너 19:06] brief accepted — 채택 — 이전 current와 작업 단추 잔존·역순 응답 덮어쓰기가 현재 코드 및 구현 전 실행 테스트와 일치했습니다.
- [러너 19:06] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- approve / low / blocking 없음: 변경 2개 파일과 커밋, 상세 응답·삭제 경합, 서버 세션·소유자/관리자 조건, 임베드 경로를 확인했습니다.
- HEAD 회귀 11개 통과; main 대체 실행은 8개 실패·2개 취소·1개 통과. webui/resultsave Go 테스트와 diff 공백 검사 통과.
- 실제 DB 삭제 종단 검증·브라우저 스모크는 재실행하지 않았습니다. 릴리즈 시 이 검증 한계를 유지해 기록하세요.
- 요청한 회사 스킬 3개는 도구·로컬에서 찾지 못해 적용 불가; 프롬프트 기준 검토에서 실제 차단 결함은 찾지 못했습니다.
- [러너 19:07] review approved — 리뷰 승인 (risk=low)
- [러너 19:08] pr created — https://github.com/hkjang/DartFly/pull/10
- [러너 19:13] ci passed — 검사 3개 모두 success
- [러너 19:13] merge done — 5cd5ca0

## 릴리즈 노트
- v2.72.0: 기존 관례대로 머지 커밋 f09cfc0에 경량 태그 생성(별도 릴리즈 커밋·버전 파일 수정 없음).
- Go race·vet·gofmt·build, JS 테스트, Chromium 31페이지·편집기 스모크, 배포 이미지 실기동 검사 통과. 브라우저 캐시 경로 수정 후 재검증.
- 기존 deploy/build-release.sh로 tar.gz·sha256 생성 및 무결성 확인. release.json과 한국어 release-notes.md 작성. 원격 푸시·업로드 없음.
- 요청한 두 회사 스킬은 도구·로컬에 없어 적용 불가. 실제 저장 데이터 삭제 종단 검증 미실행이라는 한계 유지.
- [러너 19:18] release published — v2.72.0
- [러너 19:18] gh-release created — GitHub Release v2.72.0
- [러너 19:18] manifest ok — dartfly-v2.72.0.tar.gz dartfly-v2.72.0.tar.gz.sha256 
- [러너 19:18] assets uploaded — 2개
- [러너 19:18] assets verified — v2.72.0 자산 2개 (이전 v2.71.0: 2)
