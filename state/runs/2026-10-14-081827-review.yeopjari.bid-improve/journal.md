# 회차 노트 2026-10-14-081827-review.yeopjari.bid-improve — review.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 08:18] base pinned — main@93db716
- [러너 08:18] autonomy release — 

## 구현 노트
- 무엇·왜: 리뷰 쓰기 화면의 "파트너 이름" 입력란이 `value={form.partner_name.trim()}` 로 제어돼, 한 글자씩 치면 공백을 넣는 순간 React 가 DOM 값을 되돌려 "메가존 클라우드" 가 "메가존클라우드" 로 붙었다. 또 표시 조건이 텍스트 유무였어서 칸을 다 지우면 입력란이 사라졌다. 화면 전용 `partner_by_name` 플래그로 "이름으로 적기" 선택 여부를 들고, 값은 다듬지 않고 보여 준다. 프로덕션 파일 1개(apps/web/src/routes/Write.tsx).
- 확인함: 화면이 이제 보내는 본문(화면 전용 키 `partner_by_name` 포함 + 공백 있는 이름)을 실제 서버에 POST /api/v1/reviews 로 보내 200, GET /api/v1/me/reviews 로 되읽어 `partner.name === "메가존 클라우드"` 를 확인했다(일회용 probe, 커밋 안 함). 서버 `parse` 가 shape 키만 읽어 모르는 키를 무시한다는 것도 이걸로 실증.
- 확신 없는 곳: 옛 임시 저장(partner_by_name 없는 초안)은 `form.partner_name !== ''` 로 칸을 다시 띄우게 했는데, 실제 옛 초안(localStorage 를 손으로 심은 상태)으로 재현해 보지는 않았다.
- 일부러 안 한 것: 파트너 이름란에 `error={err('partner_name')}` 를 붙이는 것(서버가 min:0 이라 이 필드로 오류가 나지 않음), v.string/v.text 글자수 기준 통일(재평가 결과 사용자 경로가 없어 ideas.json 에서 rejected), 쿼터 Retry-After 9시간 과장(ideas.json 에 신규 pending).
- 다음 역할 주의: 추가한 검사는 `scripts/ui-flows.mjs` 안에 있고 **CI(check.yml)는 ui-flows 를 돌리지 않는다** — 돌리려면 DB·서버·시드(dev-reset.sh + seed-demo.mjs)와 브라우저가 필요하다. 이 환경에서는 `CHROMIUM_PATH=/usr/bin/google-chrome node scripts/ui-flows.mjs` 로 playwright 다운로드 없이 돈다. e2e 는 dev-reset 직후에만 218/0 이다(그 뒤 ui-flows 를 돌린 DB 에서는 상태 의존 검사가 깨진다).
- [러너 08:31] verify passed — 검증 6개 통과 (auto)

## 비평 노트
- 확인함: 수정 전 코드로 새 ui-flows 검사 2개가 반드시 실패함을 diff 로 확정(렌더 조건이 `form.partner_name` 진릿값 → fill('') 시 언마운트, `value=...trim()` → 공백 삼킴). 저널에 `- 실패 재현:` 줄은 없었다.
- 실제로 돌림: :8796 이 이 워크트리 빌드(Write-U6MzVd_N.js, 바이트 동일)를 서비스 중이라 격리 playwright probe 로 공백 유지·전부 지운 뒤 재입력 모두 통과, console/pageerror 0건. `npm run lint` 도 통과(verify 는 lint 미실행).
- 구현자 '확신 없는 곳' 재현함: `{partner_id:null,partner_name:' '}` 옛 초안을 localStorage 에 심어 새로 고치니 Select='목록에 없음…', 입력란 보임(값 ' ') — 동작함. **단 옛 초안은 partner_by_name=false 라 입력란을 다 지우면 또 사라진다**(최대 14일). 초안 병합에서 `partner_by_name: true` 보정이 자가 치유 지점.
- 못 봄: ui-flows 전체·e2e·a11y 미실행(새 블록만 격리 실행). check.yml 이 ui-flows 를 안 돌리므로 이 회귀는 CI 가 막지 않는다 — 릴리즈 노트에 남길 것.
- 보안·법무: 인가·마이그레이션·비밀값 무변경, 신규 개인정보 없음, `partner_by_name` 은 `parse`(validate.ts:23) 가 shape 키만 읽어 폐기. 차단 없음. 판정 approve / risk low.
- [러너 08:37] review approved — 리뷰 승인 (risk=low)
- [러너 08:37] pr created — https://github.com/hkjang/review.yeopjari.bid/pull/2
- [러너 08:41] ci passed — 검사 1개 모두 success
- [러너 08:41] merge done — 807ab20
- [러너 08:50] release published — 0.7.4
