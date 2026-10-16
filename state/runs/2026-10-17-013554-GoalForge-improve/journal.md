# 회차 노트 2026-10-17-013554-GoalForge-improve — GoalForge
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:36] base pinned — main@3459587
- [러너 01:36] autonomy release — 
- [러너 01:40] scout done — 영문 랜딩 페이지의 `<head>`(메타·canonical·JSON-LD)를 영문으로 복구한다 — JS 가 고쳐 주지 못하는 부분만 (�

## 구현 노트
- 무엇/왜: `docs/index_en.html` 의 head 만 고쳐 손상된 title·한국어로 남은 description/keywords/og:title/og:description·한국어 루트를 가리키던 canonical/og:url/twitter:url·og:locale=ko_KR·JSON-LD FAQPage 4문4답을 복구했다(커밋 ac7967b, 한 파일 17/17, 모든 hunk 6~93행). 영문은 지어내지 않고 twitter:title·JSON-LD SoftwareApplication.description·script.js en 사전 faq_1~faq_4 에서 옮겼다.
- **과제서 전제 2건이 실측으로 틀렸다 — 비평가가 여기를 먼저 볼 것.** (가) JSON-LD Q3(프로바이더)의 영문 원문은 저장소에 없다(en 사전 faq_3 은 Verification Gates). 영문 페이지가 실제 렌더링하는 FAQ Q3 도 faq_3 이므로 구조화 데이터를 화면 내용과 일치시켰다 — 즉 **Q3 의 주제가 프로바이더에서 Verification Gates 로 바뀌었다**. "영문 금지 창작" 제약을 지키는 유일한 선택이었고 schema.org 의 FAQPage 요구(마크업=가시 내용)와도 맞지만, 주제 교체가 의도와 다르면 되돌릴 지점은 여기다. (나) script.js:420-427 은 site_title 로 document.title 을 **실제로 덮어쓴다**(과제서는 안 건드린다고 했다). 단 meta_desc 키는 두 사전에 모두 없어 description 은 끝까지 복구되지 않고, JS 미실행 크롤러에겐 정적 title 도 깨져 보이므로 과제 근거 자체는 유효하다.
- 확신 없는 것: 브라우저·실제 크롤러 실행 검증은 안 했다(수용 기준을 전부 파일 내용으로 판정). og:description 은 과제서가 twitter:description(28행)과 맞추라 했으나 그 문장에 raw `&` 가 있어 새 엔티티 오류를 만들지 않으려고 52행 JSON-LD description 을 meta description 과 og:description 양쪽에 썼다 — 두 값이 twitter:description 과 문구가 다르다(모두 기존 영문, 창작 아님).
- 일부러 안 한 것: 127행 깨진 비교(두 버튼 모두 data-lang 을 가져 textContent 폴백이 실행되지 않는 도달 불가 코드임을 확인; '한' 복원은 수용 기준 2)의 head 한글 금지와 충돌 → 과제서가 허용한 대로 미수정, head 잔존 1곳). 본문 135~874행(손상 136곳 잔존), script.js·index.html·styles.css(읽기만), hreflang 추가(index.html 을 건드려야 함).
- 다음 역할 주의: Go 는 무관하다 — 저장소에 `go:embed` 가 없어 docs/ 는 바이너리에 안 들어간다. `go test ./...` 를 통과 근거로 내세우지 말 것(이번엔 go build·go vet 만 exit 0 확인). 검증은 저장소 **밖**의 `check_head.py`(이 run 디렉터리)로 하며 `python3 -I check_head.py <path>` 로 돌린다 — 수용 기준 5개를 단언하고 수정 전 11 FAIL → 후 0 FAIL, `git show HEAD~1:docs/index_en.html` 에 돌리면 다시 11 FAIL(인과 고정). xmllint 는 전·후 61건 동일(메시지 diff 무출력)이며 **무출력이 기준이 아니다** — 28·34행의 raw `&` 등은 이전부터 있던 것이다.
- [러너 01:48] brief accepted — 채택 — 지목한 행·손상 개수·정본 위치가 지금 코드와 정확히 일치했다. 다만 과제서의 두 전제는 실측으로 틀렸고 그�
- [러너 01:49] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인: diff 는 docs/index_en.html 한 파일 17/17행뿐(비-docs 변경 0). check_head.py 를 HEAD~1 에 돌려 11 FAIL → HEAD 0 FAIL, 손상 145→136 로 인과 재현. JSON-LD 는 json.loads 통과, FAQ 4문4답이 script.js en 사전 faq_1~faq_4 와 글자 그대로 일치(창작 없음). 복원문의 CLI 주장도 main.go:990·3106-3107 과 대조해 사실.
- 구현 노트 (나)는 **틀렸다**: `meta_desc` 는 script.js:11·200 양쪽 사전에 다 있고 :426-428 이 description 을 덮어쓴다. 결론은 안 바뀌지만(정적·런타임 둘 다 영문) 다음 회차가 근거로 쓰면 안 된다. 127행 도달 불가 주장은 166·167행 data-lang 확인으로 **맞다**.
- 못 본 것: 브라우저·실제 크롤러 실행, xmllint 전수, docs 본문 745~874행(손상 136곳 잔존).
- 승인이어도 남는 우려: JSON-LD 가 'MCP 15개'라 하지만 실측 21개(tools.go) — 이번 커밋이 만든 오류는 아니나 이제 구조화 데이터의 구체 숫자다. `@id` 는 아직 한국어 루트(#faq)여서 canonical 자기참조와 어긋난다. hreflang 없음. **릴리즈 노트는 '영문 랜딩 복구 완료'가 아니라 'head/메타·구조화 데이터 복구'로 한정할 것** — 본문은 아직 깨져 있다.
- 보안·법무 차단 없음(인가·비밀값·의존성·개인정보 변경 전무). 수리가 생긴다면 먼저 볼 파일: docs/script.js(사전 정본) 과 docs/index.html(@id·hreflang 공유).
- [러너 01:53] review approved — 리뷰 승인 (risk=low)
- [러너 01:54] pr created — https://github.com/hkjang/goalforge/pull/85
- [러너 02:00] ci passed — 검사 5개 모두 success
- [러너 02:00] merge done — ac7967b
- [러너 02:14] release published — v0.57.0
- [러너 02:24] assets verified — v0.57.0 자산 7개 (이전 v0.56.0: 7)
