- 방문자 가져오기가 한국어 윈도우 엑셀이 CP949로 저장한 "CSV (쉼표로 분리)" 파일을 깨진 글자로 읽어 "이름(name) 열이 필요합니다"로 거절하던 문제를 수정. UTF-8로 읽히면 그대로 두고 아니면 EUC-KR로 해석하며, UTF-8 바이트 순서 표시도 앞에서 떼어 낸다. 더 이상 쓰지 않는 robfig/cron 의존성을 정리.
- internal/app 테스트 패키지가 go test 기본 10분 제한에 걸리던 원인을 수정. OpenAPI 경로 순회 테스트가 `/lobby/stream` SSE 핸들러에서 클라이언트 이탈을 영원히 기다렸다 — 각 프로브가 2초 데드라인 컨텍스트를 갖도록 했다.

**Full Changelog**: https://github.com/hkjang/visitflow/compare/v2.7.3...v2.7.4
