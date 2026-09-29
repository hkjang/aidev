워크스페이스를 ZIP으로 내보낼 때 **문서가 정확히 2000건이면 일부가 빠졌다고 잘못 안내하던 것**을 고쳤습니다. 내보내기 한도는 2000건이며, 실제로 한도를 넘을 때만 안내합니다. 마이그레이션과 설정 변경은 없습니다.

## 고친 것

### 2000건을 모두 담고도 생략했다고 안내하던 것

기존 조회는 최대 2000행만 읽은 뒤, 조회 결과가 2000건이면 한도를 넘었다고 판단했습니다. 이 결과만으로는 전체가 정확히 2000건인지, 그보다 많은지 구분할 수 없었습니다.

이제 PostgreSQL에서 최대 2001행을 조회하여 초과 여부를 판정합니다. 2001번째 행이 있으면 문서 목록을 2000건으로 자른 뒤 렌더링·ZIP 목록·감사 기록을 처리합니다. 초과 판정용 문서는 ZIP에 들어가지 않습니다.

| 내보내기 대상 문서 | ZIP에 담는 문서 | 생략 안내 |
| --- | --- | --- |
| 1999건 | 1999건 | 없음 |
| 2000건 | 2000건 | 없음 |
| 2001건 | 2000건 | 있음 |

ZIP에는 문서 외에 안내 파일 `목록.md` 한 개가 추가됩니다. 이번 수정은 2000건을 초과하는 워크스페이스의 전체 백업 기능을 제공하지 않습니다.

실제 PostgreSQL에 각 경계의 문서를 넣고 인증된 HTTP 라우트에서 받은 ZIP을 읽는 테스트가 안내 여부, 항목 수, 목록과 항목 이름의 일치를 확인합니다. 구현 단계에서 수정 전 실패와 수정 복원 시 재실패를 확인했고, 릴리즈 단계에서도 전체 Go 테스트를 전용 DB로 다시 실행해 통과했습니다.

### v0.48.0 이후 함께 포함된 변경

HTML·마크다운에서 가져오는 그림 설명이 길 때 첨부 이름에 AI 컨텍스트 생략 문구와 줄바꿈이 붙던 수정도 포함됩니다. 자세한 내용은 [v0.49.0 릴리스 노트](https://github.com/hkjang/muni/blob/v0.50.0/docs/releases/v0.49.0.md)에 있습니다. 이미 저장된 첨부 이름은 자동 변경되지 않습니다.

## 검증

- 전용 PostgreSQL 16에서 `go test -count=1 -json ./...`: 하위 테스트 포함 702 PASS, 외부 코퍼스·수동 출력 4 SKIP. HTTP API 테스트는 SKIP 없이 통과했습니다.
- `make test`: Go 테스트와 프런트 42파일·297건 통과.
- `go vet ./...`, Go 포맷 검사, placeholder 검사, `git diff --check` 통과.
- `npm run build`, `CGO_ENABLED=0 go build -trimpath`, Docker 이미지 빌드 통과.
- 브라우저 E2E와 실제 운영 배포는 이번 릴리즈 준비에서 실행하지 않았습니다. 배포용 archive는 태그 푸시 워크플로가 생성합니다.

## 업그레이드

기존 DB와 환경변수를 그대로 사용합니다. 이미지를 불러온 뒤 사용하는 Compose 파일의 `image`를 `muni:v0.50.0`으로 지정하세요. 저장소의 예제는 `muni:v0.1.0`을 가리키므로 운영 버전에 맞춰 변경해야 합니다.

```bash
gzip -dc muni-v0.50.0.tar.gz | docker load
# compose.example.yaml의 image를 muni:v0.50.0으로 지정합니다.
docker compose -f compose.example.yaml --env-file .env up -d
```

배포 담당자는 먼저 검증 환경에서 `/healthz`·`/readyz`와 ZIP 내보내기를 확인한 뒤 운영에 적용하세요. 두 상태 점검이 실패하거나 경계별 문서 수·안내가 위 표와 다르면 배포를 중단하고, 보관한 이전 이미지 태그로 되돌려 같은 Compose 명령을 실행합니다. 이번 변경에 따른 DB 되돌리기는 필요하지 않습니다.

배포 후 첫 주에는 ZIP 내보내기 누락 안내 관련 문의를 확인합니다. 확인 기준은 정확히 2000건인 내보내기에서 잘못된 생략 안내가 0건이고, 초과 시에는 안내가 유지되는 것입니다. 실제 운영 적용과 사후 관찰은 배포 담당자가 수행합니다.

## 오프라인 설치

워크플로가 게시할 릴리스 asset에는 `muni:v0.50.0` Docker 이미지가 포함됩니다.

```bash
gzip -dc muni-v0.50.0.tar.gz | docker load
docker image inspect muni:v0.50.0

# compose.example.yaml의 image를 muni:v0.50.0으로 지정합니다.
cp .env.example .env
# .env의 네 값을 운영 환경에 맞게 변경합니다.
docker compose -f compose.example.yaml --env-file .env up -d
```

애플리케이션이 반드시 필요로 하는 런타임 환경변수는 다음 네 개입니다.

| 환경변수                   | 설명                                 |
| -------------------------- | ------------------------------------ |
| `POSTGRES_DSN`             | PostgreSQL 접속 문자열               |
| `BOOTSTRAP_ADMIN`          | 최초 관리자 아이디 또는 이메일       |
| `BOOTSTRAP_ADMIN_PASSWORD` | 최초 관리자 비밀번호(12자 이상)      |
| `ENCRYPTION_KEY`           | base64로 인코딩한 32-byte master key |

PDF 변환 동작만 조정하는 선택 환경변수가 두 개 있습니다.

| 환경변수               | 기본값    | 설명                                       |
| ---------------------- | --------- | ------------------------------------------ |
| `MUNI_CHROMIUM_PATH`   | 자동 탐색 | PDF Export에 사용할 headless 브라우저 경로 |
| `MUNI_PDF_CONCURRENCY` | `2`       | 동시에 실행할 Chromium 프로세스 수(1~32)   |

## 운영 정보

- 서비스 포트: `8080`
- Liveness: `/healthz`
- Readiness: `/readyz`
- REST API: `/api/v1`
- 공개 링크: `/s/{token}` — 인증 없이 열리는 유일한 화면입니다
- 문서 넘기기: `/handoff?source&claim` (받는 쪽, 세션 필요, 허용 목록의 오리진만) · `/api/v1/handoff/claims/{claim}` (내주는 쪽, 인증 없이 5분짜리 단일 사용 표로만 열립니다)
- 메일 알림: 사내 SMTP 릴레이로 1분마다 배경 발송, 기본 꺼짐 · 발송 기록 `/api/v1/admin/mail/deliveries` (관리자 인증 필요)
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- Momento 프록시: `/momento/*` — 방문 추적을 Momento 로 켜 둔 동안에만 답합니다
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.50.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.50.0`

```bash
sha256sum muni-v0.50.0.tar.gz
```

태그 푸시 후 GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 수행하고 이 asset을 게시합니다. 게시 완료 여부는 해당 워크플로에서 확인하세요.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.50.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.50.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.50.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.50.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.50.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.48.0...v0.50.0)
