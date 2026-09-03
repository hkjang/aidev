# aiportal-front 자율 개선 기록

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
