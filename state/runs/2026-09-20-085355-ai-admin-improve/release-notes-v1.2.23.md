## v1.2.23 — 2026-09-20

소유자 계정이 잠기거나 비활성화된 API 키의 회전을 발급과 같은 계약으로 거부하도록 바로잡은 패치 릴리스입니다.

- `POST /api/v1/keys/{id}/rotate`가 키의 status·만료·scope·호출자 권한은 검사하면서 소유자(`api_key.user_id`) 계정 상태는 보지 않던 문제 해결. 관리자가 `keys.manage`로 `locked`·`disabled` 계정의 키를 회전하면 기존 키가 `rotated`로 닫히고 아무도 쓸 수 없는 새 키가 발급됐으며, 고권한 키면 실행될 수 없는 승인 요청까지 접수됐습니다
- 새 키를 만들거나 승인 요청을 생성하기 전에 소유자 계정을 한 번 확인합니다. `active`가 아니면 409 `user_not_active`(details의 `status`에 현재 상태)로 거부하고 기존 키는 `active`로 남으며, 이 경우 승인 요청도 남지 않습니다. 계정 조회 자체가 실패한 경우는 기존대로 500 `key_rotate_failed`입니다
- 회전 승인 실행 분기는 소유자 활성 여부를 이미 검사하고 있었으나 409 `key_conflict`("원본 키의 소유자·권한·만료·상태가 변경")로 오도하던 것을 발급과 같은 409 `key_target_unavailable`로 바로잡았습니다. 요청은 대기 상태로 남아 검토자가 반려하거나 계정 상태를 복구한 뒤 다시 승인할 수 있으며, 메시지는 발급·회전 어느 쪽에서도 읽히도록 다듬었습니다
- PostgreSQL 통합 테스트(disabled·locked 각각 409·키 수 불변·기존 키 active, 계정 복구 후 201·`rotated`, 워크플로가 켜진 상태의 비활성 409·승인 요청 0건, 대기 중 비활성화 → 승인 409 `key_target_unavailable`·pending 유지·키 active, 계정 복구 후 같은 요청 승인 200·`rotated`)로 검증
- `docs/api.md` 회전 문단에 발급과 같은 소유자 계정 확인 계약을 명시

**Full Changelog**: https://github.com/hkjang/ai-admin/compare/v1.2.22...v1.2.23
