### 추가

- 관리자가 재배포 없이 관리 화면 **방문 추적** 탭에서 방문 추적 스크립트를 붙일 수 있는 체계 추가. 기본값은 꺼짐이며 켜기 전까지 어떤 화면에도 스니펫이 들어가지 않고 응답 헤더도 이전 릴리스와 같습니다. 제공자는 `momento`(첫 자리, 권장)·`ga4`·`gtm`·`matomo`·`custom`(직접 붙여넣기)이고, 설정은 다른 관리 설정과 같이 `settings` 테이블의 `tracking` 키에 저장하며 잘못된 타입·제공자·주소와 8KB를 넘는 스니펫은 `400`으로 거부합니다
- 스니펫을 붙이면서도 `'unsafe-inline'`을 쓰지 않도록 SPA 셸 응답마다 nonce를 만들어 스니펫의 모든 `<script>`에 붙이고 같은 nonce를 `script-src`에 넣습니다. 제공자와 붙여넣은 스니펫에서 읽은 출처를 `script-src`·`connect-src`·`img-src`에 더하고, 추적이 켜진 동안에만 `report-uri`를 넣어 브라우저가 신고한 차단 출처와 지시어를 메모리(100개 고리)에 기억합니다. 신고 수신 `POST /api/v1/tracking/csp-report`는 인증 없이 항상 `204`이며 감사 로그에 남기지 않습니다
- 차단 출처를 다루는 `admin` 전용 API 추가: 목록 `GET /api/v1/tracking/violations`, 기록 삭제 `DELETE /api/v1/tracking/violations`, 한 번 눌러 허용 목록에 더하는 `POST /api/v1/tracking/violations/allow`. 방문 추적 탭의 **정책이 차단한 출처** 표가 같은 기록을 보여 주고 **허용** 버튼으로 다음 화면 요청부터 정책에 반영합니다
- Momento 같은 오리진 프록시 `/momento/*` 추가(기본 ON). 추적이 켜져 있고 제공자가 Momento일 때만 열리며 그 밖에는 `404`입니다. `GET`·`HEAD`·`POST`만 넘기고 브라우저가 붙인 `Cookie`·`Authorization`·`X-Vault-Token`을 지운 뒤 전달하며, 수집기가 돌려준 `Set-Cookie`는 브라우저에 넘기지 않습니다. 본문은 64KB, 제한 시간은 10초입니다. 프록시를 켜 두면 브라우저는 jikim 오리진만 보므로 정책에 외부 출처가 등장하지 않습니다

### 변경

- 브라우저가 그리지 않는 `/api/*`, `/v1/*`, `/mcp`, `/healthz`, `/readyz`, `/momento/*` 응답의 콘텐츠 보안 정책을 `default-src 'none'`으로 더 좁혔습니다. 설정 저장소에 닿지 못하면 화면 응답은 추적 없이 이전 정책으로 내려가 로그인 화면이 함께 멈추지 않습니다
- `internal/store/settings.go`의 쓰이지 않던 `var _ = pgx.ErrNoRows` 제거

### 문서

- 관리자 가이드 3.7절에 방문 추적 설정 방법, CSP와 nonce 동작, 프록시 경계와 API 표를 추가하고 실제로 띄운 방문 추적 탭 캡처(`docs/screenshots/admin-settings-tracking.png`)를 실었습니다. 보안 기본값 표에 방문 추적과 수집기 내부 HTTP 허용 항목을 더했고 PDF를 같은 변환기로 다시 구웠습니다


**Full Changelog**: https://github.com/hkjang/jikim/compare/v0.2.10...v0.2.11
