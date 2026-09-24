# 회차 노트 2026-09-23-123422-vibe-code-improve — vibe-code
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 12:34] base pinned — main@4969370
- [러너 12:34] autonomy release —

## 정찰 노트
- 체크포인트 과제를 골랐다: 1.4.0에서 들어온 안전장치인데 테스트가 0개이고, `git stash create`가 untracked 파일을 안 담아 "파괴적 명령 전 보호"라는 약속이 새 파일에는 성립하지 않는다. plans.ts 테스트 보강(차선)보다 실사용 결과가 바뀌는 쪽이다.
- 미확인 두 가지: 샌드박스가 막아 임시 git 저장소 실험과 `npm test`를 **한 번도 돌리지 못했다**(워크트리에 node_modules도 없다). `stash create`의 untracked 제외와 `stash apply`가 stash 형식 아닌 커밋을 거부한다는 것은 git 문서 기준 판단이다.
- 구현자는 테스트를 먼저 써서 **현재 코드에서 실패하는 것을 확인한 뒤** 고칠 것. 실패하지 않으면 전제가 틀린 것이니 차선(plans.ts 테스트)으로 넘어가라.
- 조심할 곳: `git stash push/pop` 금지(스택 공유), 임시 GIT_INDEX_FILE은 os.tmpdir에 두고 정리, 커밋 없는 저장소(HEAD 없음) 경로를 지금보다 나쁘게 만들지 말 것, 설정 키 이름 유지.
- 정규식(DESTRUCTIVE) 확대는 일부러 제외했다 — 오탐만 늘고 출력이 안 바뀌는 변경이라 반려 사유가 된다. 
- [러너 12:38] scout done — 파괴적 명령 전 체크포인트가 untracked 파일까지 담도록 고치고 첫 테스트를 붙이기 (가치 5 / 위험 2 / 작업�

## 구현 노트
- `createCheckpoint`의 `git stash create`를 임시 index 스냅샷(`GIT_INDEX_FILE`=os.tmpdir 아래, read-tree→add -A→write-tree→commit-tree)으로 교체했다. untracked 파일이 안 담기던 것이 근거였고, 고치기 전 실제 git 2.43에서 재현한 뒤 신규 테스트 4개가 그 이유로 실패하는 것을 보고 고쳤다. 복원 블록은 `git diff`/`git checkout <commit> -- .` 로 교체(비-stash 커밋이라 `git stash apply`는 거부된다).
- 확신 없는 곳: (1) **Windows 미검증** — `os.tmpdir()`가 저장소와 다른 드라이브일 때 `GIT_INDEX_FILE` 절대경로·lock 파일 동작을 리눅스에서만 확인했다. 확장 호스트/VSIX 검증은 이 세션에서 못 돌린다. (2) **성능 미측정** — `checkpointBeforeCommand`는 동기인데 이제 untracked까지 훑으므로 .gitignore에 안 걸린 대형 디렉터리가 있으면 느려질 수 있다(ideas.json에 항목으로 남김). (3) 스냅샷은 기존 `.vibe-code/`가 untracked면 그것도 담는다 — 의도한 동작이지만 노이즈일 수 있다(노트 파일 자체는 스냅샷 이후에 써서 자기 자신은 안 담긴다).
- 일부러 안 한 것: `DESTRUCTIVE` 정규식 확대(효과 없는 변경), `changelog.md`의 1.4.0 항목 수정(릴리즈 기록이고 릴리즈는 이 세션 일이 아님), `createCheckpoint`의 catch가 update-ref 실패를 '저장소 아님'으로 뭉개는 건(재현을 못 만들어 보류, ideas.json 참고).
- 다음 역할이 조심할 것: `tests/unit/checkpoints.test.ts`는 **PATH의 진짜 git**이 필요하고 `fs.mkdtempSync`로 저장소를 만든다(대역 없음, 로컬 user.name/email을 저장소에 직접 설정하므로 전역 git 설정에 의존하지 않는다). 작업 트리 무간섭을 단정하는 테스트는 노트를 워크스페이스 **밖** 임시 디렉터리에 쓴다 — 노트 파일 자체는 git status를 정당하게 바꾸기 때문이다.
- [러너 12:42] brief accepted — 채택 — 과제서의 근거(stash create가 untracked를 제외, stash apply가 비-stash 커밋 거부)를 실제 git 2.43으로 재현해 확인한 뒤 그
- [러너 12:42] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인: `git stash create`가 untracked 제외·`stash apply`가 비-stash 커밋 거부를 git 2.43에서 직접 재현했다 — 새 테스트 1·4번은 수정 전 코드에서 실제로 깨지는 진짜 회귀 테스트다. `npm run check` 전부 통과(53 tests). sparse-checkout 회귀를 의심해 실험했으나 `add -A`가 sparse 패턴을 존중해 결함 아님.
- 못 본 것: Windows(os.tmpdir이 다른 드라이브일 때 GIT_INDEX_FILE·lock), 확장 호스트/VSIX, 대형 untracked 디렉터리에서의 동기 실행 지연 — 리눅스 CI도 이 셋을 못 본다.
- 승인이어도 남는 우려: gitignore 안 된 untracked 파일이 .git/objects에 들어가 refs/vibe-checkpoints/*로 영구 고정된다(비밀 파일 잔존·저장소 증가, 정리 명령 없음 — 차단 아님, 다음 회차 후보). `.vibe-code/`가 gitignore 안 돼 감사 JSONL도 스냅샷에 들어가고 복원이 그걸 되돌린다.
- 릴리즈 역할: changelog.md에 Unreleased 절이 없다. 1.4.0 항목(`git stash create` 서술)은 출하 사실이라 그대로 두되, 다음 버전 항목에 이 fix를 반드시 적을 것.
- 판정 approve / risk low / blocking 없음. 보안·법무 차단 사유 없음(새 의존성은 node 내장 `os`, 개인정보·외부 전송 없음).
- [러너 12:45] review approved — 리뷰 승인 (risk=low)
- [러너 12:46] pr created — https://github.com/hkjang/vibe-code/pull/1
- [러너 12:48] ci passed — 검사 2개 모두 success
- [러너 12:48] merge done — 2f11b5b
- [러너 12:57] release published — v1.4.1
- [러너 12:57] gh-release created — GitHub Release v1.4.1
- [러너 13:11] assets missing — 이전 v1.4.0 엔 2개, v1.4.1 엔 0개 — 워크플로: null: null/null
