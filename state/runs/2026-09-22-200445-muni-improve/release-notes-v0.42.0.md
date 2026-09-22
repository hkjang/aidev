마크다운 표를 가져올 때 **끝 구분자를 생략한 행의 마지막 셀에서 `\|` 의 파이프가 사라지던 것**을 고쳤습니다. 화면은 바뀌지 않았고, 마이그레이션도 설정 변경도 없습니다.

## 고친 것

### 끝 구분자 없는 표 행의 마지막 셀에서 사라졌던 파이프

GFM 은 표 행의 마지막 `|` 를 생략하도록 허용합니다. muni 의 표 파서는 이스케이프를 훑기 **전에** 끝 파이프를 무조건 지우고 있었기 때문에, 끝 구분자를 생략한 행의 마지막 셀이 `\|` 로 끝나면 셀 내용인 파이프까지 함께 지워지고 역슬래시만 남았습니다 — `| 좌 | 우 | 가 | 나\|` 의 마지막 셀이 `나|` 가 아니라 `나\` 가 되었습니다. 검색 텍스트(`content_text`)도 같은 값으로 채워져, 오류 없이 조용히 글자가 망가졌습니다.

이제 끝 파이프를 미리 지우지 않고 문자열 전체를 기존 이스케이프 규칙 그대로 훑되, **마지막으로 소비한 것이 이스케이프되지 않은 `|` 였을 때만** 끝의 빈 셀을 만들지 않습니다. 끝 파이프는 이스케이프되지 않았을 때에만 구분자입니다.

마크다운 표를 읽는 세 자리 — `.md`·`.markdown` 업로드로 새 문서를 만들기, 이미 열어 둔 문서에 파일을 끼워 넣기, v0.39.0 의 문서 넘기기로 마크다운을 받기 — 가 모두 같은 파서를 쓰므로 세 자리가 함께 고쳐졌습니다.

**바뀌지 않은 것:** 끝 구분자가 있는 행, 헤더 행, 구분자 행(`| --- | :-: |`), 빈 마지막 셀(`| a | |` 은 여전히 두 칸)은 한 글자도 달라지지 않습니다. muni 의 마크다운 내보내기는 항상 행 끝에 ` |` 를 붙이므로 내보내기→가져오기 왕복도 그대로입니다. `\\|`(역슬래시 두 개 뒤의 파이프)는 전과 똑같이 "이스케이프된 파이프" 로 읽습니다 — GFM 의 역슬래시 이스케이프 전면 준수는 이번 범위가 아닙니다. `.docx`·`.hwp`·`.hwpx`·`.html`·PDF 의 표는 이 파서를 지나지 않습니다.

표-주도 단위 테스트(양쪽 구분자·끝 구분자 없음·구분자 없음·셀 안의 이스케이프된 파이프·빈 마지막 셀·구분자 행·정렬 구분자 행·빈 줄·파이프 하나)와 `renderMarkdown`→`markdownDocument` 왕복 테스트, PostgreSQL 을 띄운 live 테스트(`.md` 업로드에서 마지막 셀과 `content_text` 가 파이프를 지키는지)가 이것을 지킵니다.

## 업그레이드

마이그레이션과 설정 변경은 필요하지 않습니다. 화면은 손대지 않았고, 서버는 마크다운 표 행의 끝 구분자 판정만 v0.41.0 과 다릅니다.

```bash
gzip -dc muni-v0.42.0.tar.gz | docker load
docker compose -f compose.example.yaml --env-file .env up -d
```

**이미 파이프가 사라진 문서는 그대로입니다.** 이 고침은 가져오거나 넘겨받는 순간에만 적용되므로, 이전에 만들어져 역슬래시만 남은 셀은 편집기에서 고쳐 주세요.

## 오프라인 설치

릴리스 asset 에는 `muni:v0.42.0` Docker 이미지가 포함되어 있습니다.

```bash
gzip -dc muni-v0.42.0.tar.gz | docker load
docker image inspect muni:v0.42.0

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
- OpenAPI: `/api/openapi.yaml` — 실제 라우트와 일치하며, 테스트가 그것을 지킵니다
- Prometheus: `/metrics` — 관리자 인증 필요
- MCP: `/mcp`
- Momento 프록시: `/momento/*` — 방문 추적을 Momento 로 켜 둔 동안에만 답합니다
- 지원 DB: PostgreSQL 15 이상
- 이미지에는 PDF Export용 Chromium과 Noto CJK 글꼴이 포함되어 있습니다.

## 릴리스 파일 검증

- 파일: `muni-v0.42.0.tar.gz`
- 크기: (릴리스 후 기록)
- SHA-256: (릴리스 후 기록)
- 내부 이미지 태그: `muni:v0.42.0`

```bash
sha256sum muni-v0.42.0.tar.gz
```

GitHub Actions가 이미지 빌드, archive 생성, 내부 이미지 태그 검증을 완료한 뒤 이 asset을 게시했습니다.

## 문서

- [설치 및 운영 안내](https://github.com/hkjang/muni#readme)
- [사용자 가이드](https://github.com/hkjang/muni/blob/v0.42.0/docs/USER_GUIDE.md)
- [관리자 가이드](https://github.com/hkjang/muni/blob/v0.42.0/docs/ADMIN_GUIDE.md)
- [운영 안내](https://github.com/hkjang/muni/blob/v0.42.0/docs/OPERATIONS.md)
- [아키텍처](https://github.com/hkjang/muni/blob/v0.42.0/docs/ARCHITECTURE.md)
- [MCP 사용법](https://github.com/hkjang/muni/blob/v0.42.0/docs/MCP.md)
- [전체 변경 내역](https://github.com/hkjang/muni/compare/v0.41.0...v0.42.0)
