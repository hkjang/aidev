# Hunter v1.11.0

v1.9.0의 관리자 격리 방문 추적을 TRACKING-STANDARD §4에 맞춰 보완한 릴리즈입니다. 격리 프레임의 보안 정책이 차단한 출처·지시어를 서버에 기록해 설정 화면에서 확인하고 한 번 눌러 허용 원점에 추가할 수 있습니다. 앱 자체 CSP는 완화하지 않으며 일반 PostgreSQL·네 환경변수·서비스 Docker 이미지 하나의 기본 배포를 유지합니다.

## 사용자가 달라지는 점

- 추적 코드가 허용 원점 밖의 주소로 스크립트 로드·요청·이미지를 시도해 브라우저가 차단하면, 격리 프레임이 `securitypolicyviolation` 이벤트의 **출처와 지시어**(`script-src`·`connect-src`·`img-src`)를 로그인한 부모 화면에 전달하고, 부모 화면이 세션·CSRF 검사를 거친 `POST /api/tracking/violations`로 2초 배치로 대신 신고합니다. 격리 프레임은 불투명 출처라 `report-uri`에 세션이 실리지 않기 때문입니다. http(s) 출처만, 지시어·주소 조합당 한 번, 최대 100개까지 전달합니다.
- 서버는 출처를 정규화해 **이 인스턴스의 메모리**에 출처·지시어별 최대 100개(가장 오래 관측된 항목부터 축출)를 횟수·최초·최근 시각·고정 경로와 함께 보관합니다. 경로·쿼리·실제 문서 주소는 저장하지 않고 재시작하면 사라집니다. `inline`·`eval`·`data:`·`blob:` 차단은 허용 대상이 아니라 기록하지 않습니다.
- **관리자 → 서비스 설정 → 방문 추적** 아래 **보안 정책에서 차단된 출처** 패널에서 목록(최근 순, 허용 여부 포함)을 새로 고치고 기록을 지우며, **허용 목록에 추가**를 누르면 그 원점이 허용 원점 입력에 채워지므로 검토한 뒤 저장합니다. 격리 미리보기 중 차단된 출처도 경고 안에 같은 버튼과 함께 표시됩니다.
- `GET/DELETE /api/admin/tracking/violations`를 추가했으며 기록 지우기만 감사 사건(`tracking.violations.clear`)으로 남습니다. 추적이 꺼져 있으면 관리자 미리보기만 신고할 수 있고 그 외 신고는 404입니다. 자동 SSO 진입·추적 코드 저장·미리보기·페이지 이벤트 규칙은 v1.10.0과 같습니다.

## 운영 조건과 한계

Hunter 화면 자체의 정책은 `script-src 'self'`이며 추적 코드를 위해 `'unsafe-inline'`이나 nonce를 더하지 않습니다. 코드는 `sandbox allow-scripts` 격리 프레임에서만 실행되고 그 프레임의 정책은 추적이 켜진 동안에만 존재합니다. 차단 목록은 로그인한 사용자 브라우저가 보낸 값이므로 수집기 주소가 맞는지 확인한 뒤 허용하고, 기록은 감사 로그가 아니며 여러 인스턴스 사이에 공유되지 않습니다. 실제 운영 ReSSO 계정 연동과 외부 분석 서비스의 수신 검증은 이전과 마찬가지로 미수행입니다.

## 검증과 배포

Go 테스트 `TestTrackingViolationLogNormalisesDedupesAndEvicts`(출처 정규화·중복 합산·고리 버퍼 축출)와 `TestTrackingViolationReportsRequireSessionAndAdminView`(세션 없는 신고 거부·추적 꺼짐 시 관리자 미리보기만 허용·관리자 조회·지우기), web 테스트(허용 가능한 http(s) 대상만 프레임 메시지로 전달)를 추가했습니다. 릴리즈 커밋에서 `go vet`, 임시 PostgreSQL 컨테이너의 `go test -race`, `npm test`, `npm run build`(tsc 포함), `node scripts/check-docs.mjs`, `bash -n scripts/release.sh`, `python3 -m py_compile scripts/release-notes.py`, `node scripts/verify-pentagi.mjs`를 실행했습니다. 최종 게시 커밋의 CI와 공개 아카이브 다운로드 검증은 실제 완료한 뒤 [릴리즈 본문](https://github.com/hkjang/hunter/releases/tag/v1.11.0)에 기록합니다.

화면 캡처는 v1.9.0의 검증 장면을 보존했습니다. 관리자 가이드(md·html·pdf)·README·llms.txt·OpenAPI의 CSP 설명과 새 패널·API를 갱신했습니다.

배포 이미지: `hunter:v1.11.0` · 유일한 첨부 자산: `hunter-v1.11.0.tar.gz`.

[사용자 가이드](guides/user-guide.html) · [관리자 가이드](guides/admin-guide.html) · [공식 조사와 적용 범위](research-sso-tracking.md) · [v1.10.0 릴리즈 노트](release-v1.10.0.md)
