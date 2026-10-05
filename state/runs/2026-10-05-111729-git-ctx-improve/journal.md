# 회차 노트 2026-10-05-111729-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:17] base pinned — main@d198fb3
- [러너 11:17] autonomy release — 
- [러너 11:26] scout timeout — 단계 제한 시간 초과
- [러너 11:26] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇/왜: `clampResponse` 가 답변 어디에 있든 마지막 `\n### Notes\n` 를 진단 블록으로 예약하던 것을 `notesSection(tool,text)` 로 바꿨다 — "답변의 마지막 `### ` 제목일 때만", fencesContent 도구는 "내용 펜스 밖일 때만". Notes 섹션이 없는 get-symbol-context(끝나지 않는 펜스)와 find-runbook(공지가 5건을 "1 of 2" 로)이 재현 경로였다. 프로덕션 1파일(budget.go), 신규 테스트 1파일, commit 1f15215.
- 확신 없는 곳: ① `notes == last` 조건이라 **"Notes" 제목의 결과가 마지막일 때는 여전히 오인**한다 — 알고 남긴 한계이고 ideas.json 에 후보로 적었다. ② 비 fencesContent 도구에 펜스 인식을 켜지 않은 것은 2026-10-03 대조군(`TestSearchCodeCountsHitsPastASnippetThatOpensAFence`)의 판단을 따랐을 뿐 이번에 따로 재측정하지 않았다. ③ `eachLine` 분리는 동작 동일이라고 보지만 증명은 "기존 mcp 테스트 전부 무수정 통과" 뿐이다. ④ export-context 는 Notes 가 없고 섹션 본문이 산문이라 본문 안 `### Notes` 가 마지막 `### ` 제목이면 같은 오인이 남을 수 있다 — 재현 fixture 를 만들지 않았다.
- 일부러 하지 않은 것: 도구별 "Notes 를 쓰는가" 허용목록(=완전한 해법)은 `fencesContent` 허용목록 과제와 한 몸이라 이번 범위에서 뺐다. `format.go` 의 제목 수준·포매터 본문은 한 글자도 안 건드렸다. 버전·릴리즈 노트·CHANGELOG 무수정.
- 다음 역할이 조심할 것: 새 테스트는 `internal/mcp` 의 공유 in-memory SQLite fixture 를 쓰므로 `t.Parallel()` 금지, DB·네트워크 불필요(1.3초대). `TestRunbookTruncationCountsASectionHeadedNotesAsAResult` 는 **같은 fixture 로 find-runbook 을 두 번** 부르므로 `auditedResultCount`(감사 행 1개 가정)를 거기에 쓰면 안 된다. 미실행: 외부 DB·Vault·Docker·실브라우저·govulncheck.
- [러너 11:37] verify passed — 검증 4개 통과 (auto)
- [러너 11:51] review timeout — 단계 제한 시간 초과
- [러너 11:51] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 11:51] pr created — https://github.com/hkjang/git-ctx/pull/46
