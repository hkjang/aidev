# 회차 노트 2026-10-03-070753-dataworks-improve — dataworks
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:07] base pinned — main@98c5081
- [러너 07:07] autonomy release — 

## 구현 노트
- 무엇/왜: `buildProductEvidencePack` 이 상품정의·리스크점검·PoC계획 조회 오류를 `_` 로 버려서, `refreshProductEvidencePack` → `ReplaceProductEvidencePack`(DELETE 후 INSERT)이 읽지 못한 증거 행을 영구 삭제하고 `POST …/evidence` 가 200 을 돌려줬다. 세 오류를 반환하게 바꿔 실패 시 교체 전에 끊고 500 `evidence_refresh_failed`(GET 즉석 생성 경로는 `evidence_failed`)를 낸다. 프로덕션 1파일(`internal/proxy/admin_dataworks_ops.go`).
- 확신 없는 곳·검증 못 한 것: (1) `admin_factory.go:310` 의 `_ = s.refreshProductEvidencePack(...)` 은 시그니처 변경만 받고 테스트하지 않았다 — 이제 출처 조회 실패 시 그 경로가 증거를 아예 쓰지 않고 넘어간다(전에는 짧은 팩을 썼다). factory 정의 생성 직후라 저장된 증거가 없으므로 손실은 없다고 판단했지만 실행으로 확인하진 않았다. (2) PostgreSQL 에서 `ALTER TABLE … RENAME` 식 재현은 안 했다(SQLite 만). (3) 웹 워크벤치의 증거 화면이 500 을 어떻게 표시하는지는 확인하지 않았다 — web 변경·web 검증 없음.
- 일부러 하지 않은 것: 퍼블리시 게이트 조립(admin_dataworks.go:1750·1758·1769·1775)의 같은 종류 오류 버리기는 게이트 의미가 바뀔 수 있어 손대지 않고 보류 목록에 남겼다. `usableEntitlement`(dataworks_runtime.go:253)도 같은 이유 + 결정적 재현 방법 미확보로 제외. web/ 는 전혀 건드리지 않았다(1d63b7c·5ed6876·f927cea 세 건이 main 에 머지되지 않아 그 방향이 반려됐을 수 있다고 판단).
- 다음 역할이 조심할 것: 새 테스트 `internal/proxy/admin_dataworks_evidence_test.go` 는 실제 SQLite 파일 DSN 을 두 번 열어 `ALTER TABLE … RENAME` 으로 장애를 만든다. 서브케이스가 중간에 죽으면 `unavailable_source` 라는 이름의 테이블이 남을 수 있으나 DB 는 `t.TempDir()` 안이라 다른 테스트에 영향 없다. 서브케이스를 `-run` 으로 하나만 골라 돌리면 cleanup 의 복구 단언까지 같이 돈다.
- [러너 07:18] verify failed — 실패한 검증: cd web && npm test --silent (exit 1)
