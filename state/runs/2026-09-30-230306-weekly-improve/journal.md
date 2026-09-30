# 회차 노트 2026-09-30-230306-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 23:03] base pinned — main@5463db6
- [러너 23:03] autonomy release — 

## 구현 노트
- `uploadAttachments` 가 파일을 하나씩 저장하며 진행해, 중간 파일이 거절되면 400 을 답하면서 앞선 이미지는 디스크·표에 남겼다(재업로드 → 같은 캡처 두 벌). 검사 전량 선행 + 행 삽입 한 트랜잭션으로 고쳤다. 커밋 9f23531(백엔드+시험), b9eb36c(가이드).
- 확신 없는 곳: ① 저장 단계에서 `header.Open()` 을 **두 번째로** 연다. 메모리 회귀(10MB×20장)를 피하려고 일부러 그렇게 했고 전체 시험이 통과하지만, 파일 경로로만 쓰이던 것을 두 번 읽는 것은 이번에 처음이다. ② `tx.Commit` 실패 경로와 `STORAGE_FULL`(ENOSPC) 경로는 시험으로 밟지 않았다 — 새 시험이 증명하는 것은 **검사 실패** 경로뿐이다. ③ `guard-check` 가 새 시험의 `uploadAttachments` 도달률을 56% 로 잰다.
- 일부러 하지 않은 것: 프런트 무변경(응답 코드·문구·필드가 그대로라 고칠 것이 없다) → `npm` 검증 미실행. 롤백 뒤 디스크에 남는 고아 파일은 지우지 않았다(내용 주소라 재시도가 같은 이름에 덮어쓴다). `deleteAttachment` 의 참조 수 질의 오류 무시는 손대지 않고 다음 후보로 적었다.
- 다음 역할이 조심할 것: `TestARefusedAttachmentUploadStoresNoneOfTheImages` 는 실제 PostgreSQL(`WEEKLY_TEST_POSTGRES_DSN`)이 있어야 돈다 — DSN 없이는 SKIP 이고, 그 SKIP 을 통과로 세면 안 된다. 같은 시험 끝의 "2 rows" 단언이 '전부 거절하는' 가짜 고침을 막는 축이니 지우지 말 것.
- [러너 23:17] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다. main 을 /tmp 로 git archive 해 새 시험만 얹고 실제 PostgreSQL(15434)로 돌리니 `stored 1 image(s)` / 디스크에 파일 1개 / 재업로드 후 `3 rows` 로 실패하고, HEAD 에서는 통과한다 — 시험은 진짜로 바뀐 경로를 지난다. 전체 `go test ./...` 실제 DSN 통과(internal/app 151.6s), guard-check 12개 통과, openapi/paging 통과.
- 구현자 의심 ①(두 번째 `header.Open()`)은 문제 없다. multipart 본문은 메모리 사본이거나 `RemoveAll` 전까지 살아 있는 임시 파일이라 두 패스가 다른 바이트를 볼 수 없고, 1패스의 sha256/width/height 와 2패스가 쓰는 본문이 어긋나지 않는다.
- 못 본 것: `tx.Commit` 실패와 ENOSPC 경로는 나도 밟지 못했다. 그 경로에서 행은 롤백되지만 이미 쓴 파일은 남고, `cleanupAttachmentFiles` 는 보고서의 행이 0일 때만 디렉터리를 지우므로(attachments.go:617-623) 첨부가 이미 있는 보고서의 고아 파일은 영구히 남는다 — 변경 전에도 같았으니 회귀는 아니고 다음 회차 후보.
- 승인이어도 남는 우려 둘: (a) 트랜잭션이 파일 쓰기 20×10MB 를 가로질러 열려 있고 pgxpool MaxConns=20 이라, 파일을 다 쓴 뒤 INSERT 만 트랜잭션에 담는 편이 낫다. (b) `nextOrder` 는 여전히 트랜잭션 밖·오류 무시라 같은 작성자의 동시 업로드에서 같은 sort_order 로 시작할 창이 전보다 약간 넓다. 둘 다 실패를 재현하지 못해 차단하지 않았다.
- 보안·법무 차단 없음: 새 경로·권한 변경 없고 저장 경로는 sha256+허용확장자라 Filename 경로 탈출이 없으며, 새로 수집하는 개인정보·새 의존성·외부 약속 문구가 없다. 릴리즈 노트에는 "거절된 업로드는 이제 아무 이미지도 남기지 않는다"만 적으면 되고, 문서의 그 문장은 UNSUPPORTED_IMAGE·FILE_TOO_LARGE 범위에서만 참임을 기억할 것.
- [러너 23:22] review approved — 리뷰 승인 (risk=low)
- [러너 23:22] pr created — https://github.com/hkjang/weekly/pull/27
- [러너 23:45] ci passed — 검사 1개 모두 success
- [러너 23:45] merge done — b9eb36c
