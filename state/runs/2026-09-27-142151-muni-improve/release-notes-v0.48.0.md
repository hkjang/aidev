워크스페이스를 ZIP 으로 내보낼 때 **제목이 `목록` 인 문서가 안내 파일에 덮여 사라지던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 안내 파일 `목록.md` 와 제목이 `목록` 인 문서가 같은 이름을 쓰던 것

워크스페이스 ZIP 은 같은 제목의 문서 둘이 한 파일이 되지 않도록 항목 이름을 추적합니다. `uniqueEntryName` 은 이미 쓴 이름을 기억해 두 번째 문서를 `제목 (2).md` 로 보내는데, **이 추적을 거치지 않는 항목이 정확히 하나 있었습니다** — 아카이브 맨 끝에 쓰이는 안내 파일 `목록.md` 입니다. 문자열이 그 자리에 직접 적혀 있었고, 문서 순회는 그런 이름이 예약돼 있다는 것을 몰랐습니다.

그래서 **워크스페이스 루트에 제목이 `목록` 인 문서가 있고 md 로 내보내면** ZIP 안에 같은 이름의 항목이 둘 들어갑니다.

```
목록.md        ← 제목이 「목록」 인 문서
회의/목록.md
목록.md        ← 아카이브 안내 파일
```

`archive/zip` 은 이런 아카이브를 거부하지 않고, **항목을 찾는 순서대로 풀어쓰는 압축 해제 도구에서는 두 파일이 있어야 할 자리에 한 파일만 남습니다.** `uniqueEntryName` 이 막으려고 쓰인 그 조용한 문서 분실이, 그것을 거치지 않는 이름 하나에서 그대로 일어났습니다. 회의록이나 색인처럼 `목록` 은 실제로 흔한 제목입니다.

이제 이름을 상수 `workspaceManifestName` 으로 두고 **문서 순회를 시작하기 전에 예약합니다.** 그 제목의 문서는 다른 중복과 똑같이 `목록 (2).md` 로 가고, 안내 파일은 읽는 쪽이 기대하는 이름에 남습니다 — 안내 파일 안의 목록도 실제 항목 이름을 적으므로 `목록 (2).md` 로 기록됩니다.

**바뀌지 않은 것:** 예약하는 이름은 아카이브 루트의 `목록.md` 하나뿐입니다.

- **폴더 안의 `목록` 은 영향이 없습니다.** 추적 키가 폴더를 포함한 전체 경로라 `회의/목록.md` 는 애초에 충돌이 아니었고, 지금도 그대로 나갑니다.
- **휴지통도 그대로입니다.** 지워진 문서는 `휴지통/` 아래로 가므로 루트의 이 이름에 닿지 않습니다.
- **html·txt 내보내기는 한 바이트도 달라지지 않습니다.** 문서 항목의 확장자는 항상 요청한 `format` 이라, `목록.html` 과 `목록.txt` 는 안내 파일 이름이 아닙니다.
- `uniqueEntryName` 의 시그니처와 `safeFilename` 의 치환·100룬 절단(v0.45.0), 내려받기 헤더(v0.46.0·v0.47.0)는 손대지 않았습니다.

검증은 **실제 라우트(`GET /api/v1/workspaces/{id}/export.zip?format=md&trash=true`)로 루트 문서 `목록` 과 폴더 `회의` 안의 `목록` 을 넣고 ZIP 항목 이름을 되읽는** live 테스트입니다. 고치기 전에 그 테스트가 `two entries are called "목록.md"` 로 먼저 실패하는 것을 확인했고, 예약 한 줄만 되돌리자 그 하나만 다시 실패(나머지 234건은 계속 통과)해 인과를 확정했습니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 화면은 손대지 않았고, 서버는 워크스페이스 ZIP 의 항목 이름을 정하는 자리만 v0.47.0 과 다릅니다.

```bash
gzip -dc muni-v0.48.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**이미 내려받은 ZIP 파일은 그대로입니다.** 루트에 제목이 `목록` 인 문서가 있는 워크스페이스를 md 로 내보낸 적이 있다면, 그 아카이브에는 두 항목 중 하나만 풀릴 수 있으므로 **다시 내보내는 것을 권합니다.** 그 밖의 워크스페이스에서는 ZIP 의 내용과 항목 이름이 v0.47.0 과 완전히 같습니다.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.48.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.48.0.tar.gz | docker load
docker image inspect muni:v0.48.0

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

- 파일: `muni-v0.48.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.48.0`

```bash
sha256sum muni-v0.48.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.48.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.48.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.48.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.48.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.48.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.47.0...v0.48.0)
