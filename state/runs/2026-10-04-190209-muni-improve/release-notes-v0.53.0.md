워크스페이스를 ZIP으로 내보낼 때 **제목이 대소문자로만 다른 문서 둘이 압축을 푸는 쪽에서 한 파일이 되어 하나가 조용히 사라질 수 있던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 대소문자로만 다른 항목 이름이 ZIP에 둘 다 들어가던 것

워크스페이스 ZIP은 제목이 같은 문서 둘이 한 파일이 되지 않도록 뒤쪽에 `(2)`를 붙입니다. 그 중복 검사가 쓰는 장부는 이름을 **바이트 그대로** 비교했습니다. 그래서 제목이 `Report`와 `report`인 문서 둘은 서로 다른 이름으로 판정되어 ZIP 안에 두 항목으로 들어갔습니다.

압축을 푸는 것은 내려받은 기계이고, 배포 대상인 **Windows에서(그리고 기본 설정의 macOS에서) `Report.md`와 `report.md`는 같은 파일 이름입니다.** 항목을 찾은 순서대로 쓰는 도구에서는 뒤 항목이 앞 항목을 덮어씁니다. ZIP 안에는 문서 둘이 들어 있는데 풀고 나면 파일이 하나뿐인 — `(2)` 접미사가 막으려고 쓰인 바로 그 분실입니다.

| 워크스페이스의 문서 제목 | ZIP 항목 (이전) | 풀었을 때 (Windows) | ZIP 항목 (지금) |
| --- | --- | --- | --- |
| `보고서` / `보고서` | `보고서.md`, `보고서 (2).md` | 파일 둘 | 그대로 |
| `Report` / `report` | `Report.md`, `report.md` | **파일 하나** | `Report.md`, `report (2).md` |

### 폴더 이름이 대소문자로만 갈린 경우도 같은 문제였습니다

항목 이름은 폴더 경로를 포함한 전체 경로입니다. 같은 부모 아래 `Report` 폴더와 `report` 폴더가 있고 각각 `회의록` 문서가 있으면, 이전에는 `Report/회의록.md`와 `report/회의록.md`가 되어 역시 둘 중 하나가 남지 않았습니다. 중복 검사가 전체 경로를 접어서 비교하므로 이 경우도 함께 구분됩니다.

폴더 이름은 같은 부모 아래 중복을 막는 검사가 없어 두 폴더를 실제 API로 만들 수 있습니다. 이번 수정이 지키는 것은 **그 안의 문서가 사라지지 않는 것까지**이며, 폴더 이름 중복 자체는 그대로 둡니다.

### 사용자가 보는 이름은 바뀌지 않습니다

접는 것은 **중복 검사용 비교 키에만** 적용됩니다. ZIP에 실제로 쓰는 이름과 사용자가 푼 뒤 보는 파일 이름은 제목에 적힌 대소문자를 글자 그대로 유지합니다 — `README`가 `readme`로 바뀌는 일은 없습니다.

| 제목 | ZIP 항목 이름 |
| --- | --- |
| `Report` | `Report.md` |
| `report` | `report (2).md` |

안내 파일 `목록.md`의 이름 예약도 같은 헬퍼를 지나게 해 예약과 조회가 한 규칙을 씁니다. `목록.md`는 소문자화가 항등이라 지금 동작은 달라지지 않지만, 두 자리가 서로 다른 규칙을 쓰면 그 이름을 ASCII로 바꾸는 순간 조용히 깨집니다.

내보내기 형식 세 가지(`md`·`html`·`txt`)는 모두 같은 중복 검사를 지나므로 이번 수정이 함께 적용됩니다.

### 남아 있는 것

- **이번 변경이 지키는 것은 항목 이름의 유일성, 즉 문서가 사라지지 않는 것까지입니다.** 대소문자로만 다른 폴더 둘은 Windows에서 여전히 한 디렉터리로 합쳐지므로 **사용자가 보는 폴더 구조는 뭉개진 채로 남습니다** — 안쪽 문서는 이제 이름이 갈려 모두 살아남습니다. 폴더 구조 보존은 이번 릴리스가 약속하지 않습니다.
- **안내 파일 `목록.md`의 각 줄은 ZIP 항목 이름을 적힌 대소문자 그대로 적습니다.** 그래서 ZIP 안에서는 목록과 항목이 정확히 일치하지만, 위처럼 폴더 둘이 Windows에서 한 디렉터리로 합쳐지면 목록에 적힌 폴더 이름과 디스크에 풀린 폴더 이름의 대소문자가 어긋날 수 있습니다. 이전에도 그랬고 분실은 아닙니다.
- **실제 Windows·macOS에서 뒤 항목이 앞 항목을 덮는 것은 이번 릴리스 준비에서 재현하지 않았습니다.** 리눅스 파일 시스템은 대소문자를 구분하므로 거기서는 두 파일이 됩니다. 검증이 증명한 것은 "ZIP 항목 이름이 대소문자를 무시해도 서로 유일하다"까지이며, 덮어쓰기가 사라진다는 것은 그 성질에서 따라오는 결론입니다.
- **둘 중 어느 쪽이 `(2)`를 받는지는 고정하지 않았습니다.** 문서 조회 정렬이 제목 동률에서 순서를 보장하지 않기 때문이고, 그래서 테스트도 "한쪽만 `(2)`를 받는다"까지만 확인합니다.
- `safeFilename`, `safeFolderSegment`, 2000건 한도, 감사 기록, 나머지 네 내려받기 경로는 **그대로 두었습니다.** 저장되는 값은 바뀌지 않으므로 이미 만들어 둔 문서와 폴더에 다시 손댈 필요가 없습니다.

## 검증

- 전용 PostgreSQL 16에서 `go test -count=1 -json ./...`: 하위 테스트 포함 **721 PASS**, 외부 코퍼스·수동 출력 4 SKIP, 실패 0. HTTP API 테스트는 SKIP 없이 통과했습니다.
- 실제 라우트로 검증을 추가했습니다: `Report`·`report` 문서 둘과 `Report`·`report` 폴더 둘을 인증 API로 만든 뒤 ZIP 응답의 항목 이름을 모두 읽어, **소문자로 접었을 때 겹치는 항목이 없는지**와 두 제목·두 폴더가 적힌 대소문자를 그대로 유지하는지 확인합니다. 고치기 전에 이 검증이 `entries "Report.md" and "report.md" differ only in case`로 실패하는 것을 먼저 확인했습니다.
- 중복 검사 단위 검증도 추가했습니다: `Report`/`report`가 `Report.md`/`report (2).md`가 되고, 경로의 폴더만 대소문자로 갈린 경우도 같은 충돌로 판정되는지 확인합니다.
- 프런트 `npm run lint`(`tsc -b`), `npm test` 42파일·297건, `npm run build` 통과.
- `go vet ./...`, Go 포맷 검사, placeholder 검사, `git diff --check` 통과.
- `CGO_ENABLED=0 go build -trimpath`, Docker 이미지 빌드 통과.
- 브라우저 E2E, 실제 Windows·macOS 압축 풀기, 실제 운영 배포는 이번 릴리즈 준비에서 실행하지 않았습니다. 배포용 archive는 태그 푸시 워크플로가 생성합니다.

## 업그레이드

기존 DB와 환경변수를 그대로 사용합니다. 이미지를 불러온 뒤 사용하는 Compose 파일의 `image`를 `muni:v0.53.0`으로 지정하세요. 저장소의 예제는 `muni:v0.1.0`을 가리키므로 운영 버전에 맞춰 변경해야 합니다.

```bash
gzip -dc muni-v0.53.0.tar.gz | docker load
# compose.example.yaml의 image를 muni:v0.53.0으로 지정합니다.
docker compose -f compose.example.yaml --env-file .env up -d
```

배포 담당자는 먼저 검증 환경에서 `/healthz`·`/readyz`와 워크스페이스 ZIP 내보내기를 확인한 뒤 운영에 적용하세요. 제목이 대소문자로만 다른 문서가 없는 워크스페이스를 내보내 **항목 이름이 이전과 한 글자도 다르지 않은지** 함께 확인합니다. 두 상태 점검이 실패하거나 평범한 워크스페이스의 ZIP 항목 이름이 이전과 다르면 배포를 중단하고, 보관한 이전 이미지 태그로 되돌려 같은 Compose 명령을 실행합니다. 이번 변경에 따른 DB 되돌리기는 필요하지 않습니다.

배포 후 첫 주에는 워크스페이스 내보내기 관련 문의를 확인합니다. 확인 기준은 평범한 워크스페이스의 항목 이름이 0건 바뀌고, Windows에서 ZIP을 풀었더니 문서가 비는 것 같다는 문의가 더 오지 않는 것입니다. 실제 운영 적용과 사후 관찰은 배포 담당자가 수행합니다.

## 오프라인 설치

워크플로가 게시할 릴리스 asset에는 `muni:v0.53.0` Docker 이미지가 포함됩니다.

```bash
gzip -dc muni-v0.53.0.tar.gz | docker load
docker image inspect muni:v0.53.0

# compose.example.yaml의 image를 muni:v0.53.0으로 지정합니다.
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

- 파일: `muni-v0.53.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.53.0`

```bash
sha256sum muni-v0.53.0.tar.gz
```

태그 푸시 후 GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 수행하고 이 asset을 게시합니다. 게시 완료 여부는 해당 워크플로에서 확인하세요.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.53.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.53.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.53.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.53.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.53.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.52.0...v0.53.0)
