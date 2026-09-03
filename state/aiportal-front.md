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
