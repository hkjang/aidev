---
title: "aiportal-py — 자율 개선 이력"
description: "aiportal-py: 자율 개선 회차 9회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-06 13:08:15 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "aiportal-py",
 "codeRepository": "https://github.com/hkjang/aiportal-py",
 "url": "https://hkjang.github.io/aidev/projects/aiportal-py/",
 "description": "aiportal-py: 자율 개선 회차 9회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-06T13:08:15+09:00"
}
</script>

# aiportal-py

<p class="tldr"><strong>요약.</strong> aiportal-py: 자율 개선 회차 9회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>9</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>6</b><span>병합 완료</span></li><li><b>2</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>1</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$5.95</b><span>비용</span></li><li><b>15분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/aiportal-py">https://github.com/hkjang/aiportal-py</a></dd>
<dt>마지막 회차</dt><dd>2026-09-06 12:52 KST — <span class="pill pill-other">• 기타</span> CI no-ci, PR open <a href="https://github.com/hkjang/aiportal-py/pull/8">PR #8</a></dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>릴리즈 이력이 전혀 없는 저장소입니다. 확인 결과: git tag 0개(로컬·origin ls-remote 모두 없음), gh release list 결과 없음, CHANGELOG.md/docs/RELEASE*.md 등 릴리즈 노트 없음, 버전 파일 없음(pyproject.toml/setup.py/setup.cfg/package.json/VERSION/Chart.yaml/Cargo.toml 모두 부재), .github/workflows 디렉터리 자체가 없어 태그·릴리즈에 반응하는 워크플로 없음, scripts/ 나 Makefile 등 릴리즈·오프라인 패키징 스크립트 없음, git log 전체 14개 커밋에 릴리즈/버전 형식 커밋 없음(first commit, docs, fix, merge PR 뿐). 배포는 .gitlab-ci.yml 이 main/develop 브랜치 푸시에 반응해 배포 디렉터리에서 git reset --hard 후 k8s rollout restart/podman restart 하는 방식이라 버전·태그 개념을 쓰지 않습니다. api.py:106 의 FastAPI version=&quot;1.0.0&quot; 은 첫 커밋 이후 한 번도 변경된 적 없는 앱 메타데이터이며 릴리즈 절차와 무관합니다. docs/ROADMAP.md 의 R0~R5 는 버전이 아니라 로드맵 단계 표기입니다. 따를 기존 관례가 없어 새 관례를 만들지 않고 건너뜁니다(관례 수립은 사람의 결정 사항). 커밋·태그·파일 변경 없음.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt" data-filter="1"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-06 12:52</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> CI no-ci, PR open <a href="https://github.com/hkjang/aiportal-py/pull/8">PR #8</a><div class="meta">7파일 <span style="color:var(--good)">+427</span>/<span style="color:var(--bad)">−29</span> · 테스트 1 — fix: 추천 질문 캐시를 원자적 교체 방식으로 변경 (감사 A-206)</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 02:31</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> CI no-ci, PR open <a href="https://github.com/hkjang/aiportal-py/pull/7">PR #7</a><div class="meta">7파일 <span style="color:var(--good)">+97</span>/<span style="color:var(--bad)">−14</span> · 테스트 1 — fix: 오류 경로의 미할당 이름 참조 수정 (감사 A-105)</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-05 01:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/6">PR #6</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-04 06:09</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/5">PR #5</a>, release skipped</td></tr><tr data-status="nochange"><td data-label="일시">2026-09-03 22:23</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> no change</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 14:06</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 02:07</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/3">PR #3</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 19:48</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 13:45</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">12:48</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="단계">review</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">20</td><td data-label="비용" class="num">$0.92</td><td data-label="토큰 입력/출력" class="num">729K / 8K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">12:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">43</td><td data-label="비용" class="num">$2.36</td><td data-label="토큰 입력/출력" class="num">2.1M / 25K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">02:27</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="단계">review</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">11</td><td data-label="비용" class="num">$0.66</td><td data-label="토큰 입력/출력" class="num">378K / 7K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">02:25</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">47</td><td data-label="비용" class="num">$2.01</td><td data-label="토큰 입력/출력" class="num">1.8M / 22K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 11 / 전체 12

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가</td><td data-label="가치/위험/크기">4/3/M</td><td data-label="상태">대기</td><td data-label="메모">lifespan 이 while token_store.policy_token is None 에서 timeout 없이 0.5초 sleep 을 반복한다. 총 대기 제한·exponential backoff·degraded mode 필요. 기동 동작 변경이라 운영 검증 필요</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">A-105 후속: 공통 오류 응답 model 도입과 traceback 노출 제거</td><td data-label="가치/위험/크기">4/3/L</td><td data-label="상태">대기</td><td data-label="메모">미할당 이름 참조는 해결됐으나 HTTP 200+code 299/499, 빈 문자열, None, HTTP 500 혼재는 그대로. 클라이언트에 traceback 을 그대로 돌려주는 것도 A-106 과 연계해 정리 필요. 응답 형태 변경이라 클라이언트 영향 큼</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">GitLab CI 에 deploy 앞단 test stage 추가</td><td data-label="가치/위험/크기">4/3/L</td><td data-label="상태">대기</td><td data-label="메모">러너가 shell 태그 기반이라 python 실행 환경이 불확실하고, 실패 시 배포가 막힐 위험이 있어 러너 환경 확인이 선행돼야 함</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">pipeline/law/library/xml_parser_core.py, json_formatter.py 단위 테스트</td><td data-label="가치/위험/크기">3/1/M</td><td data-label="상태">대기</td><td data-label="메모">fixture XML/JSON 을 만들어야 함. 순수 변환 로직이라 외부 의존성 없이 테스트 가능</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">util/extract_minor.py 의 except Exception: return e 정리 및 테스트</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">예외 객체를 값처럼 반환해 호출부가 성공/실패를 구분하지 못함. 호출부 확인 후 명시적 오류 반환으로 교체</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">A-104 후속: collection name allowlist 와 field 별 타입 검증</td><td data-label="가치/위험/크기">3/2/M</td><td data-label="상태">대기</td><td data-label="메모">표현식 escape 는 끝났으나 요청이 지정하는 collection/table 이름은 여전히 검증 없이 Collection() 에 전달됨</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">A-102/A-206 후속: 추천 질문·토큰 캐시를 프로세스 간 공유</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">캐시 교체는 정리됐지만 여전히 worker 프로세스마다 별도 캐시라 worker 수만큼 LLM 호출이 중복되고 응답이 worker 별로 다르다. 외부 cache 또는 sticky routing 이 필요. 신규(2026-09-06)</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">A-108 Confluence 그래프의 retrieve → llm_response → END 명시적 edge 추가</td><td data-label="가치/위험/크기">3/3/S</td><td data-label="상태">대기</td><td data-label="메모">util/cf_workflow.py 가 entry point 를 retrieve 로 두지만 후속 edge 가 없음. langgraph 가 설치돼 있지 않아 실제 실행 확인이 선행돼야 함</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">A-107 후속: 재시도·circuit breaker·공용 requests.Session 도입</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">timeout 은 전부 지정됐으나 재시도·backoff·connection pooling 은 없음. idempotent 호출에만 제한 재시도를 붙이는 설계가 필요</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">except 절 범위 축소 (bare except → 구체 예외)</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">저장소 전반의 bare except 가 KeyboardInterrupt/CancelledError/GeneratorExit 까지 삼킨다. /status_completions 와 추천 질문 갱신 루프는 좁혔음. 나머지는 예외 종류를 확인해야 하므로 파일별로 나눠 진행</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">api.py 의 미사용 import 정리 (A-204 일부)</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">pyflakes 가 api.py 에서만 20건 이상의 &#x27;imported but unused&#x27; 를 낸다. 정리하면 pyflakes 를 CI 게이트로 쓸 수 있다. 다만 side-effect import 여부를 하나씩 확인해야 함. 신규(2026-09-06)</td><td data-label="갱신">2026-09-06</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거·최대 크기</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">완료</td><td data-label="메모">util/question_cache.py (QuestionCache.replace/sample/updated_at, extract_question_list) 추가. api.py 갱신 루프를 교체 방식으로 바꾸고 앱별 실패 격리·빈 결과 시 기존 캐시 유지. 실패 sentinel 문자열이 글자 단위로 저장되던 버그와 str+list TypeError 로 태스크가 죽던 버그도 해결. tests/unit/test_question_cache.py 추가. 커밋 0ab2740</td><td data-label="갱신">2026-09-06</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-02
- 선택: mask_sensitive_data NameError 수정 + pytest 테스트 기반 도입 (가치 5 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `util/sensitive_data_masker.py` 의 `mask_sensitive_data()` 가 dict/list 입력에서 존재하지 않는 `_mask_sensitive_data()` 를 호출해 NameError 로 실패하는 실제 버그를 재현·수정했다. 동시에 저장소에 전혀 없던 테스트 기반(`pytest.ini`, `tests/conftest.py`, `requirements-dev.txt`)을 만들고 외부 의존성 없는 순수 unit 테스트(sensitive_data_masker, parse_promptmanager, dpe_merge, text_renderer)와 저장소 전체 Python AST parse 테스트를 추가했다. `python -m pytest` 로 177 passed 확인, README/docs/TESTING.md 를 실제 상태에 맞게 갱신. 커밋 `8122984`.
- 보류 아이디어:
  - GitLab CI 에 deploy 앞단 test stage 추가 — 가치 4 / 위험 3 / L (러너가 shell 태그 기반이라 python 실행 환경 불확실, 실패 시 배포 차단 위험)
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - feedback loop 의 undefined `ENV` 수정 (감사 A-003) — 가치 4 / 위험 2 / S
  - `pipeline/law/library/xml_parser_core.py`, `json_formatter.py` 단위 테스트 (fixture 필요) — 가치 3 / 위험 1 / M
  - `util/extract_minor.py` 의 `except Exception: return e` (예외 객체 반환) 정리 및 테스트 — 가치 3 / 위험 2 / S

## 2026-09-02 (2회차)
- 선택: 정의되지 않은 이름으로 인한 런타임 오류 일괄 수정 + 정적 회귀 테스트 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: pyflakes 로 저장소를 훑어 NameError/UnboundLocalError 를 일으키는 실제 결함 전부(감사 A-003 undefined `ENV`, A-004 `Collection(collection)`, api.py 의 `payload` 선참조·typing import 누락, kai/cf workflow 의 `guard_rail_completions_*` import 누락, chunker 의 `traceback` import 및 `text_chunking(file)` 인자 누락, truncate_table 의 rows/collection_name 참조)를 수정했다. 함께 A-005(`close_all_db_connections.closeall()` → 함수 호출)와 A-109(finally 가 try 지역 `conn` 참조하는 8곳에 `conn = None` 초기화, `fetch_unprocessed_feedback` 의 중복 `conn.close()` 제거)도 정리했다. 회귀 방지로 운영 의존성 없이 AST 만 쓰는 `tests/unit/test_undefined_names.py`(pyflakes 기반, 미설치 시 skip)와 `tests/unit/test_cleanup_paths.py`(finally/종료 경로 검사)를 추가하고 requirements-dev 에 pyflakes 를 넣었다. `python -m pytest` 352 passed(기존 177 → 352), `python -m pyflakes .` 의 undefined name 0건 확인. docs/TESTING.md·CURRENT_STATE_AUDIT.md 를 실제 상태로 갱신. 커밋 `89cf479`.
- 보류 아이디어:
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M (기동 동작 변경이라 운영 검증 필요)
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S
  - `pipeline/law/library/xml_parser_core.py`, `json_formatter.py` 단위 테스트 (fixture 필요) — 가치 3 / 위험 1 / M
  - `util/extract_minor.py` 의 `except Exception: return e` (예외 객체 반환) 정리 및 테스트 — 가치 3 / 위험 2 / S

## 2026-09-03
- 선택: 외부 HTTP 호출 timeout 누락 일괄 수정 (감사 A-107) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `requests` 는 timeout 이 없으면 무한 대기하는데, FastAPI 라우트와 LangGraph 노드가 이 호출을 동기로 하므로 상대 서비스가 멈추면 worker 가 그대로 묶인다. 설정 모듈에 의존하지 않는 `util/http_timeouts.py`(DEFAULT 5/30s, LLM 5/300s, FILE 5/600s, 환경변수로 조정 가능)를 추가하고 timeout 이 없던 호출 53곳(api.py, service/*, util/*, pipeline/*, config/eval_class.py)에 용도별 값을 지정했다. AST 로 저장소 전체의 `requests`/`Session` 호출을 검사해 timeout 누락·`timeout=None` 재발을 막는 `tests/unit/test_http_timeouts.py` 를 추가했고, `python -m pytest` 450 passed(기존 352 → 450), `python -m pyflakes .` undefined name 0건 확인. docs 3종(CURRENT_STATE_AUDIT/TESTING/CONFIGURATION) 갱신. 커밋 `be6f163`.
- 보류 아이디어:
  - A-104 Milvus filter 표현식 직접 조립 → 안전한 expression builder + 입력 검증 — 가치 4 / 위험 3 / M
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S
  - A-107 후속: 재시도·circuit breaker·공용 requests.Session 도입 — 가치 3 / 위험 3 / M

## 2026-09-03 (2회차)
- 선택: Milvus filter 표현식 주입 방지용 expression builder (감사 A-104) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: session id, filename, page, doc id, space key 등 외부 입력이 f-string 으로 Milvus 표현식에 그대로 들어가, filename 이 `a' or session_id != '` 이면 다른 세션 파일까지 삭제되는 구조였다. field 이름을 식별자로 제한하는 `field()` 와 `\`/`'`/`"`/개행을 escape 하는 `quote()`, `eq/ne/lt/in_/and_` 를 가진 `util/milvus_expr.py` 를 추가하고 조립 지점 전부(util/search_module, milvus_collection, milvus_confluence, milvus_batch, pipeline/kai·kcb·law)를 교체했다. 요청 body 의 `req.key_field` 가 field 이름 자리에 들어가던 경로와, `f"...'{a}'" + f"and ..."` 로 이어 붙여 `'value'and` 처럼 공백이 빠지던 버그 2곳도 함께 고쳤다. 기존 코드가 모두 문자열 literal 비교였으므로 builder 도 값을 항상 문자열로 인용해 의미를 바꾸지 않는다. 단위 테스트(`test_milvus_expr.py`)와 저장소 전체 AST 회귀 테스트(`test_milvus_expr_usage.py`)를 추가해 `python -m pytest` 574 passed(기존 450), `python -m pyflakes .` undefined name 0건 확인. docs 4종(CURRENT_STATE_AUDIT/TESTING/MILVUS_SEARCH/SQL_SECURITY) 갱신. 커밋 `53ce1f7`.
- 보류 아이디어:
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
  - `collection_script/*` 의 import-time collection create/drop 부작용 제거 (감사 A-006) — 가치 4 / 위험 3 / M
  - A-105 오류 응답 계약 통일 (except block 의 미할당 `response` 포함) — 가치 4 / 위험 3 / L
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S
  - A-104 후속: collection name allowlist 와 field별 타입 검증 — 가치 3 / 위험 2 / M

## 2026-09-04
- 선택: collection_script 의 import-time 컬렉션 생성·삭제 제거 (감사 A-006) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `collection_script/` 의 여러 스크립트가 모듈 최상위에서 Milvus 에 연결하고 컬렉션을 만들거나 지웠다. 특히 `api.py` 의 `/create_filestorage_collection` 이 라우트 안에서 `create_collection_FILE_STORAGE` 를 import 하므로 라우트 호출만으로 최상위 `create("FILE_STORAGE")` 가 실행됐고, `delete_collection.py` 는 import 시점에 `KCBLAW_VIEW_BOX` 를 drop 했다. 호출 시점에만 연결하는 `connect()` 와 파괴적 작업 확인용 `confirm()` 을 가진 `collection_script/common.py` 를 추가하고, 최상위 실행(FILE_STORAGE·CONFLUENCE·KAI·KCBLAWVIEW·delete_collection)을 argparse 기반 `main()` 과 `__main__` 가드로 옮겼다. 하드코딩된 Milvus 주소 3곳(192.168.120.99×2, 192.168.116.99)을 `connect()` 로 교체하고, KAI 스크립트에 중복 정의된 컬렉션 목록을 `configs.local_variable.COLLECTION_LIST_KAI` 로 통일했으며, `delete_collection` 은 컬렉션 이름을 CLI 인자로 받고 `--yes` 없이는 확인을 요구한다. `api.py` 가 쓰는 `create`/`create_law_detail`/`create_law_viewer` signature 는 유지했다. AST 만 쓰는 `tests/unit/test_collection_script_side_effects.py`(import 시점 부작용 호출·최상위 실행 구문·하드코딩 주소·삭제 확인 절차)를 추가했고, 옛 코드를 되돌려 넣어 실제로 실패하는지 확인했다. `python -m pytest` 611 passed(기존 574), `python -m pyflakes .` undefined name 0건. docs 5종(CURRENT_STATE_AUDIT/CODEBASE_MAP/INSTALL/TROUBLESHOOTING/TESTING) 갱신. 커밋 `8ae2a04`.
- 보류 아이디어:
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
  - A-101 워크플로 모듈 전역 `original_question` 제거 → GraphState 로 전달 (동시 요청 간 질문 오염) — 가치 4 / 위험 3 / M
  - A-105 오류 응답 계약 통일 (except block 의 미할당 `response` 포함) — 가치 4 / 위험 3 / L
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거 — 가치 3 / 위험 2 / S

## 2026-09-05
- 선택: 워크플로 전역 `original_question` 제거 → GraphState 로 전달 (감사 A-101) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: KAI·KCB·법률·Confluence 4개 LangGraph 워크플로의 entry function 이 모듈 전역 `original_question` 을 요청마다 덮어쓰고 노드들이 그 전역을 읽고 있었다. 같은 프로세스에서 요청이 겹치면 A 사용자의 질문이 B 사용자의 `guard_rail`/`query_extension`/`eval_weight_search`/`intro`/`check_search` 에서 쓰여 가드레일 분기·검색 가중치·조항 조회가 엉뚱한 질문 기준으로 수행된다. `retrieve` 이후 노드가 `question` 을 확장 질의로 덮어쓰기 때문에 기존 `question` 필드만으로는 원본 질문을 보관할 수 없어, `original_question` 전용 필드를 각 `GraphState` 에 추가하고 helper `util/workflow_state.py`(`with_original_question()` 으로 inputs 에 주입, `get_original_question(state)` 로 읽되 없으면 `question` 으로 fallback)를 도입했다. `global` 선언 4곳과 전역 읽기 8곳을 전부 교체하고, `kai_workflow` 의 미사용 전역 `original_question`·`last_rewritten` 도 제거했다. 진입점이 `*_langgraph` 4개뿐임을 확인해 호출 계약은 그대로다. `langgraph` 가 설치돼 있지 않아 워크플로 모듈은 import 할 수 없으므로, helper 는 실제 동시 실행(asyncio.gather) 테스트로(`tests/unit/test_workflow_state.py`), 워크플로 모듈은 AST 검사로(`tests/unit/test_workflow_globals.py`: `global` 선언 금지, 모듈 전역 요청 상태 금지, 노드의 전역 읽기 금지, 진입점의 `with_original_question()` 호출, `GraphState` 필드 선언) 검증했다. 옛 `law_workflow.py` 를 되돌려 넣어 신규 테스트 4건이 실제로 실패하는지 확인했다. `python -m pytest` 654 passed(기존 611), `python -m pyflakes .` undefined name 0건. docs 3종(CURRENT_STATE_AUDIT/LANGGRAPH_WORKFLOW/TESTING) 갱신. 커밋 `f4fb71b`.
- 보류 아이디어:
  - A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
  - A-105 오류 응답 계약 통일 (except block 의 미할당 `response` 포함) — 가치 4 / 위험 3 / L
  - A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거·최대 크기 — 가치 3 / 위험 2 / S
  - A-108 Confluence 그래프의 `retrieve → llm_response → END` 명시적 edge 추가 — 가치 3 / 위험 3 / S (LangGraph 실행 확인 필요)
## 2026-09-06
- 선택: 오류 경로의 미할당 이름 참조 수정 (감사 A-105 부분 해결) (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `api.py` 의 5개 라우트(`/newchatsession`, `/getrefreshtoken`, `/history_list`, `/delete_history_session`, `/history_session`)가 `except` 에서 `response.json()` 을 호출했는데, 서비스 호출 자체가 실패하면 `response` 가 대입되지 않아 `UnboundLocalError` 가 원래 오류를 덮고 HTTP 500 이 됐다(history 계열은 성공 경로가 dict 를 반환하므로 `response` 가 있어도 `.json()` 이 없어 항상 실패). 저장소 표준인 `{"code":499, "message": traceback.format_exc()}` 로 교체하고, `/status_completions` 의 스트리밍 generator 는 `last_state` 를 `try` 밖에서 초기화하고 오류 시 종료 프레임을 실제로 `yield` 하며 `GeneratorExit`/`CancelledError` 를 삼키지 않도록 `except Exception` 으로 좁혔다. 같은 패턴인 `kcb_pipeline_milvus.remove_legacy_file` 의 `collection_name` 과 `milvus_collection.sync_collection` 의 `response` 초기화(및 `list.append` 결과를 재대입해 `data` 가 `None` 이 되던 버그, 반환 arity 불일치)도 함께 고쳤다. 회귀 방지로 기존 `tests/unit/test_cleanup_paths.py` 에 `finally` 검사와 같은 스코프 분석을 쓰는 `except` 블록 검사와 검사기 자체 self-test 2건을 추가했다(수정 전 저장소에서 7건 검출 확인). `python -m pytest` 751 passed(기존 654), `python -m pyflakes .` undefined name 0건. docs 3종(CURRENT_STATE_AUDIT/API_REFERENCE/TESTING) 갱신. 커밋 `4587666`.
- 보류 아이디어: A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
- 보류 아이디어: A-105 후속 — 공통 오류 응답 model 도입과 traceback 노출 제거(A-106 연계) — 가치 4 / 위험 3 / L
- 보류 아이디어: A-206 추천 질문 캐시 무한 append → 원자적 교체·중복 제거·최대 크기 — 가치 3 / 위험 2 / S
- 보류 아이디어: A-108 Confluence 그래프의 `retrieve → llm_response → END` 명시적 edge 추가 — 가치 3 / 위험 3 / S (LangGraph 실행 확인 필요)

## 2026-09-06
- 선택: 추천 질문 캐시 무한 append → 원자적 교체 (감사 A-206) (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `api.py` 의 `QuestionStore.auto_refresh_question` 이 30분마다 수집 결과를 기존 리스트에 `append` 만 해서 프로세스가 오래 살수록 캐시가 무한히 커지고 같은 질문이 갱신 횟수만큼 중복 저장됐다. 더 심각하게는 `question_recommend` 가 실패 시 `{"questions": "질문 생성 실패"}` 라는 **문자열**을 돌려주는데 루프가 이를 그대로 순회해 한 글자짜리 질문이 캐시에 쌓였고, 피드백 질문 병합 단계에서는 `str + list` 로 `TypeError` 가 나 백그라운드 태스크가 조용히 죽어 캐시가 영구히 멈췄다. 원자적 교체·중복 제거·최대 크기(`QUESTION_CACHE_MAX`, 기본 500)·`updated_at`·`sample()` 을 가진 `util/question_cache.py` 와 응답에서 리스트일 때만 질문을 꺼내는 `extract_question_list()` 를 추가하고, 갱신 루프는 앱별 수집을 `_collect_for_app()` 으로 분리해 앱 하나가 실패해도 나머지 앱과 다음 주기가 계속되게 했으며 수집 결과가 비면 기존 캐시를 유지한다. `question_store.questions` 는 스냅샷 property 로 남겨 읽기 경로 계약을 유지했다. `api.py` 는 kiwipiepy 미설치로 import 할 수 없어 캐시 모듈은 단위 테스트로, 갱신 루프는 AST 검사(append 금지·`cache.replace()` 존재·bare except 금지·헬퍼 import)로 검증하는 `tests/unit/test_question_cache.py` 를 추가했고, 옛 `api.py` 를 되돌려 넣어 정적 테스트 4건이 실제로 실패하는지 확인했다. `python -m pytest` 691 passed(기존 654), `python -m pyflakes .` undefined name 0건. docs 4종(CURRENT_STATE_AUDIT/API_REFERENCE/CONFIGURATION/TESTING) 갱신. 커밋 `0ab2740`.
- 보류 아이디어: A-002 startup 무기한 대기(policy token) 에 timeout/backoff 추가 — 가치 4 / 위험 3 / M
- 보류 아이디어: A-105 후속 — 공통 오류 응답 model 도입과 traceback 노출 제거(A-106 연계) — 가치 4 / 위험 3 / L
- 보류 아이디어: A-102/A-206 후속 — 추천 질문 캐시를 프로세스 간 공유(외부 cache 또는 sticky routing) — 가치 3 / 위험 3 / M
- 보류 아이디어: `util/extract_minor.py` 의 `except Exception: return e` (예외 객체 반환) 정리 및 테스트 — 가치 3 / 위험 2 / S
- 보류 아이디어: A-108 Confluence 그래프의 `retrieve → llm_response → END` 명시적 edge 추가 — 가치 3 / 위험 3 / S (LangGraph 실행 확인 필요)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
