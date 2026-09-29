<p align="center">
  <img src="docs/favicon.svg" alt="SeatOn Logo" width="90"><br><br>
  <h1 align="center">SeatOn</h1>
</p>

<p align="center">
  <strong>사무실 도면과 직원 정보를 연결하는 오프라인 우선 스마트 좌석 & 공간 관리 플랫폼</strong><br>
  SVG 비율좌표 좌석맵, 오프라인 CV·사내 비전 모델 선택형 도면 판독, 이상 좌석 자동 감지 및 Streamable MCP 지원.
</p>

<p align="center">
  <a href="https://hkjang.github.io/seaton/">🇰🇷 홍보 페이지</a> · <a href="https://hkjang.github.io/seaton/index_en.html">🇺🇸 English Page</a> · <a href="https://hkjang.github.io/">🌐 전체 서비스</a> · <a href="https://github.com/sponsors/hkjang">💖 Sponsor</a>
</p>

---

## 주요 기능

- 일반 직원용 검색 중심 좌석맵과 역할별 관리자 페이지
- PNG/JPG/PDF 도면 버전 관리, 비율 좌표 SVG 편집 기반, PDF도 좌석 오버레이 지원
- 좌석 인식 엔진 선택: 오프라인 CV(기본, 외부 통신 없음) · 사내 비전 모델(VLM) · 두 결과를 교차 검증하는 하이브리드
- 도면별 좌석 격자 보정과 일괄 정렬로 실제 책상 열 간격에 맞춘 스냅
- 직원 CSV/XLSX 가져오기, Drag & Drop 배정, 좌석 변경 이력
- 미배정·퇴직자 점유·조직 영역 불일치·저신뢰 좌석 자동 탐지
- 예외 중심 관리자 작업 큐와 즉시 조치, 운영 준비도 및 연동 상태 대시보드
- 좌석 직접 이동, Shift 다중 선택, 스냅·정렬·회전·Undo/Redo 배치 편집
- 도면 팬·휠 확대, 검색 좌석 자동 이동, 미니맵, 조직별 색상과 구역 표시, 좌석 상태 필터
- 변경 이력 검색·기간 필터와 CSV 내보내기, 세션 만료 시 로그인 안내
- Keycloak OIDC Discovery + Authorization Code/PKCE + nonce 검증
- Keycloak 그룹 기반 RBAC와 SSO 사용자 자동 생성
- 설치별 암호화 키, 개인별 API 키 생성·회전·폐기·범위 제어
- REST/OpenAPI 및 MCP Streamable HTTP
- 로그인 화면과 프로필 컨텍스트 메뉴의 빌드 버전 표시
- 관리자가 화면에서 붙이는 방문 추적(Momento 우선, nonce 기반 CSP, 차단 출처 기록)
- 런타임 외부 CDN/인터넷 연결이 없는 단일 서비스 이미지

## UI 프레임워크 결정

운영형 관리자 화면에는 [Material UI](https://mui.com/material-ui/)를 사용했다. 접근성 있는 폼·테이블·반응형 레이아웃과 일관된 테마를 빠르게 유지할 수 있고 React 19를 공식 지원한다. 좌석맵은 별도 외부 지도 SDK 대신 SVG로 구현하여 비율 좌표, 선택, 줌, Drag & Drop을 오프라인에서도 예측 가능하게 유지한다.

## 실행

외부 또는 사내 PostgreSQL 14+ 데이터베이스를 준비한다. SeatOn이 시작할 때 스키마를 자동 생성한다.

```bash
docker load < SeatOn-v1.4.12.tar.gz

export POSTGRES_DSN='postgres://seaton:password@postgres.intra:5432/seaton?sslmode=require'
export BOOTSTRAP_ADMIN='admin'
export BOOTSTRAP_ADMIN_PASSWORD='change-this-strong-password'
export SEATON_IMAGE_TAG='v1.4.12'
docker compose up -d
```

`SEATON_IMAGE_TAG`는 Compose 파일의 이미지 선택용 셸 치환값이며 컨테이너 환경변수로 전달되지 않는다. 애플리케이션이 받는 환경변수는 아래 세 개뿐이다.

| 환경변수 | 설명 |
| --- | --- |
| `POSTGRES_DSN` | PostgreSQL DSN |
| `BOOTSTRAP_ADMIN` | 최초 시스템 관리자 아이디 |
| `BOOTSTRAP_ADMIN_PASSWORD` | 최초 관리자 비밀번호, 12자 이상 |

부트스트랩 계정이 이미 있으면 환경변수 비밀번호로 덮어쓰지 않는다. 비밀번호 변경이나 SSO 전환 후에도 안전하다.

`http://host:8080`에 접속한 뒤 시스템 설정에서 서비스명, Keycloak, 인사 연동, 좌석 인식 엔진과 AI 기준, 세션과 키 정책을 관리한다. 좌석 인식 엔진 기본값은 오프라인 CV이며, 이 상태에서는 런타임 외부 통신이 전혀 없다. 운영 환경에서는 리버스 프록시에서 HTTPS를 종료하고 `X-Forwarded-Proto`와 `X-Forwarded-Host`를 전달한다.

## Keycloak 연결

1. 관리자 → 시스템 설정 → Keycloak SSO에서 `Issuer URL`, `Client ID`, `Client Secret`을 입력한다.
2. Keycloak Client의 Client authentication을 켠다.
3. Valid Redirect URI에 `https://서비스주소/api/v1/auth/oidc/callback`을 등록한다.
4. 필요하면 `/seaton-admins`, `/seaton-seat-managers` 그룹명을 바꾼다.
5. Keycloak의 Group Membership mapper로 `groups` claim을 ID Token에 포함한다.
6. **저장 후 연결 테스트**로 Discovery URL과 Callback을 확인하고 SSO를 활성화한다.
7. Keycloak에 이미 로그인한 사람이 로그인 화면 없이 바로 들어오게 하려면 **Keycloak 세션이 있으면 자동 로그인**(`oidc.auto_login`, 기본 꺼짐)을 켠다. `prompt=none` 조용한 로그인이며 동작과 재시도 방지는 관리자 가이드 §3.3에 있다.

그 외 Keycloak 엔드포인트는 Issuer의 표준 Discovery 문서에서 자동 구성한다.

### MCP를 SSO로 열기

`/mcp`는 기본적으로 개인 API 키로만 열린다. 같은 탭의 **MCP SSO(OAuth) 인증** 카드를 켜면(`mcp.oauth.enabled`, 기본 꺼짐) MCP 클라이언트에 주소 하나만 주면 되고, 클라이언트가 `/.well-known/oauth-protected-resource/mcp`(RFC 9728)를 읽어 Keycloak에서 스스로 로그인해 토큰을 받아 온다. SeatOn은 리소스 서버로서 토큰의 서명·발급자·만료·대상(`aud`에 리소스 식별자, 또는 `aud`/`azp`가 허용 대상에)만 검사하고, 이미 웹으로 로그인해 등록된 활성 계정만 통과시킨다 — 계정을 만들거나 권한을 올리지 않으며 REST는 여전히 키와 세션만 받는다. Keycloak 클라이언트·Audience 매퍼 설정과 거부 메시지별 조치는 관리자 가이드 §3.3에 있다.

## 좌석 인식 엔진

관리자 → 시스템 설정 → AI 분석에서 도면 판독 방식을 고른다.

| 엔진 | 동작 | 외부 통신 |
| --- | --- | --- |
| `cv` (기본) | 내장 오프라인 CV 파이프라인 `offline-cv-v2` | 없음 |
| `vlm` | 사내 비전 모델 단독 판독 | 설정한 사내 엔드포인트 |
| `hybrid` | CV와 VLM 결과를 IoU로 교차 검증 | 설정한 사내 엔드포인트 |

`vlm`·`hybrid`를 쓰려면 OpenAI 호환 `/chat/completions`를 제공하는 **사내** 추론 서버 주소를 입력한다. vLLM이나 Ollama로 Qwen2.5-VL 계열을 서빙한 주소를 쓰고, **저장 후 VLM 연결 시험**으로 주소·인증·응답 형식·좌표계를 한 번에 확인한다. 비전 모델의 확신도는 보정된 확률이 아니므로 VLM이 단독으로 찾은 좌석은 자동 승인되지 않고 항상 검토 큐에 남는다. VLM 호출이 실패하면 CV 결과로 자동 대체되어 분석은 완료된다.

분석은 비동기 작업이다. 분석 요청은 즉시 `202`와 작업 ID를 돌려주고 관리자 화면이 완료까지 진행 상태를 보여준다. 엔진별 동작과 예상 실패 처리는 [docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md)에 정리했다.

## 방문 추적

관리자 → 시스템 설정 → 방문 추적에서 화면을 다시 배포하지 않고 추적 도구를 붙인다. 기본값은 꺼짐이다. 추적 도구는 사내 자체 호스팅 수집기 **Momento**(기본, 같은 오리진 프록시 `/momento/*`로 외부 출처 없이 동작)와 GA4·GTM·Matomo·직접 붙여넣기(8KB)를 지원한다.

SeatOn의 응답은 `script-src 'self'` 로 잠겨 있으므로 스니펫을 그냥 넣으면 브라우저가 조용히 막는다. 추적이 켜진 문서에는 요청마다 다른 nonce 를 스니펫의 모든 `<script>` 와 `script-src` 에 함께 넣고, 스니펫에 적힌 출처를 정책에 더하며, 브라우저가 차단한 출처를 `report-uri` 로 받아 같은 화면에서 한 번 눌러 허용에 더할 수 있다. `'unsafe-inline'` 은 쓰지 않고, 끄면 정책이 원래대로 돌아간다. 자세한 설정과 CSP 설명은 [docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md) §3.6 에 있다.

## 백업과 복구

PostgreSQL 백업과 `seaton-data` 볼륨을 한 세트로 백업한다. `/var/lib/seaton/master.key`를 잃으면 DB의 암호화된 Client Secret과 API Token을 복호화할 수 없으므로 새로 입력해야 한다. 개인 API 키 원문은 어떤 경우에도 복구되지 않는다.

## 개발

```bash
cd web && npm ci && npm run build
cd .. && go test ./...
docker build -t seaton:dev .
```

### 화면 검증

타입 검사와 단위 테스트는 화면이 실제로 어떻게 그려지는지 보지 못한다. 좌석 라벨이 잘려 읽히지 않거나, 조작 패널이 도면을 가리거나, 좌석 클릭이 먹히지 않는 결함은 브라우저로 열어야 드러난다. 그래서 실행 중인 SeatOn을 그대로 열어 확인하는 Playwright 검증을 둔다.

```bash
docker compose up -d                       # 검증 대상 서버 기동
cd web && npm ci
npx playwright install --with-deps chromium
E2E_BASE_URL=http://127.0.0.1:8080 \
E2E_USERNAME=admin E2E_PASSWORD=... npm run e2e
```

첫 실행 시 `web/e2e/seed.mjs` 가 사업장·층·도면·좌석·직원·배정을 만들어 둔다. 도면이 이미 있으면 아무것도 하지 않으므로 반복 실행해도 안전하다. 좌석은 `web/e2e/fixtures/plan.png` 를 CV 엔진으로 분석해 만들어지므로, 인식 정확도가 무너지면 시드 단계에서 바로 실패한다.

검증 항목은 과거에 실제로 났던 결함을 그대로 따라간다. 좌석 번호가 접두사에 밀려 잘리지 않을 것, 조작 패널이 도면 영역을 침범하지 않을 것, 휠 확대가 동작하고 그때 라벨이 좌석을 뒤덮지 않을 것, 좌석 클릭으로 상세가 열릴 것, 각 화면에서 콘솔 오류가 없을 것이다. CI의 `e2e` 잡이 PostgreSQL 서비스와 방금 만든 이미지를 띄워 같은 검증을 돌리고, 실패하면 서버 로그와 Playwright 리포트를 남긴다.

API/MCP 세부사항은 [docs/API_AND_MCP.md](docs/API_AND_MCP.md), 보안·배치 구조는 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md), 운영과 엔진 설정은 [docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md)를 참고한다.

### 문서 산출물

`docs/*.md` 가 단일 원본이고 배포용 HTML·PDF는 생성물이다. 문서를 고친 뒤에는 생성 스크립트를 다시 실행한다.

```bash
pip install reportlab
python3 scripts/build-docs.py             # docs 전체 HTML(+ 보고서 PDF) 재생성
python3 scripts/build-docs.py ADMIN_GUIDE  # 특정 문서만
```

보고서 PDF는 `docs/fonts/NanumGothic.ttf` 를 임베드하므로 한글 폰트가 없는 환경에서도 동일하게 열린다.

사용자 가이드([docs/USER_GUIDE.md](docs/USER_GUIDE.md))와 관리자 가이드([docs/ADMIN_GUIDE.md](docs/ADMIN_GUIDE.md))는 다른 프로젝트와 서식을 맞추기 위해 PDF를 공용 도구로 굽는다. 두 문서가 싣는 화면 캡처는 `docs/assets/guide/` 에 있으며, 실제로 띄운 SeatOn을 찍은 것만 둔다.

```bash
# 1. 캡처 — 버려도 되는 로컬 배포를 가리킨다. 시드가 없으면 채우고, 만든 API 키는 폐기한다.
cd web && GUIDE_SHOT_BASE_URL=http://127.0.0.1:8080 \
GUIDE_SHOT_USERNAME=admin GUIDE_SHOT_PASSWORD=... node e2e/guide-shots.mjs

# 2. PDF — aidev 저장소의 공용 변환기
node ../aidev/tools/guide/md2pdf.mjs docs/USER_GUIDE.md docs/USER_GUIDE.pdf \
  --title "사용자 가이드" --subtitle "좌석맵 검색부터 도면 등록·배정·MCP 연동까지" --project SeatOn --version v1.4.12
node ../aidev/tools/guide/md2pdf.mjs docs/ADMIN_GUIDE.md docs/ADMIN_GUIDE.pdf \
  --title "관리자 가이드" --subtitle "설치·설정·계정·운영·장애 대응·보안" --project SeatOn --version v1.4.12
```

## 릴리스

`v1.4.12` 형태의 태그를 push하면 GitHub Actions가 `linux/amd64` 서비스 이미지를 `seaton:v1.4.12` 으로 빌드하고 `docker save` 결과를 `SeatOn-v1.4.12.tar.gz` 로 GitHub Release에 첨부한다. 런타임에는 레지스트리나 인터넷이 필요 없다.

이름 규칙은 이미지 `서비스명:v버전`, 배포 파일 `서비스명-v버전.tar.gz` 이다. 애플리케이션이 `/api/v1/version` 으로 알리는 버전 문자열은 `v` 없는 semver(`1.4.12`)를 그대로 쓴다.

로컬 검증은 다음과 같다.

```bash
./scripts/release-image.sh 1.4.12
gzip -t SeatOn-v1.4.12.tar.gz
```
