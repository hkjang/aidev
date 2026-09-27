## Data Works v0.9.64

### 주요 변경 사항
- **규제 추적 재생성이 승인 이력의 `expires_at` 을 지워 만료된 승인이 퍼블리시 게이트를 영구히 다시 열던 문제 수정 (v0.9.64)**: `syncApprovalTracesFromRegulatoryTrace`(`internal/proxy/admin_dataworks_ops.go`)는 결정적 id `appr_<product_key>_<step>` 로 `store.ApprovalTrace` 를 UPSERT 하면서 `ExpiresAt` 을 채우지 않았고, `SQLStore.UpsertApprovalTrace` 의 `ON CONFLICT(id) DO UPDATE SET … expires_at = excluded.expires_at`(`internal/store/dataworks_governance.go`)이 기존 값을 무조건 빈 문자열로 덮었다. 그래서 관리자가 `POST …/approvals`(워크벤치 "승인 결정 수정" 이 같은 id 를 실어 보낸다)로 기록한 만료일이 규제 추적 재생성 한 번에 사라지고, `bestApprovalStatus` 는 빈 `expires_at` 을 "만료 없음" 으로 읽어 만료로 차단돼야 할 승인이 퍼블리시 게이트를 영구히 다시 열었다. 루프 전에 `ListApprovalTraces` 를 한 번 호출해 `id → ExpiresAt` 맵을 만들고 그 값을 실어 준다. 조회 오류는 빈 만료일을 쓰는 대신 sync 를 건너뛴다(빈 값 쓰기가 바로 이 결함이고 게이트를 느슨하게 하지 않는 방향이라서 — 주석에 판단 근거 명시). `status`·`decided_by`·`notes`·`evidence_ref` 는 종전대로 덮고 `expires_at` 만 보존하며, `UpsertApprovalTrace` 의 UPSERT 컬럼 목록은 건드리지 않아 `POST …/approvals` 로 만료일을 비우는 정상 경로는 그대로다. 검증: 실제 `store.SQLStore`(SQLite) 와 `NewServer(...).Routes()` HTTP 경로만 사용하는 표 테스트 3건을 추가했다 — 과거 만료(`2020-01-01T00:00:00Z`) 사례는 재생성 전·후 모두 `publish` 409 + `"error":"publish gate blocked"` 및 `publish_gate.missing_approvals`·`blocked_reasons` 에 `legal` 유지, 미래 만료(`2030-01-01T00:00:00Z`) 대조 사례는 전·후 모두 200 이고 `GET …/approvals` 의 `expires_at` 이 트림·재포맷 없이 원문 그대로이며, 재생성에서 `decision` 을 `approved`→`rejected` 로 바꾼 사례는 `status`·`decided_by`·`notes`·`evidence_ref` 가 모두 새 값으로 갱신되고 `expires_at` 만 보존된다. 프로덕션 파일만 되돌린 변이에서 새 테스트 3건만 실패하고 기존 `Approval|DataWorks|RegulatoryTrace` 테스트는 전부 통과함을 실행으로 확인했다. `go build ./...`·`go vet ./...`·`go test ./...` 전체 16패키지 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths). `docs/OPERATIONS.md` 4절에 만료일 보존 규칙을 명시했다. `dataworks:v0.9.64` 이미지를 `dataworks-v0.9.64.tar.gz` 단일 오프라인 GitHub Release asset으로 배포.

### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.64.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.64.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.64
```
