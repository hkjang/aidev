# 수리 요약 (수리 시도 3) — 커밋 없음

- 지적(CI `check` 실패)은 **이 PR 의 결함이 아니다**: 잡이 **3초**에 죽었고 실패한 step 이름도 없다(러너 기록 `실패한 검사: ?`, 과제서 로그 끝부분 비어 있음). `npm ci` 만 12초(`/tmp/npmci.log` "added 231 packages in 12s")이고 그 앞에 postgres 서비스 컨테이너 초기화·checkout·setup-node 가 있으므로, 3초 안에는 `npm run check` 는 물론 build·migrate·e2e·screenshots 중 **어느 step 도 시작조차 못 한다**. 저장소 내용이 닿을 수 없는 자리에서 죽었다.
- 로컬에서 같은 게이트를 HEAD(2f37e63) 그대로 돌려 전부 통과: `npm ci` exit 0, `npm run check` exit 0(eslint + typecheck 3 + **109 테스트**), `npm run build` exit 0(web 7196 모듈 + server 3 번들). `git merge-base origin/main HEAD == origin/main` 이라 `refs/pull/6/merge` 는 HEAD 와 같은 트리 — CI 가 보는 것과 같은 것을 돌렸다.
- 비평가가 남긴 유일한 내용 가설(테스트2 `dropped - before > 3` 가 느린 러너에서 깨진다)도 재현되지 않았다: 2코어 고정(`taskset -c 0,1`)에 스피너 6개를 얹은 4배 과포화에서 5회 연속 11/11 통과, 테스트 본문 1.26~1.39초(상한 10초). 순수 CPU 경합으로는 깨지지 않는다.
- 남은 설명은 모두 저장소 밖이다(`Set up job` 단계: Actions 과금·지출 한도, 컨테이너 이미지 pull 제한, 러너 할당). 마지막 녹색(739ea33) 이후 `.github/` 변경은 0줄이고(`git diff 739ea33 HEAD -- .github/` 빈 출력), 서로 다른 세 diff(PR #4·#5·#6)와 main@601659d 가 똑같이 죽는다 — 내용 무관 원인의 모양이다. 이 세션도 `gh` 미인증·비공개 저장소(API 404)로 로그를 못 봤다.
- 따라서 **고치지 않고 커밋 없이 끝낸다**. 통과시킬 수 있는 유일한 방법은 `check.yml` 수정이나 단정 완화인데 둘 다 절대 규칙 위반이다. 사람이 PR #6 잡 117445410932 의 "Set up job" 출력과 Actions 과금 상태를 직접 확인해야 한다. 릴리즈 노트에 'CI 고침' 으로 적지 말 것.
