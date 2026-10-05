## Data Works v0.9.68

### 주요 변경 사항
- **DB 조회 실패를 데이터 없음으로 숨기던 퍼블리시 게이트·증거 팩·권한 walk 3건 수정 (v0.9.68)**: 릴리즈 검증의 web 게이트가 깨져 있던 2026-10-01~10-04 기간에 main 으로 올라가지 못한 검증 완료 수정 3건을 원 커밋 그대로 되살렸다. (1) 액션 센터는 `dataWorksPublishGate` 의 오류를 `err == nil` 조건으로 버려, 평가하지 못한 게이트를 "출시 막힘 아님" 으로 집계했다 — 운영자가 보는 그 한 화면에서 `blocked_launches` 가 0 인 동안 `GET .../publish-gate` 와 `POST .../publish` 는 같은 조회 실패에 이미 500 을 돌려주고 있었다. 이제 v0.9.66 의 인벤토리 조회와 같이 `500 publish_gate_failed` 로 응답 전체를 실패시킨다. (2) `buildProductEvidencePack` 은 `LatestProductDefinition`·`LatestProductRiskReview`·`LatestProductPOCPlan` 의 오류를 버려 조회 실패를 출처 부재와 똑같이 취급했고, `refreshProductEvidencePack` 이 그렇게 짧아진 팩을 기존 행을 지우고 삽입하는 `ReplaceProductEvidencePack` 에 넘겨 `POST .../evidence` 가 200 으로 끝났다 — 출시 판단의 근거인 definition version·risk basis·PoC success metric 이 아무 보고 없이 거버넌스 기록에서 영구히 사라졌다. 같은 세 출처를 읽는 `buildEvidencePackJSON` 은 이미 오류를 반환하고 있어 두 경로가 같은 값을 두고 어긋나 있었다. 이제 오류를 반환해 저장된 팩을 건드리지 않고 `500 evidence_refresh_failed`(즉석으로 팩을 만드는 읽기 경로는 `evidence_failed`)로 보고하며, 성공 경로가 만드는 행은 이전과 같다. (3) `usableEntitlement` 는 `GetContractScope` 의 조회 실패를 "해당 계약 없음"·"상품 불일치" 와 한 `continue` 로 묶어, 사용 가능한 후보가 없으면 `candidates[0]` 의 사유로 떨어져 다른 후보의 조회 실패를 `403 contract_scope_inactive` 로 바꿨다 — 후보가 하나인 경로는 똑같은 실패에 이미 `500 contract_lookup_failed` 를 돌려준다. 이제 첫 조회 실패를 따로 보관한 채 계속 걸어, 모든 검사를 통과하는 후보가 하나도 없을 때만 그 오류를 반환한다(살아 있는 다른 권한이 있는 호출자는 영향이 없고 오류는 버려진다). 새 500 은 사용량 미터링 defer 가 설치되기 전·`contractKey` 가 빈 상태에서 나가므로 과금 행이 생기지 않고 errCode 배선도 그대로다. 세 결함 모두 실제 SQLite·`store.SQLStore`·`NewServer.Routes()` HTTP 회귀 테스트로 덮었고(3스위트 9사례), 각 수정의 프로덕션 파일만 v0.9.67 상태로 되돌리면 해당 스위트만 실패하고 나머지 둘은 통과함을 건별로 확인했다. `docs/OPERATIONS.md` 에 세 오류 코드의 운영 대응 문단을 더했다. 퍼블리시 게이트 조립 내부의 선택적 조회(SLA·비용·품질 결과·리스크 리뷰) 오류 처리는 이번 범위 밖이다. 스키마 변경은 없다. 릴리즈 검증 실행: 저장소 루트에서 `npm run lint`(eslint 무경고)·`npm test --silent`(11파일 36사례 전부 통과)·`npm run build`(tsc -b && vite build) 통과, `go build ./...`·`go vet ./...`·`go test ./... -count=1` 전체 통과, `go run ./cmd/api-surface-audit` gap 0(550 routes / 612 OpenAPI paths). `dataworks:v0.9.68` 이미지를 `dataworks-v0.9.68.tar.gz` 단일 오프라인 GitHub Release asset으로 제공한다.


### 배포 파일
| 파일 | 설명 |
|------|------|
| dataworks-v0.9.68.tar.gz | 오프라인 적재 가능한 Data Works Docker 이미지 (linux/amd64) |

### 빠른 시작
```bash
# 이미지 로드
gunzip -c dataworks-v0.9.68.tar.gz | docker load

# 실행
docker run -d --name dataworks --restart=always \
  -p 8080:8080 \
  -e POSTGRES_DSN='postgres://dataworks:change-me@postgres:5432/dataworks?sslmode=disable' \
  -e BOOTSTRAP_ADMIN='admin@dataworks.local' \
  -e BOOTSTRAP_ADMIN_PASSWORD='change-me' \
  -e ENCRYPTION_KEY='replace-with-64-hex-characters' \
  dataworks:v0.9.68
```