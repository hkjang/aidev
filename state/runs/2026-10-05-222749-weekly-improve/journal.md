# 회차 노트 2026-10-05-222749-weekly-improve — weekly
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:27] base pinned — main@f7488dd
- [러너 22:27] autonomy release — 

## 정찰 노트
- 골랐다: `updateAttachment` 가 placement 만 바뀐 PATCH 에서 sort_order 를 그대로 둬 목적지 그룹에 중복 번호를 만드는 결함. 차선 embeddingStatus 를 제친 이유 둘 — 오류 삼키기 수정은 Confluence 에서 두 번 머지돼 세 번째 반복이고, 이쪽은 질의 실패 주입 없이 프런트가 실제 내는 요청 4번으로 결정적 재현되며 프로덕션 1파일로 끝난다.
- 재현을 코드로 따라가 확인: 업로드는 항상 AFTER(AttachmentPanel.tsx:41), max+1 은 그룹 안에서만(attachments.go:269) → 빠져나간 번호가 재사용된다. 2장 업로드·B 이동·1장 업로드·C 이동 = BEFORE 에 1,1. 그 뒤 move()(85-93)가 같은 값끼리 맞바꿔 사용자가 순서를 고칠 수 없다.
- 추측으로 적은 것: 프런트 주장은 소스를 읽은 것까지다. AttachmentPanel 렌더 시험의 유무는 미확인이고, 4단계 재현을 **HTTP 로 실행하지는 않았다**(이번 회차에 시험을 돌리지 않음). 운영 DB 의 기존 중복도 미확인.
- 구현자가 조심할 것: migrations/008 에 UNIQUE 를 새로 걸지 말 것(출하된 체크섬·기존 중복 가능성). 프런트는 한 줄도 바꾸지 말 것(그러면 npm 검증이 빠진다). 업로드와 이동이 같은 max+1 식을 쓰게 맞출 것. pgx 매개변수에 명시적 형(`$3::int`)을 붙일 것.
- 프로필은 0일 전 것이라 새로 쓰지 않았다. 단 기준 커밋이 낡았으니 guard-check 는 `--changed f7488dd` 로 돌릴 것(8ab3712 아님) — 과제서에 적어 뒀다.
- [러너 22:33] scout done — 첨부 이미지의 삽입 위치를 바꿀 때 목적지 그룹의 끝으로 보내, 두 이미지가 같은 순서 번호를 갖지 않게

## 구현 노트
- 무엇을 왜: `updateAttachment` 의 단일 `UPDATE`(attachments.go:432)에 `sort_order` `CASE` 를 넣어, `placement` 가 **현재 값과 다를 때만** 목적지 그룹의 `coalesce(max(sort_order),-1)+1`(업로드 269행과 같은 식)로 다시 센다. 보낸 `sortOrder` 는 그대로 이기고, 같은 placement 재전송은 이동이 아니다. 프로덕션 1파일.
- 확신 없는 곳·검증 못 한 것: (1) **프런트는 소스를 읽은 것까지다** — `AttachmentPanel.tsx` 의 `move()` 맞바꾸기와 placement-only select 를 렌더 시험으로 확인하지 않았다(이 저장소에 해당 렌더 시험이 없다). 시험은 그 요청 **모양**을 Go 쪽에서 재현한 것이다. (2) 운영 DB 에 **이미 있는** 중복 `sort_order` 는 확인하지 않았고 정리하지도 않았다 — 이 수정은 앞으로 만들지 않는 것까지다. (3) 과제서에 없던 가드 하나(`sqlparams_test.go` 매개변수 재사용)에 걸려, 그 가드가 지시한 절차대로 PREPARE 로 형을 확인하고(`{character varying, text, integer, bigint, bigint}`) 허용 목록에 사유와 함께 등록했다 — 비평가가 볼 자리는 여기다. 그 결과 attachments.go 의 **다른** 재사용도 앞으로 이 가드를 지나가지 않는다(가드 입도가 파일 단위다). 별도 placeholder($6/$7) 로 피할 수도 있었으나 같은 값을 두 번 넘기는 쪽이 더 틀리기 쉽다고 판단했다.
- 일부러 하지 않은 것: `migrations/008` 에 `(report_id, placement, sort_order)` UNIQUE 추가(출하된 체크섬 + 기존 중복 가능성), 프런트 수정(그래서 `npm` 검증 없음), 269행 `nextOrder` 질의의 오류 무시(같은 증상의 두 번째 입구지만 질의 실패 주입이 필요해 ideas.json 에 남김), 문서 갱신(사용자 가이드가 순서 규칙을 서술하지 않아 고칠 문장이 없었다).
- 다음 역할이 조심할 것: 새 시험 3개는 **DB 가 있어야 돈다** — `WEEKLY_TEST_POSTGRES_DSN` 없이 돈 SKIP 은 성공이 아니다(`weekly-test-pg` 15434, 자격은 `docker inspect` 로 자식 환경에만). `-run 'Attachment'` 로는 새 시험이 **안 잡힌다**(이름에 Attachment 가 없다) — `-run 'Attachment|Capture|SortOrder|Placement'` 를 쓸 것. `guard-check` 기준은 `f7488dd`.
- [러너 22:45] brief accepted — 채택 — 과제서의 4단계 실패 경로를 실제 HTTP 로 그대로 거쳐 중복 `sortOrder` 를 재현했고, 지정한 프로덕션 1파일의 `CASE` 
- [러너 22:48] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: main 핸들러를 overlay 로 끼워 새 시험을 돌렸고 머리 시험이 주장한 문구 그대로 실패(`BEFORE holds b.png and c.png on the same sortOrder 1`), 수정본에서 3개 통과. guard-check `--changed f7488dd` 9개·build/vet·`Attachment|Capture|SortOrder|Placement|Export|Pptx|SQL`·openapi 통과, 트리 깨끗. 승인.
- 못 본 것: 프런트 렌더 시험(이 저장소에 없음), npm 전체 검증, 운영 DB 의 기존 중복 `sort_order`.
- **남는 우려(릴리즈 노트)**: 서버가 호출자가 보내지 않은 `sort_order` 를 쓰는데 응답은 `{"id":…}` 뿐이고 `AttachmentPanel.tsx:78-84` 의 `change()` 는 성공 시 reload 를 안 한다. 실제 HTTP 로 재현 — BEFORE=x(0),y(1)/AFTER=a(0) 에서 a 를 BEFORE 로 옮긴 뒤(서버 2, 캐시 0) ↑ 한 번이면 x=0,y=0 중복. **main 에서도 같은 최종 상태라 퇴행은 아니며**(그래서 거절하지 않음) 두 입구 중 프런트 쪽이 남았다.
- 다음 회차 후보: `updateAttachment` 응답에 바뀐 `sortOrder` 를 싣거나 `change()` 가 placement 변경 시 `load()` 하게 하기; `sqlparams_test.go` 허용 목록을 파일 단위에서 (파일, 질의) 단위로 좁히기.
- 작은 흠: `attachments.go:444` 의 "nothing to race" 는 과하다 — 같은 목적지로 들어가는 동시 PATCH 둘은 서로 다른 행이라 잠금 없이 같은 `max+1` 을 읽는다(269행 업로드도 동일, 새 위험 아님).
- [러너 22:54] review approved — 리뷰 승인 (risk=low)
- [러너 22:54] pr created — https://github.com/hkjang/weekly/pull/31
