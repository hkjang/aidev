# Hunter v1.18.0

발견 건 **일괄 변경**의 담당자 200바이트 한도를 화면과 서버가 같은 문자열로 재는지 공유 벡터로 고정한 릴리즈입니다. 사용자가 보는 동작, 저장되는 값, 오류 문구는 달라지지 않으며 서버 API·권한·감사 기록과 일반 PostgreSQL·네 환경변수·서비스 Docker 이미지 하나의 기본 배포를 유지합니다.

## 사용자가 달라지는 점

사용자 화면의 동작은 달라지지 않습니다. 이번 릴리즈는 이미 맞는 판정이 다음 변경에서 어긋나지 않도록 고정하는 회귀 검사입니다.

- **화면과 서버가 같은 문자열을 셉니다.** 일괄 변경 폼의 `findingBulkPatch`(`web/src/finding-bulk-state.ts`)는 `POST /api/findings/bulk` 에 담당자 이름을 앞뒤 공백을 제거해 보냅니다. 그래서 200바이트 한도도 보내는 그 값에서 세고, 서버 `validateFindingBulk`(`internal/app/finding_bulk.go`)도 받은 그 문자열에서 `len(name)` 으로 셉니다. 표에서 복사해 뒤에 공백이 붙은 이름은 화면이 미리 거절하지 않고, 화면이 통과시킨 이름을 서버가 `400` 으로 돌려보내지도 않습니다. 어느 쪽이든 사용자는 한 번의 왕복이나 쓸 수 있는 이름을 잃습니다.
- **두 판정이 어긋나면 검사가 실패합니다.** 그 일치를 지키는 검사가 지금까지 없었고, 서버의 순수 검증 함수에는 데이터베이스 없이 도는 단위 검사도 없었습니다. 이제 `internal/app/testdata/finding-bulk-assignee.json` 의 벡터 21개(허용 11개·거절 10개)를 Go `TestFindingBulkAssigneeSharedVectors`(`internal/app/finding_bulk_validate_test.go`)와 화면 `web/tests/finding-bulk.test.mjs` 가 함께 읽고 같은 `accepted` 판정에 이르는지 확인합니다.
- **벡터가 경계를 직접 짚습니다.** 한글 200바이트 정확히·201바이트, 이모지가 걸친 200바이트 경계, 공백만 입력해 담당자를 비우는 경우, C0·DEL·C1 제어 문자와 제어 문자 바로 바깥의 `U+00A0` 이 들어 있습니다. Go 의 `strings.TrimSpace` 와 JavaScript 의 `String.prototype.trim` 이 실제로 갈리는 부호 위치도 두 가지 모두 벡터에 있습니다. `U+FEFF` 는 JavaScript 만 버리므로 화면이 보내는 값에서 사라지고, `U+0085` 는 Go 만 버리지만 두 쪽 모두 트림 전에 제어 문자로 거절하므로 그 차이가 드러나지 않습니다. 벡터 파일의 설명도 이 두 가지만 차이라고 바로잡았습니다.

## 운영 조건과 한계

담당자 한도 200바이트, 제어 문자 거절, 한 번에 최대 100개, 각 항목의 현재 권한과 `updated_at` 재검사, 한 항목만 충돌해도 전체 롤백하는 규칙은 달라지지 않습니다. 저장된 자료를 다시 해석하지 않고 관리자 설정 항목이나 API 필드를 추가하지 않았습니다. 벡터는 담당자 필드 하나를 다루며 조치 기한·상태의 검증은 이번 공유 벡터에 포함하지 않았습니다. 공유 벡터는 두 검증 함수의 판정 일치를 고정하는 것이며 브라우저에서 실제 폼을 조작한 검증은 아닙니다.

## 검증과 배포

`npm --prefix web test` 와 `go test ./internal/app -run TestFindingBulkAssigneeSharedVectors` 로 같은 벡터를 양쪽에서 실행했습니다. 화면 검사는 `assignee.trim() === wire` 를 먼저 확인해 Go 가 재는 열이 실제 요청 본문과 같도록 고정하고, 거절 벡터에서는 `제어 문자`·`200바이트` 중 사유에 맞는 문구가 나오는지까지 확인합니다. 벡터 하나를 지우거나 판정을 바꾸면 두 검사 중 하나가 실패합니다. 릴리즈 커밋에서 실행한 검사와 건너뛴 검사는 [검증 기록](validation.md)에 적었습니다.

화면 캡처는 v1.9.0의 검증 장면을 보존했습니다. 가이드·README·llms.txt·openapi.json 의 버전 표기를 갱신하고 가이드 HTML·PDF 를 재생성했습니다.

배포 이미지: `hunter:v1.18.0` · 유일한 첨부 자산: `hunter-v1.18.0.tar.gz`.

[사용자 가이드](guides/user-guide.html) · [관리자 가이드](guides/admin-guide.html) · [검증 기록](validation.md) · [v1.17.0 릴리즈 노트](release-v1.17.0.md)
