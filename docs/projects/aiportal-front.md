---
title: "aiportal-front — 자율 개선 이력"
description: "aiportal-front: 자율 개선 회차 4회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-05 18:02:02 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "aiportal-front",
 "codeRepository": "https://github.com/hkjang/aiportal-front",
 "url": "https://hkjang.github.io/aidev/projects/aiportal-front/",
 "description": "aiportal-front: 자율 개선 회차 4회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-05T18:02:02+09:00"
}
</script>

# aiportal-front

<p class="tldr"><strong>요약.</strong> aiportal-front: 자율 개선 회차 4회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>4</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>4</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/aiportal-front">https://github.com/hkjang/aiportal-front</a></dd>
<dt>마지막 회차</dt><dd>2026-09-05 00:31 KST — <span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/4">PR #4</a>, release skipped</dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>저장소에 릴리즈 이력이 전혀 없음: git 태그 0개, GitHub Release 0개(gh release list 결과 없음), CHANGELOG.md/RELEASE 노트 없음, .github/workflows 디렉터리 자체가 없음, scripts/ 디렉터리·Makefile·version-check 등 릴리즈 스크립트 없음. package.json 의 version 은 첫 커밋(ab700ba)의 Vite 스캐폴드 기본값 &quot;0.0.0&quot; 그대로이며 전체 히스토리(5개 커밋)에서 한 번도 변경된 적 없고 private:true 라 배포 대상도 아님. .gitlab-ci.yml 은 $CI_COMMIT_BRANCH == main/develop 브랜치 푸시에만 반응하는 빌드·배포 잡만 있고 태그나 릴리즈에 반응하는 잡은 없음(파일 내 tags: 는 모두 GitLab Runner 태그). 따를 이전 방식(버전 체계·노트 양식·태그 형식·자산 규칙)이 존재하지 않아 관례를 새로 만들지 않고 건너뜀.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="merged"><td data-label="일시">2026-09-05 00:31</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-04 05:45</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/3">PR #3</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 22:04</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 13:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-03
- 선택: 공통 유틸 Vitest 단위 테스트 도입 및 파일 확장자 판별 오류 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 자동화 테스트가 전무했던 저장소에 Vitest + jsdom 환경(`vitest.config.js`, `npm test` / `npm run test:watch`)을 추가하고 `common`, `page`, `chatFile`, `favToggle` 유틸에 대한 단위 테스트 61건을 작성했다. 테스트 작성 중 발견한 실제 버그 2건(`isPreviewablXlsx` 가 `['xlsx','xlsx']` 로 중복 비교해 `.xls` 를 인식하지 못하던 문제, 파일 판별 함수들이 `file` 이 null 일 때 `file.file_name` 접근으로 예외를 던지던 문제)을 고치고 회귀 테스트로 고정했다. 검증은 `npm test`(61건 통과)와 `npm run build:dev`(빌드 성공)로 수행했다. README 와 `docs/09-테스트가이드-총론.md` 에 실행 방법을 문서화했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (가치 4 / 위험 3 / M)
  - ESLint + Prettier 도입 (탭·스페이스 혼재로 초기 diff 노이즈가 커서 보류) (가치 3 / 위험 2 / M)
  - `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (운영 표시값 영향 커서 보류) (가치 3 / 위험 4 / S)
  - `useFileAccept` 의 mode 값 불일치(`modeKey` 는 'ocr', 제외 확장자 분기는 'OCR') 수정 (가치 3 / 위험 3 / S)
  - `docs/07-개선사항-권장사항.md` 의 스토어 패턴 통일(useServiceStore → Pinia defineStore) (가치 3 / 위험 4 / L)

## 2026-09-03
- 선택: 파일 업로드 accept/용량 정책 오류 수정 및 단위 테스트 추가 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `useFileAccept` 에서 실사용 버그 4건을 고쳤다 — (1) `SupportOcr.vue` 가 넘기는 `"OCR"` 이 소문자 키 맵과 매칭되지 않아 `ocrFileLimit` 정책이 적용되지 않고 항상 기본 50MB 였던 문제, (2) 괄호 표기 확장자(`"엑셀(xlsx,xls)"`)를 파싱할 때 `"." + [".xlsx"].join()` 으로 점을 중복해 `..xlsx` 같은 잘못된 accept 토큰을 만들던 문제, (3) OCR 제외 확장자 필터가 괄호 표기 항목과 fallback 목록에는 적용되지 않던 문제, (4) 세 개 화면(`SupportOcr`, `PopSimpleBot`, `PopSimpleBotUpdate`)의 용량 초과 알림이 바이트 값을 "52428800MB" 처럼 MB 로 표기하던 문제. accept/용량 계산을 순수 함수(`parseExtensionEntry`, `buildAcceptString`, `buildFallbackAccept`, `resolveMaxSizeMb`)로 분리하고 회귀 테스트 18건을 추가했다. 검증은 `npm test`(총 79건 통과)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (가치 4 / 위험 3 / M)
  - ESLint + Prettier 도입 (탭·스페이스 혼재로 초기 diff 노이즈가 커서 보류) (가치 3 / 위험 2 / M)
  - `parseUtcToKstDate` 가 로컬 타임존이 KST 일 때 +9h 를 이중 적용하는 문제 정리 (운영 표시값 영향 커서 보류) (가치 3 / 위험 4 / S)
  - `buttonSettingPolicy.buildSettingActions` 모드별 액션 구성 단위 테스트 보강 (가치 2 / 위험 1 / S)
  - 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-04
- 선택: 앱 정보 조회(getAppInfo) 캐시/재조회 경로 오류 수정 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `src/utils/appList.js` 의 `getAppInfo` 에서 실사용 버그 3건을 고쳤다 — (1) 사이드메뉴 캐시 미존재 시 기본값이 `{}` 라 `appList.length === 0` 이 항상 false 가 되어 API 재조회를 건너뛰고 `{}.find` 에서 TypeError 가 나 `{}` 를 반환하던 문제(딥링크로 /chatmain?appId= 진입 시 앱 정보 유실), (2) `serviceMenu === 'all'` 재조회가 `serviceCode === 'all'` 로 필터링돼 항상 빈 목록을 반환, 사이드메뉴 상위 30개에 없는 앱은 절대 찾지 못하던 문제, (3) 재조회로 찾은 앱은 `writeOpenApp`/simple 모드 후처리를 건너뛰던 경로 불일치. 순수 함수(`toAppArray`, `pickServiceRows`, `flattenAppList`, `findAppInfo`)로 분리하고 `findAppInfo` 의 app_id 문자열 비교·빈 키·비배열 입력 방어를 추가했다. 검증은 `npm test`(총 100건 통과, 신규 21건)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (가치 4 / 위험 3 / M)
  - `useFileAttach` 의 알림 문구가 옵션(maxFileSize/maxTotalSize)을 무시하고 20MB/100MB 로 하드코딩된 문제 수정 (가치 3 / 위험 1 / S)
  - `useAppList.getFormattedAppList` 의 `String(app?.knowledge_info?.status) ?? ''` 가 문자열 "undefined" 를 만드는 문제 정리 (Sidemenu.vue 에도 동일 코드 중복) (가치 2 / 위험 1 / S)
  - `mitt` 이 `src/utils/eventBus.js` 에서 직접 import 되는데 package.json 직접 의존성에 없어 전이 의존성에 기대고 있는 문제 (가치 3 / 위험 1 / S)
  - 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)

## 2026-09-05
- 선택: 앱 이동(updateLogAndGo) 쿼리 유실·외부 링크 취약점 수정 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 모든 앱 카드 클릭이 거치는 공통 네비게이션 경로 `src/utils/appUpdateLog.js` 에서 5건을 고쳤다 — (1) `buildTo` 가 `to` 를 문자열로 받으면 `extraQuery` 를 그대로 버려 img 모드 앱의 `?mode=img` 가 유실되던 문제(문자열도 `{path, query, hash}` 로 병합하도록 변경), (2) 같은 이유로 `nextTo.path` 가 undefined 라 `/chatmain` 판별에 실패해 `writeOpenApp` 이 호출되지 않고 챗 화면이 앱 컨텍스트를 잃던 문제(`toRoutePath` 도입), (3) `mode === 'link'` 의 `window.open(urlLink, '_blank')` 이 `noopener` 없이 열려 reverse tabnabbing 이 가능하고 `javascript:`/`data:` url_link 가 그대로 실행되던 문제(`isSafeExternalLink`/`openExternalLink` 로 http·https·mailto 만 허용, 차단 시 일반 이동으로 폴백), (4) iframe 모드도 동일하게 검증해 위험한 값은 빈 문자열로 넘기도록(ChatFrame 은 빈 값이면 "불러올 메신저가 없습니다" 표시) 처리, (5) `url_link` 가 문자열이 아닐 때 `.replace` 로 예외가 나던 문제. 순수 헬퍼(`toRoutePath`, `buildTo`, `resolveUserIdFromEmail`, `applyUserIdTemplate`, `isSafeExternalLink`, `openExternalLink`)로 분리하고 회귀 테스트 30건을 추가했다. 검증은 `npm test`(총 130건 통과, 신규 30건)와 `npm run build:dev`(빌드 성공)로 수행했다.
- 보류 아이디어:
  - markdown.js 의 DOMPurify 설정/onclick 파싱 경로 XSS 하드닝 검토 (transformAppAnchors 가 `&lt;` 를 실태그로 되돌림) (가치 4 / 위험 3 / M)
  - `useFileAttach` 알림 문구 하드코딩(20MB/100MB) 및 `msg.includes('10')` 로 서버 오류를 용량초과로 오판하는 문제 수정 (가치 3 / 위험 1 / S)
  - `useTableUtils.sortData` 가 모든 값을 `getNum` 으로 숫자 변환해 텍스트·날짜 컬럼 정렬이 동작하지 않는 문제 (현재 .vue 에서 미사용) (가치 2 / 위험 2 / S)
  - `mapAppToCard(item, idx, 'prompt')` 결과에 `to` 가 없어 GlobalSearch 프롬프트 카드 클릭이 무반응인 문제 (의도 확인 필요) (가치 3 / 위험 3 / S)
  - 빌드 산출물 단일 청크 6.2MB 문제(manualChunks 코드 스플리팅) 개선 (가치 3 / 위험 3 / M)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
