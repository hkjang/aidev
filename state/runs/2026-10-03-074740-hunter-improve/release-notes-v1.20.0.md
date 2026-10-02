## Hunter v1.20.0

폐쇄망 반입용 서비스 Docker 이미지입니다. PostgreSQL은 사내에 별도 준비합니다.

- 이미지: `hunter:v1.20.0`
- 유일한 첨부 자산: `hunter-v1.20.0.tar.gz`
- 플랫폼: `linux/amd64`
- SHA-256: `릴리즈 워크플로가 빌드한 아카이브의 해시로 기록됩니다`

### 발견 건 일괄 변경 — 조치 기한 폼이 서버가 받을 수 있는 값만 제출합니다

- **같은 변경의 담당자·상태까지 되돌려졌습니다**: 일괄 변경 폼의 `findingBulkPatch`(`web/src/finding-bulk-state.ts`)는 `new Date(...).toISOString()` 결과를 검사 없이 `due_date` 로 보냈습니다. ECMA-262 는 UTC 순간이 0000~9999 를 벗어나면 확장 연도(`+YYYYYY`/`-YYYYYY`)를 내놓지만 서버 `validateFindingOpsResource`(`internal/app/finding_ops.go`)는 `time.Parse(time.RFC3339, s)` 로만 읽어 부호 없는 네 자리 연도만 받으므로, 요청 전체가 `400` 이 되어 같은 변경에 담은 담당자·진행 상태까지 취소됐습니다.
- **브라우저 시간대가 수락·거절을 갈랐습니다**: 타이핑할 수 있고 사양상 유효한 `datetime-local` 값 `9999-12-31T23:59` 은 UTC 동쪽에서는 범위 안이지만 서쪽 시간대에서는 유한한 값인 채로 `+010000-01-01T04:59:00.000Z` 로 직렬화됩니다. 같은 입력이 사용자의 시간대에 따라 저장되거나 전체 취소됐습니다.
- **보내는 문자열을 검사합니다**: 이제 `input.due` 가 아니라 실제로 제출하는 문자열이 네 자리 연도로 시작하는지 확인하고, 아니면 요청을 만들지 않고 기한 칸 옆에 한국어로 안내합니다. 범위 안의 기한이 만드는 요청 본문은 한 글자도 달라지지 않습니다.
- **공유 벡터로 고정합니다**: `internal/app/testdata/finding-bulk-due-date.json` 이 (시간대, 입력) 쌍마다 보내는 문자열과 서버 판정을 적고, Go 는 `validateFindingBulk` 로 PostgreSQL 없이 확인하며 화면 시험은 `TZ` 를 바꿔 `accepted` 로 표시된 문자열만 제출하는지 단언합니다.

서버 계약은 바꾸지 않았습니다. `due_date` 는 그대로 네 자리 연도 RFC3339 만 받고 API 필드·권한 검사·감사 기록, 최대 100개 일괄 변경과 각 항목 `updated_at` 비교, 한 항목 충돌 시 전체 롤백 규칙도 그대로입니다. 프로덕션 변경은 화면 파일 한 개입니다.

네 환경변수, 일반 PostgreSQL과 서비스 Docker 이미지 하나의 배포 조건을 유지합니다. 최종 게시 커밋의 CI·공개 파일 검증 결과는 실제 완료 후 이 본문에 별도로 기록합니다.

[릴리즈 노트](https://github.com/hkjang/hunter/blob/main/docs/release-v1.20.0.md) · [검증 기록](https://github.com/hkjang/hunter/blob/main/docs/validation.md)

### 설치

```sh
docker load -i hunter-v1.20.0.tar.gz
docker compose up -d
```

[설치 및 관리자 가이드](https://hkjang.github.io/hunter/guides/admin-guide.html) · [사용자 가이드](https://hkjang.github.io/hunter/guides/user-guide.html)

GitHub가 자동 표시하는 소스 코드 다운로드는 릴리즈 첨부 자산과 별개입니다.
