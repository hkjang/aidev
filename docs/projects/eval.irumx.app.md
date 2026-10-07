---
title: "eval.irumx.app — 자율 개선 이력"
description: "eval.irumx.app: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-10-07 15:22:49 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "eval.irumx.app",
 "codeRepository": "https://github.com/hkjang/eval.irumx.app",
 "url": "https://hkjang.github.io/aidev/projects/eval.irumx.app/",
 "description": "eval.irumx.app: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-10-07T15:22:49+09:00"
}
</script>

# eval.irumx.app

<p class="tldr"><strong>요약.</strong> eval.irumx.app: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>1</b><span>실행 오류</span></li><li><b>$2.04</b><span>비용</span></li><li><b>5분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/eval.irumx.app">https://github.com/hkjang/eval.irumx.app</a></dd>
<dt>마지막 회차</dt><dd>2026-10-07 14:46 KST — <span class="pill pill-other">• 기타</span> error: agent produced no result ()</dd>
<dt>수정 과제</dt><dd>⚠️ 오류 대응(자동 적재): 마지막 회차가 &#x27;error&#x27; 로 끝났습니다. error: agent produced no result ()</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-10-07 14:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/eval.irumx.app/">eval.irumx.app</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> error: agent produced no result ()</td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">14:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/eval.irumx.app/">eval.irumx.app</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">0분</td><td data-label="턴" class="num">0</td><td data-label="비용" class="num">$0.00</td><td data-label="토큰 입력/출력" class="num">0 / 0</td><td data-label="종료">unknown</td></tr><tr data-status="other"><td data-label="시각">14:32</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/eval.irumx.app/">eval.irumx.app</a></td><td data-label="단계">정찰</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">29</td><td data-label="비용" class="num">$2.04</td><td data-label="토큰 입력/출력" class="num">1.6M / 15K</td><td data-label="종료">error_max_budget_usd</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">예산·시험 수 한도 동시 예약 시험(수용 시험 12)</td><td data-label="가치/위험/크기">4/2/S</td><td data-label="상태">대기</td><td data-label="메모">이번 회차 선택. docs/spec-coverage.md 12번이 🟡 — 순차 시험만 있다. tests/api-targets.spec.ts 에 Promise.all 동시 5건 + GET /api/w/:ws/usage 로 합계 검증. 프로덕션 코드 0개.</td><td data-label="갱신">2026-10-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">큐 중복 전달 시험(수용 시험 10)</td><td data-label="가치/위험/크기">4/3/M</td><td data-label="상태">대기</td><td data-label="메모">engine.ts processTrial 의 lease·generation 가드는 있지만 같은 메시지를 두 번 보내는 시험이 없다. 시험에서 큐에 직접 두 번 넣는 경로가 있는지 미확인 — 없으면 시험용 훅이 필요해 위험이 올라간다.</td><td data-label="갱신">2026-10-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">개발/보류(holdout) 사례 묶음 구분</td><td data-label="가치/위험/크기">3/3/L</td><td data-label="상태">대기</td><td data-label="메모">docs/spec-coverage.md 14장 ⬜. 데이터셋·실험·판단 화면·보고서까지 번지는 기능으로 파일 6개를 넘긴다 — 한 회차 과제로 부적합, 쪼개야 함.</td><td data-label="갱신">2026-10-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">모델 출력 속 비밀 모양 문자열 자동 가림(수용 시험 23)</td><td data-label="가치/위험/크기">3/4/M</td><td data-label="상태">대기</td><td data-label="메모">대응표 23번 🟡. 다만 이 서비스는 합성 자료의 &#x27;표식(marker) 노출&#x27;을 판정 근거로 쓴다 — 자동 가림이 판정기 역검증(40개)을 망칠 수 있다. 가림 범위를 원문·보고서 출력 경로로만 좁히지 않으면 위험. 운영자 교훈(&#x27;감사 details 에 검출 대상 문자열을 넘기지 말 것&#x27;)과 같은 결.</td><td data-label="갱신">2026-10-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">util.ts 주석의 없는 시험 파일 참조 정리(tests/guard.spec.ts)</td><td data-label="가치/위험/크기">1/1/S</td><td data-label="상태">대기</td><td data-label="메모">src/worker/lib/util.ts:76 주석이 tests/guard.spec.ts 를 가리키지만 그 파일은 없다(가드는 experiments·engine·usage 의 batch 경로에서 간접적으로만 확인됨). 과제로는 너무 작아 다른 변경에 끼워 갈 것.</td><td data-label="갱신">2026-10-07</td></tr><tr data-status="failed"><td data-label="아이디어" class="primary">GitHub Actions CI 추가(빌드·시험 자동화)</td><td data-label="가치/위험/크기">2/4/M</td><td data-label="상태">기각</td><td data-label="메모">.github 가 아예 없고 시험이 wrangler dev + 가짜 서버 4개를 띄워야 한다(수 분). 운영자가 workflows 경로는 피하라고 반복 지시 — 기각.</td><td data-label="갱신">2026-10-07</td></tr></tbody></table></div>


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
