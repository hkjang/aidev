# 회차 노트 2026-10-18-142134-GoalForge-improve — GoalForge
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:21] base pinned — main@f746b00
- [러너 14:21] autonomy release — 

## 정찰 노트
- 직전 회차의 보류 1위(영문 본문 L 전체)를 쪼개 **JS 가 끝까지 덮지 않는 9 줄**만 골랐다. Go 쪽 상위 후보(AutoApproveMerges DailyLimit, FailureRateLimit 무한 재시도)는 전부 설계 판단이 선행하는 권한·중단 조건 변경이라 45분 단독 회차로 성립하지 않고, MCP since 7d→168h 는 2026-10-09 에 no-change 로 끝난 전력이 있어 제쳤다.
- 직전 정찰 노트를 **실측으로 정정했다**: FAQ 답변·히어로 서브타이틀·GitHub 버튼은 `data-i18n` 이 **부모 요소에** 있어 런타임에 덮인다(`grep -v data-i18n` 이 줄 단위라 생긴 오판). 그래서 "보이는 손상" 은 166행 + cli-code 주석 8줄뿐이다.
- 추측으로 남긴 것: (가) 233행 `#terminal-body` 가 `script.js:525` 로 교체되는지 — **미확인**이라 범위에서 뺐다. (나) `xmllint` 설치 여부 — 샌드박스가 `command -v` 를 거부해 **미확인**, 없으면 수용 기준 6)은 건너뛰어도 된다. (다) 8 줄의 영문 문구는 한국어 원문의 직역 제안이며 정본 영문은 없다.
- 구현자가 조심할 것: 127행(head 인라인 스크립트)을 되돌리면 2026-10-17 의 "head 한글 금지" 기준과 충돌한다 — 손대지 말 것. `grep -c` 가 **136 → 127** 로 정확히 9 만 줄어드는지가 범위 초과를 잡는 가장 싼 검사다. 584행에는 이스케이프되지 않은 생 `&` 가 섞여 있으니 새 영문에서는 `&amp;` 로 쓸 것.
- [러너 14:26] scout done — 영문 랜딩 페이지에서 **JS 가 절대 덮어쓰지 않는** 9 줄을 영문으로 복구한다 — 언어 버튼 라벨 1 줄 + `pre

## 구현 노트
- 바꾼 것: `docs/index_en.html` 9 줄만. 166 행 → `한`(`docs/index.html:166` 과 바이트 동일), 553/556/568/573/584/588/598/602 행 `term-comment` 내부 텍스트 → 같은 행 한국어 원문의 직역 영문. 이 9 줄은 어느 조상에도 `data-i18n` 이 없어 JS 를 켜도 보이는 유일한 손상이다.
- 확신하는 것(실측): `script.js:391-398` 은 `.lang-btn` 에 click 리스너만 붙이고 텍스트를 쓰지 않는다. `grep -n "cli-code\|term-comment\|lang-btn\|cli-card" docs/script.js` 결과 `.cli-code`·`.cli-card` 참조는 0 건(`term-comment` 히트는 전부 `terminalTabsData` 안의 `#terminal-body` 문자열). 체크 스크립트가 주석 바깥 markup 을 HEAD 와 문자열 비교해 태그·명령줄 무변경을 단언한다.
- **확신 없는 것**: (가) 영문 문구는 정본이 없다 — 한국어 원문의 직역이며 `script.js` en 사전(`cli_1_title`~`cli_4_title`) 용어에 맞췄을 뿐이다. 특히 568 행 "Create a goal for the version"(원문 "버전에 맞는 목표 생성")은 직역이라 영어로 약간 어색할 수 있다. (나) **브라우저로 실제 렌더링을 확인하지 않았다** — JS 가 덮지 않는다는 결론은 소스 추적(리스너/셀렉터)에 근거하며 실행 증거가 아니다. (다) 233 행 `#terminal-body` 가 초기 로드에 교체되는지 끝까지 추적하지 않았다.
- **수용 기준 6) 미충족, 의도적**: `xmllint` 경고가 61 → 60 으로 **하나 줄었다**(diff 는 `49d48` 삭제 한 줄, 추가 0 건). 사라진 것은 584 행의 `htmlParseEntityRef: no name` 으로, 손상이 남긴 **생 `&`** 가 내던 오류다. 과제서가 같은 문서에서 `&amp;` 로 적으라고 지시했으므로 기준 6) 과 모순이며, 집합을 보존하려면 마크업 결함을 일부러 되살려야 해 택하지 않았다. `xmllint` 는 설치돼 있었다.
- 일부러 안 한 것: 127 행(head 인라인 사문), 233 행(`#terminal-body`), `data-i18n` 이 책임지는 127 줄, `docs/script.js`, `docs/index.html`, Go 코드 — 과제서의 범위 경계 그대로.
- 다음 역할이 조심할 것: 체크 스크립트는 저장소 **밖** `check_visible_en.py` 이며 커밋되지 않았다. 실행은 `python3 -I check_visible_en.py <en> <ko> <head_en>` 이고 `head_index_en.html`/`head_index.html`(= `git show HEAD:…` 산출물)이 같은 디렉터리에 있어야 한다. 인과 고정은 "수정 전 파일 9 FAIL → 수정 후 0 FAIL" 이다. `go test ./... -count=1` 은 exit 0 (32 ok)이지만 docs 를 읽는 테스트는 없으므로 이 변경의 증거가 아니다 — 증거는 체크 스크립트와 `git diff -U0` 의 hunk 범위다.
- [러너 14:32] brief accepted — 채택 — 지목한 행 번호·손상 개수(136)·원문 대조표 8 줄이 지금 코드와 **정확히 일치**했고, 직전 정찰의 오판(FAQ·히어�
- [러너 14:34] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인한 것: HEAD~1..HEAD 가 docs/index_en.html 9 줄뿐임(origin/main==f746b00, 로컬 main 은 31 behind 라 three-dot diff 의 나머지 31 커밋은 이미 상류 머지분), 545-610 구간 ko/en diff 로 태그·명령줄 무변경, 166 행 '한' 동일성, 손상 카운트 136→127, xmllint 183→180 줄.
- **구현자가 '확신 없다' 한 자리에서 전제가 깨졌다**: script.js:403 updateLanguage 는 click(:398)에서만 호출되고 index_en.html:872 의 setLanguage 는 docs/ 어디에도 정의가 없다 → **초기 로드에 i18n 이 전혀 적용되지 않아 남은 127 줄도 다 보인다**. 고친 주석 바로 위 h3(551·566·582·596)이 깨진 채 렌더된다.
- 못 본 것: 브라우저 실측 렌더링(샌드박스에 브라우저 없음), 233 행 #terminal-body 초기 교체 여부, 커밋되지 않은 check_visible_en.py 재실행.
- 승인하지만 남는 우려: 릴리즈 노트에 '보이는 손상 해소' 로 쓰면 거짓이다. 9 줄 수정은 순개선이고 결함을 넣지 않아 차단하지 않았다(보안·법무 모두 공격 경로·약속 문구 없음).
- 다음 회차 최우선: i18n 부트스트랩 수리(로드 시 updateLanguage('en')). 127 줄 직역보다 싸고 그것이 실제로 보이는 손상을 없앤다.
- [러너 14:37] review approved — 리뷰 승인 (risk=low)
- [러너 14:37] pr created — https://github.com/hkjang/goalforge/pull/86
- [러너 14:47] ci passed — 검사 5개 모두 success
- [러너 14:47] merge done — 4417128
