# PR 처리기 노트 2026-10-08-041355-moina-shepherd — moina PR #44
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-013831-moina-improve)
# 회차 노트 2026-10-08-013831-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:38] base pinned — main@c9bfca2
- [러너 01:38] autonomy release — 

## 정찰 노트
- 다른 후보(admin.go 25곳 이관 등)를 모두 제친 이유: `origin/main` 이 아직 `c9bfca2` 이고 2026-10-06 두 회차의 결과물(`238eff1` social.go 이관 · `7b0f70d` AuthContext 경합 수정)이 **둘 다 미머지**임을 실행으로 확인했다. 즉 새 기능 과제를 올려도 같은 프런트 경합이 image 잡에서 또 죽인다 — 블로커를 먼저 치우고 유실된 이관을 함께 태우는 것이 이번 회차의 유일한 합리적 선택이다. 프로덕션 파일은 `social.go` **1개**뿐이다.
- 과제서에서 추측으로 적은 것: `TestPostgreSQL*` **45건**과 프런트 **36파일 250테스트**는 지난 두 회차 원장에서 옮긴 값으로 이번에 재확인하지 않았다. `make image` 가 지금 통과한다는 것도 미확인(0917 회차의 보고를 믿었을 뿐 이 워크트리에서 재현하지 않았다).
- 구현자가 조심할 것 ①: 이 워크트리의 **로컬 `main` ref 가 낡았다**(`95255f3` = v0.1.28). `git log main..<branch>` 가 87파일 7500줄짜리 허깨비 diff 를 낸다 — 판정은 반드시 `origin/main` 과 `git merge-base --is-ancestor <sha> origin/main` 으로.
- 구현자가 조심할 것 ②: `238eff1` 재제출은 "릴리즈를 깨뜨린 접근의 반복" 이 **아니다**(CI 실패 원인이 그 커밋 밖의 프런트 경합이었고, 이번에는 그 고침을 같은 브랜치에 먼저 담는다). 커밋·PR 본문에 이 인과를 적어 중재가 오판하지 않게 할 것. 또 프런트 경합은 부하 의존이므로 `npm test` 1회 통과로 판정하지 말고 `make image` 까지 볼 것.
- [러너 01:44] scout done — 머지되지 못한 지난 두 회차 커밋을 한 브랜치로 복원 — `7b0f70d`(AuthContext.test.tsx 경합 수정 = `98529c0` 재복�

## 구현 노트
- 무엇을 왜: 과제서대로 미머지 두 커밋을 한 브랜치에 복원했습니다 — `cherry-pick 7b0f70d`(프런트 블로커 먼저) → 프런트 검증 → `cherry-pick 238eff1`. 네 파일 모두 원본과 byte-identical(`git diff <sha> HEAD -- <paths>` 빈 출력), 프로덕션 파일은 `social.go` 1개, 보호 경로 0건. 위 커밋 메시지에만 이번 회차의 인과(이 변경이 반려된 적은 없고 CI 를 죽인 것은 바깥의 프런트 경합)와 검증 결과를 **덧붙였습니다** — 파일은 그대로이므로 amend 후에도 byte-identical 단언은 빈 출력입니다. 커밋은 정확히 2개입니다.
- 확신 없는 곳·검증 못 한 것: (1) **프런트 경합을 이번에 probe 로 재현하지 않았습니다.** 근거는 `make image` exit 0 과 호스트 `npm test` 연속 2회뿐이고, 경합은 부하 의존이므로 이 호스트에서 통과한 것이 CI 러너에서 통과한다는 보장은 원리적으로 없습니다(복원된 고침 자체는 2026-10-06 회차가 `setImmediate` 5ms probe 로 결정론적 red→green 을 관측한 것입니다). (2) e2e(Playwright 시각/접근성/smoke)와 `make package`·`make verify-package` 는 **돌리지 않았습니다** — 응답이 불변이고 과제서의 검증 목록에도 없으며 회차 TIMEOUT 전례가 2건이라 뺐습니다. CI image 잡은 이것들을 돌리므로 그쪽이 첫 미지입니다. (3) `followUser` 네 출구가 `handler` 만으로는 서로 구분되지 않는 한계는 `238eff1` 그대로 남아 있습니다.
- 일부러 하지 않은 것: `storage_error.go` 헬퍼·`RowsAffected()==0` 404 4곳·`IsConflict` 409·`owner_cannot_leave` 409·`store.ErrNotFound` 404 분기 무변경. `AuthContext.tsx`(프로덕션) 무변경. 차선 후보 `admin.go` 25곳 이관은 1순위가 성립했으므로 손대지 않았습니다. OpenAPI·프런트 소스도 응답 불변이라 무변경입니다.
- 다음 역할이 조심할 것: 백엔드 integration 은 **DB 가 있어야 돕니다** — `MOINA_TEST_POSTGRES_DSN` 없이 돌리면 `t.Skip` 이 되어 통과처럼 보입니다. 반드시 `-v` 로 `--- SKIP` **0줄**을 눈으로 확인하세요(이번 회차 확인값: FAIL 0 · SKIP 0 · 최상위 `TestPostgreSQL*` PASS **45건**). 검증에 쓴 throwaway 컨테이너(`moina-test-pg`, 127.0.0.1:55781)는 끝에 지웠습니다. 로컬 `main` ref 는 여전히 낡았으므로(`95255f3`) 판정은 `origin/main` 기준으로 하세요.
- [러너 01:51] brief accepted — 채택 — 과제서가 실행으로 확인했다고 적은 것을 전부 재확인했고 모두 참이었습니다(`origin/main` = `c9bfca2`, 두 커밋 모두
- [러너 01:51] verify passed — 검증 7개 통과 (auto)
- [러너 01:51] pr created — https://github.com/hkjang/moina/pull/44
- [러너 01:51] guard held — frontend/src/auth/AuthContext.test.tsx 
