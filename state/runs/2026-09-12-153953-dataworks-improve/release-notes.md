## Data Works v0.9.53

### 주요 변경 사항
- **관리자가 화면에서 붙이는 방문 추적 스크립트 체계 추가 — nonce CSP·출처 추출·차단 기록·같은 오리진 Momento 프록시 (v0.9.53)**: 워크벤치에 방문 추적 스니펫을 넣을 방법이 없었고 SPA shell 에는 CSP 자체가 없었음. 새 `internal/tracking` 패키지가 설정을 읽어 provider 스니펫을 요청별 nonce 로 렌더링하고(모든 `<script>` 에 부착), 스니펫에서 http(s) 출처를 추출해 정책에 넣으며, 브라우저가 막은 출처를 100개 고리 버퍼에 기록하고 허용 목록에 추가할 수 있게 함. 설정은 기존 admin settings 레지스트리에 `tracking.*` 13개(카테고리 `tracking`, 전부 기본 꺼짐, 8KB·provider·URL·placement·allowed_hosts 검증)로 얹어 런타임 스냅샷과 멀티 파드 폴링에 그대로 편승. 워크벤치 shell(`index.html`) 응답에만 `script-src 'self'`(+nonce·provider 출처·report-uri, `style-src 'unsafe-inline'` 은 React 인라인 스타일 때문에 허용) 정책을 달고 비화면 경로는 `default-src 'none'` 으로 좁혔으며 zero modtime 으로 304 를 막아 캐시된 본문과 새 nonce 가 어긋나지 않게 함. 레거시 콘솔(`/admin`)은 인라인 핸들러·style 때문에 CSP 없이 `tracking.include_admin` 일 때만 스니펫을 삽입(가이드에 명시). Momento 를 첫 provider 로 두고 `tracking.momento_proxy`(기본 켜짐)면 `/momento/*` 를 `httputil.ReverseProxy` 로 수집기에 넘기며(쿠키·Authorization 제거, 추적 꺼지면 404) 스니펫에 `data-endpoint="/momento"` 를 줘 정책에 외부 출처가 등장하지 않음. 엔드포인트 `POST /tracking/csp-report`(무인증, 추적 꺼져 있으면 폐기)·`GET /admin/tracking/status`·`GET|DELETE /admin/tracking/violations`·`POST /admin/tracking/violations/allow` 를 추가하고 SPA 설정 화면에 "방문 추적" 탭(provider 별 필드, 현재 상태·미완료 사유·정책 출처, 차단된 출처 목록 + 허용/비우기)을 추가. `internal/tracking` 단위 테스트 11건과 proxy HTTP 회귀 테스트 6건(꺼짐 시 페이지 불변·엄격 정책·프록시 닫힘 / 프록시 nonce 주입·요청별 새 nonce·자격 증명 제거 / custom 스니펫 출처 반영·신고→목록→허용→정책 반영 / 설정 검증 400)을 추가했고 실제 SPA 를 임베드한 서버와 스텁 수집기·headless Chromium 으로 비콘 도착·CSP 콘솔 오류 없음·끄면 원복을 확인. `docs/ADMIN_GUIDE.md` 4.5절에 설정 방법과 CSP·nonce·차단 출처 설명을 추가하고 PDF 를 다시 생성. `dataworks:v0.9.53` 이미지를 `dataworks-v0.9.53.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.53.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.53.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.53
```