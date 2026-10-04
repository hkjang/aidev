## 2026-10-05
- 선택: 사진 정제의 EXIF·GPS 제거를 실제 JPEG 회귀 테스트로 보장 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: image.test.ts 1파일에 실제 jpeg-js.encode/decode로 만든 합성 JPEG의 TIFF GPS IFD·위경도 RATIONAL·COM 존재를 입증하고, 실제 sanitise의 축소 유무 × full/thumb 4경우에서 정상 디코딩·예상 크기·메타데이터 제거를 검증하는 총 7건을 추가했다. JPEG 세그먼트 길이를 따라 SOS 이전 헤더를 검사하며 EXIF 원본 및 COM만 있는 원본을 각각 거부하는 대조 assertion을 포함했고 프로덕션 코드는 바꾸지 않았다. Node v22.23.1에서 npm ci 성공 후 npm test -- packages/core/src/__tests__/image.test.ts(입력 검증 1건, 완성 후 7건), npm run check(린트·3 workspace 타입 검사·17파일 180건), npm run build 모두 exit 0; 커밋 14f79f7.
- 실패 재현: 못 함 — 기존 sanitise가 이미 메타데이터를 제거하므로 프로덕션 결함의 red→green은 없으며 입력 검증 첫 실행부터 통과했다. EXIF APP1 must be removed / COM must be removed assertion이 메타데이터가 남은 실제 JPEG를 거부함을 각각 toThrow로 확인했다(이 대조군을 기존 코드의 실패 재현으로 주장하지 않음).
- 보류 아이디어: Node StaticFiles 실제 파일 기반 HEAD·SPA·경로 격리 테스트 [3/1/S] — 이번 차선 후보, 현재 shell fallback 계약 유지.
- 보류 아이디어: 사진 입력 바이트·형식 한계의 AppError 회귀 테스트 [3/1/S] — 실제 JPEG와 손상 입력으로 별도 검증.
- 보류 아이디어: 연봉 RLS E2E 쿼리·결과 파싱 실패를 성공 처리하지 않게 한다 [4/2/M] — 격리 PostgreSQL 필요.
- 보류 아이디어: http/pipeline.ts CSRF·본문 크기 한계 테스트 [3/2/M] — 실제 DB·createApp 배선 필요, 대역 금지.
- 과제서: 채택 — 현재 sanitise와 mediaRoutes가 정찰 근거와 일치하고 지정된 테스트 1파일만으로 수용 기준을 충족했다.
