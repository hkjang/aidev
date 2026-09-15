# Hunter v1.11.0

에이전트 실행 보고서를 관리자가 허용한 사내 서비스로 직접 넘기는 **다른 서비스로 보내기**와, 방문 추적 격리 프레임의 CSP 차단 출처를 한 번에 허용 목록에 넣는 **보안 정책에서 차단된 출처** 패널을 추가한 릴리즈입니다. 두 기능 모두 기본 꺼짐이며 일반 PostgreSQL·네 환경변수·서비스 Docker 이미지 하나의 기본 배포를 유지합니다.

## 사용자가 달라지는 점

- 실행 상세의 **실행 보고서** 메뉴에 **다른 서비스로 보내기 → 서비스 이름** 항목이 나타납니다. 고르면 파일을 내려받지 않고 새 창에서 그 서비스가 열려 Markdown 보고서를 직접 받아 갑니다. 항목은 관리자가 허용 목록에 markdown 을 받는 서비스를 적어 둔 경우에만 보이며, 기본값은 빈 목록이므로 새 설치는 달라지지 않습니다.
- 넘기는 표(claim)는 사내 문서 넘기기 표준(HANDOFF-STANDARD)의 보내는 쪽 규격을 따릅니다. `POST /api/v1/handoff/claims`는 지금 이 실행을 읽을 수 있는 사용자에게만 256비트 난수·5분·1회용 표를 발급하고(다운로드와 같은 `reportRun` 권한 검사, 읽을 수 없는 실행·미설정 404, 형식 오류 400), `GET /api/v1/handoff/claims/{claim}`은 로그인 없이 `text/markdown` 첨부로 한 번만 내주며 사용됨·만료·미발급을 구별 없이 404로 답합니다. 본문은 발급 시점에 암호화 저장하고 표는 SHA-256 다이제스트만 남기며, 감사 기록(`agent.handoff`)에는 실행 ID와 바이트 수만 적고 로그 경로의 표는 마스킹합니다.
- 관리자는 **서비스 설정 → 다른 서비스로 보내기** 탭(설정 그룹 `handoff.targets`)에서 이름·오리진·받는 형식을 최대 20개 관리합니다. 오리진은 방문 추적과 같은 규칙으로 정규화하며 중복·경로·와일드카드는 거절합니다. markdown 을 받지 않는 서비스는 저장돼도 메뉴에 오르지 않습니다. 받는 쪽에 알려 줄 Hunter 오리진을 같은 화면에서 확인합니다.
- 방문 추적 격리 프레임에서 정책이 차단한 출처·지시어를 로그인된 부모 화면이 `POST /api/tracking/violations`로 대신 신고합니다. 관리자는 **서비스 설정 → 방문 추적**의 **보안 정책에서 차단된 출처** 패널에서 조회·지우기·허용 원점에 추가한 뒤 저장하고, 격리 미리보기의 경고에도 차단된 출처가 함께 보입니다. 서버는 인스턴스 메모리의 100개 고리 버퍼에 원점·지시어만 보관하며 경로·쿼리·inline·eval·data:·blob: 은 기록하지 않습니다. 앱 자체 CSP는 완화하지 않습니다.

## 운영 조건과 한계

받는 쪽(HANDOFF 표준의 받는 서비스)은 Hunter 에 만들지 않았습니다. Hunter 가 들이는 SARIF·JSON 진단 결과·SBOM 은 표준의 형식 표에 없으므로 보내는 쪽 `markdown` 하나만 제공합니다. 실제 사내 받는 서비스(ptium·weekly 등)와의 왕복은 route 스텁으로만 확인했으며 조직에서 `origin/handoff?source=…&claim=…` 진입 화면을 준비해야 합니다. 차단 출처 버퍼는 서버 재시작 시 사라지고 추적이 꺼진 동안 비관리자 신고는 거절합니다.

## 검증과 배포

Go 테스트로 설정 검증(10가지 거절·정규화), 파일명·RFC 5987·로그 마스킹, 통합 테스트(기본 404·형식 표 필터·CSRF·남의 실행 404·표 1회·만료 404·감사에 표 없음·만료 행 정리·열람자 403)와 CSP 위반 신고·정규화·관리자 조회·삭제를 추가했고, web 테스트로 넘기기 URL 조립·페이로드 정리와 위반 이벤트 파서를 추가했습니다. 릴리즈 커밋에서 `go vet`, 임시 PostgreSQL 컨테이너의 `go test -race`, `npm test`, `npm run build`(tsc 포함), `node scripts/render-guides.mjs`, `node scripts/check-docs.mjs`, `bash -n scripts/release.sh`, `python3 -m py_compile scripts/release-notes.py`를 실행했습니다. 최종 게시 커밋의 CI와 공개 아카이브 다운로드 검증은 실제 완료한 뒤 [릴리즈 본문](https://github.com/hkjang/hunter/releases/tag/v1.11.0)에 기록합니다.

화면 캡처는 v1.9.0의 검증 장면을 보존했습니다. 관리자 가이드 §5.7·사용자 가이드 §10.8·README·llms.txt·openapi.json 을 갱신하고 가이드 HTML·PDF 를 재생성했습니다.

배포 이미지: `hunter:v1.11.0` · 유일한 첨부 자산: `hunter-v1.11.0.tar.gz`.

[사용자 가이드](guides/user-guide.html) · [관리자 가이드](guides/admin-guide.html) · [공식 조사와 적용 범위](research-sso-tracking.md) · [v1.10.0 릴리즈 노트](release-v1.10.0.md)
