---
title: "aidev 자율 개선 대시보드"
description: "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 1회차·릴리즈 0건, 누적 285회차·릴리즈 145건, 주의 필요 6건."
last_modified_at: 2026-09-06 00:17:21 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "WebSite",
 "name": "aidev 자율 개선 대시보드",
 "url": "https://hkjang.github.io/aidev/",
 "description": "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 1회차·릴리즈 0건, 누적 285회차·릴리즈 145건, 주의 필요 6건.",
 "inLanguage": "ko",
 "author": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-06T00:17:21+09:00"
}
</script>

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "FAQPage",
 "mainEntity": [
  {
   "@type": "Question",
   "name": "이 페이지는 무엇인가요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다."
   }
  },
  {
   "@type": "Question",
   "name": "얼마나 자주 갱신되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다. Atom 피드(/feed.xml)를 구독하면 일일 보고를 받아볼 수 있습니다."
   }
  },
  {
   "@type": "Question",
   "name": "한 회차에서 에이전트는 무엇을 하나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열고 CI 통과를 확인한 뒤 머지하며, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로·첨부 자산)을 확인해 같은 방식으로 다음 버전을 냅니다."
   }
  },
  {
   "@type": "Question",
   "name": "릴리즈 워크플로가 실패하면 어떻게 되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "러너가 한 번 자동으로 재실행합니다. 그래도 실패하면 실패 단계와 로그 요지를 수정 과제 큐에 넣고, 다음 회차에서 그 프로젝트를 우선 배정해 에이전트가 원인을 고칩니다(검증을 느슨하게 만드는 것은 금지). 큐에 있는 동안 '주의 필요'에 표시됩니다."
   }
  },
  {
   "@type": "Question",
   "name": "어떤 프로젝트가 대상인가요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다."
   }
  },
  {
   "@type": "Question",
   "name": "'머지(릴리즈 없음)'는 무슨 뜻인가요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다."
   }
  },
  {
   "@type": "Question",
   "name": "'주의 필요'에는 무엇이 뜨나요? 알림은 어디로 오나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "최근 2일 회차 중 CI 실패로 머지되지 않은 PR, 릴리즈 실패, 이전 릴리즈에는 있던 첨부 자산이 빠진 릴리즈, 수정 과제 대기처럼 사람이 확인해야 할 항목입니다. 새 경고가 생기면 GitHub Issue(라벨 alert)에 기록하고, 설정돼 있으면 Slack·이메일·Windows 알림으로도 보냅니다."
   }
  },
  {
   "@type": "Question",
   "name": "머지 전에 검토는 없나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "있습니다. 구현과 다른 세션의 리뷰 에이전트가 diff 만 읽고 '머지하면 안 되는 이유'(검증하지 않는 테스트, 논리 오류, 설명과 다른 동작, 위험한 변경)를 찾습니다. 거절하면 PR 에 사유를 달고 열어 둡니다. 또 워크플로·마이그레이션·인증·결제·배포 파일(보호 파일)을 건드린 PR 은 자동 머지하지 않습니다."
   }
  },
  {
   "@type": "Question",
   "name": "깨진 변경은 어떻게 되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "머지 2시간 뒤부터 main CI 실패와 되돌림 커밋을 확인해 '교훈'으로 기록하고, 그 프로젝트의 다음 회차 프롬프트에 주입해 같은 실수를 피하게 합니다. 릴리즈 워크플로가 반복 실패하고 수정 회차도 실패하면 원래 머지를 되돌리는 롤백 PR 을 자동으로 엽니다(머지는 사람). 프로젝트마다 최근 14일의 실패·경고·회귀로 건강 등급 A~D 를 매깁니다."
   }
  },
  {
   "@type": "Question",
   "name": "자율화 단계란 무엇인가요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "프로젝트마다 러너 권한을 '분석만 → PR 생성 → 승인 후 병합 → 저위험 자동 병합 → 검증된 릴리즈 게시' 다섯 단계로 나눕니다. 단계를 올리는 것은 사람이 정책 파일(state/<프로젝트>.policy.json 의 autonomy)을 고쳐야 하고, 롤백이나 회귀가 생기면 러너가 한 단계 내립니다(⬇ 표시). 작업함에서 승인(aidev-approved)·반려(aidev-rejected) 라벨로 개별 PR 을 처리할 수 있습니다."
   }
  },
  {
   "@type": "Question",
   "name": "긴급히 멈추려면?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "bin/stop.sh all|merge|release|<프로젝트> on \"사유\" 또는 aidev 저장소에 라벨 stop, 제목 stop: <범위> 이슈를 만들면 됩니다. 새 회차 시작뿐 아니라 진행 중인 회차도 에이전트 시작 전·머지 전·릴리즈 전 경계에서 멈춥니다."
   }
  },
  {
   "@type": "Question",
   "name": "비용은 어떻게 계산되나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "각 회차의 claude -p 세션이 보고한 추정 비용(USD)과 소요 시간·턴 수·토큰을 그대로 합산합니다. 정액제 구독에서는 실제 청구가 아닌 참고값입니다."
   }
  },
  {
   "@type": "Question",
   "name": "원본 데이터는 어디서 보나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "회차 기록은 runs.jsonl, 사용량은 usage.jsonl, 요약은 data/summary.json, 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다."
   }
  }
 ]
}
</script>

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "ItemList",
 "name": "일일 보고",
 "itemListOrder": "Descending",
 "itemListElement": [
  {
   "@type": "ListItem",
   "position": 1,
   "name": "일일 보고 2026-09-06",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-06/"
  },
  {
   "@type": "ListItem",
   "position": 2,
   "name": "일일 보고 2026-09-05",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-05/"
  },
  {
   "@type": "ListItem",
   "position": 3,
   "name": "일일 보고 2026-09-04",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-04/"
  },
  {
   "@type": "ListItem",
   "position": 4,
   "name": "일일 보고 2026-09-03",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-03/"
  },
  {
   "@type": "ListItem",
   "position": 5,
   "name": "일일 보고 2026-09-02",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-02/"
  }
 ]
}
</script>

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "Dataset",
 "name": "aidev 회차 기록 (runs.jsonl)",
 "description": "자율 개선 에이전트의 회차별 기록. 한 줄에 한 회차, 필드: ts, date, project, result.",
 "url": "https://hkjang.github.io/aidev/data/runs.jsonl",
 "license": "https://opensource.org/license/mit",
 "inLanguage": "ko",
 "creator": {
  "@type": "Person",
  "name": "hkjang"
 },
 "encodingFormat": "application/x-ndjson",
 "distribution": [
  {
   "@type": "DataDownload",
   "encodingFormat": "application/x-ndjson",
   "contentUrl": "https://hkjang.github.io/aidev/data/runs.jsonl"
  },
  {
   "@type": "DataDownload",
   "encodingFormat": "application/x-ndjson",
   "contentUrl": "https://hkjang.github.io/aidev/data/usage.jsonl"
  },
  {
   "@type": "DataDownload",
   "encodingFormat": "application/json",
   "contentUrl": "https://hkjang.github.io/aidev/data/summary.json"
  }
 ]
}
</script>

# aidev 자율 개선 대시보드

<p class="tldr"><strong>한 줄 요약.</strong> Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 1회차·릴리즈 0건, 누적 285회차·릴리즈 145건, 주의 필요 6건. 회차가 끝날 때마다 자동 갱신됩니다 (마지막 갱신 <time datetime="2026-09-06T00:17:21+09:00" data-rel>2026-09-06 00:17</time> KST).</p>

<div class="alerts" role="alert"><strong>⚠️ 주의 필요 6건</strong> <span class="meta">— 새 경고는 GitHub Issue·Slack·이메일·Windows 알림으로도 보냅니다</span><ul><li><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a> — 릴리즈 v0.1.21 가 GitHub 에 없음 — 워크플로 Release offline image: failure <span class=meta>(2026-09-05 05:15)</span></li><li><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> — 릴리즈 v1.11.16 가 GitHub 에 없음 — 워크플로 Release Offline Docker Image: failure <span class=meta>(2026-09-05 07:33)</span></li><li><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> — 릴리즈 v1.11.17 가 GitHub 에 없음 — 워크플로 Release Offline Docker Image: failure <span class=meta>(2026-09-05 08:12)</span></li><li><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-05 11:20)</span></li><li><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-05 16:44)</span></li><li><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a> — 최신 릴리즈 v0.234.0 자산 0개 (이전 v0.233.0: 10개)</li></ul></div>

<div class="alerts ok" role="status"><strong>🩺 러너 정상</strong> <span class="meta">— 마지막 회차 1시간 54분 전 · 스케줄러 실행 중 · 다음 실행 2026-09-06 오전 12:20:00 · 디스크 68% · 최근 7일 회귀 0건 · 점검 00:15</span></div>


<p><a href="https://hkjang.github.io/aidev/inbox/"><strong>📥 작업함</strong></a> — 사람 판단이 필요한 PR·복구·수정 과제 3건</p>

[운영 문서](https://github.com/hkjang/aidev#readme) · [원장](https://github.com/hkjang/aidev/tree/main/state) · [실행 이력](https://github.com/hkjang/aidev/commits/main) · [경고 이슈](https://github.com/hkjang/aidev/issues?q=label%3Aalert) · [교훈 0건](https://hkjang.github.io/aidev/lessons/) · [Atom 피드](https://hkjang.github.io/aidev/feed.xml) · [summary.json](https://hkjang.github.io/aidev/data/summary.json)

## 오늘 (2026-09-06)

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>1</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$6.62</b><span>비용</span></li><li><b>16분</b><span>에이전트 시간</span></li></ul>

[2026-09-06 보고 자세히 보기 →](https://hkjang.github.io/aidev/reports/2026-09-06/)

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">오늘 전체 회차 (KST)</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">00:17</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/Clustara/">Clustara</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> review held, PR open <a href="https://github.com/hkjang/clustara/pull/11">PR #11</a><div class="meta">9파일 <span style="color:var(--good)">+337</span>/<span style="color:var(--bad)">−74</span> · 테스트 1 — fix(capacity): finished pods held node capacity, cost and GPU slots (v0.9.272)</div></td></tr></tbody></table></div>

## 최근 14일

<div class="chart"><svg viewBox="0 0 640 200" role="img" aria-labelledby="chart-t chart-d"><title id="chart-t">최근 14일 회차 수</title><desc id="chart-d">날짜별 회차 수를 릴리즈·머지·변경 없음·실패로 나눠 쌓은 막대. 같은 값은 아래 표에 있다.</desc><line class="grid" x1="28" x2="636" y1="172.0" y2="172.0"/><text x="22" y="176.0" text-anchor="end">0</text><line class="grid" x1="28" x2="636" y1="155.8" y2="155.8"/><text x="22" y="159.8" text-anchor="end">10</text><line class="grid" x1="28" x2="636" y1="139.6" y2="139.6"/><text x="22" y="143.6" text-anchor="end">20</text><line class="grid" x1="28" x2="636" y1="123.4" y2="123.4"/><text x="22" y="127.4" text-anchor="end">30</text><line class="grid" x1="28" x2="636" y1="107.2" y2="107.2"/><text x="22" y="111.2" text-anchor="end">40</text><line class="grid" x1="28" x2="636" y1="91.0" y2="91.0"/><text x="22" y="95.0" text-anchor="end">50</text><line class="grid" x1="28" x2="636" y1="74.8" y2="74.8"/><text x="22" y="78.8" text-anchor="end">60</text><line class="grid" x1="28" x2="636" y1="58.6" y2="58.6"/><text x="22" y="62.6" text-anchor="end">70</text><line class="grid" x1="28" x2="636" y1="42.4" y2="42.4"/><text x="22" y="46.4" text-anchor="end">80</text><line class="grid" x1="28" x2="636" y1="26.2" y2="26.2"/><text x="22" y="30.2" text-anchor="end">90</text><line class="grid" x1="28" x2="636" y1="10.0" y2="10.0"/><text x="22" y="14.0" text-anchor="end">100</text><g><title>2026-08-24: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-25: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="92.7" y="186" text-anchor="middle">08/25</text></g><g><title>2026-08-26: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-27: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="179.0" y="186" text-anchor="middle">08/27</text></g><g><title>2026-08-28: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-29: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="265.3" y="186" text-anchor="middle">08/29</text></g><g><title>2026-08-30: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-08-31: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="351.6" y="186" text-anchor="middle">08/31</text></g><g><title>2026-09-01: 회차 0 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title></g><g><title>2026-09-02: 회차 50 · 릴리즈 28 · 머지 16 · 변경 없음 3 · 실패 0</title><rect x="425.9" y="126.6" width="24.0" height="43.4" fill="#0ca30c"/><rect x="425.9" y="100.7" width="24.0" height="23.9" fill="#fab219"/><path d="M425.9,100.7 V98.3 Q425.9,95.9 428.3,95.9 H447.4 Q449.9,95.9 449.9,98.3 V100.7 Z" fill="#9ca3af"/><text x="437.9" y="186" text-anchor="middle">09/02</text></g><g><title>2026-09-03: 회차 96 · 릴리즈 63 · 머지 25 · 변경 없음 8 · 실패 0</title><rect x="469.0" y="69.9" width="24.0" height="100.1" fill="#0ca30c"/><rect x="469.0" y="29.4" width="24.0" height="38.5" fill="#fab219"/><path d="M469.0,29.4 V20.5 Q469.0,16.5 473.0,16.5 H489.0 Q493.0,16.5 493.0,20.5 V29.4 Z" fill="#9ca3af"/></g><g><title>2026-09-04: 회차 77 · 릴리즈 33 · 머지 9 · 변경 없음 1 · 실패 0</title><rect x="512.1" y="118.5" width="24.0" height="51.5" fill="#0ca30c"/><rect x="512.1" y="104.0" width="24.0" height="12.6" fill="#fab219"/><path d="M512.1,104.0 V103.1 Q512.1,102.3 513.0,102.3 H535.3 Q536.1,102.3 536.1,103.1 V104.0 Z" fill="#9ca3af"/><text x="524.1" y="186" text-anchor="middle">09/04</text></g><g><title>2026-09-05: 회차 61 · 릴리즈 21 · 머지 6 · 변경 없음 1 · 실패 0</title><rect x="555.3" y="138.0" width="24.0" height="32.0" fill="#0ca30c"/><rect x="555.3" y="128.3" width="24.0" height="7.7" fill="#fab219"/><path d="M555.3,128.3 V127.4 Q555.3,126.6 556.1,126.6 H578.5 Q579.3,126.6 579.3,127.4 V128.3 Z" fill="#9ca3af"/></g><g><title>2026-09-06: 회차 1 · 릴리즈 0 · 머지 0 · 변경 없음 0 · 실패 0</title><text x="610.4" y="168.0" text-anchor="middle">1</text><text x="610.4" y="186" text-anchor="middle">09/06</text></g><line class="grid" x1="28" x2="636" y1="172" y2="172"/></svg><ul class="legend" aria-hidden="true"><li><i style="background:#0ca30c"></i>🚀 릴리즈</li><li><i style="background:#fab219"></i>✅ 머지</li><li><i style="background:#9ca3af"></i>➖ 변경 없음</li><li><i style="background:#d03b3b"></i>❌ 실패</li></ul></div>

## 일일 보고

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">날짜</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">머지</th><th class="num">변경 없음</th><th class="num">실패</th><th class="num">비용</th></tr></thead><tbody><tr data-status="nochange"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-06/">2026-09-06</a></td><td data-label="회차" class="num">1</td><td data-label="릴리즈" class="num">0</td><td data-label="머지" class="num">0</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$6.62</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-05/">2026-09-05</a></td><td data-label="회차" class="num">61</td><td data-label="릴리즈" class="num">21</td><td data-label="머지" class="num">6</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$50.01</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-04/">2026-09-04</a></td><td data-label="회차" class="num">77</td><td data-label="릴리즈" class="num">33</td><td data-label="머지" class="num">9</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-03/">2026-09-03</a></td><td data-label="회차" class="num">96</td><td data-label="릴리즈" class="num">63</td><td data-label="머지" class="num">25</td><td data-label="변경 없음" class="num">8</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-02/">2026-09-02</a></td><td data-label="회차" class="num">50</td><td data-label="릴리즈" class="num">28</td><td data-label="머지" class="num">16</td><td data-label="변경 없음" class="num">3</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr></tbody></table></div>

## 주간·월간 보고

### 주간

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">주</th><th class="num">활동일</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">실패</th><th class="num">비용</th><th class="num">시간</th></tr></thead><tbody><tr data-status="released"><td data-label="주" class="primary"><a href="https://hkjang.github.io/aidev/weekly/2026-W36/">2026-W36</a></td><td data-label="활동일" class="num">5</td><td data-label="회차" class="num">285</td><td data-label="릴리즈" class="num">145</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$56.63</td><td data-label="시간" class="num">2시간 17분</td></tr></tbody></table></div>

### 월간

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">월</th><th class="num">활동일</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">실패</th><th class="num">비용</th><th class="num">시간</th></tr></thead><tbody><tr data-status="released"><td data-label="월" class="primary"><a href="https://hkjang.github.io/aidev/monthly/2026-09/">2026-09</a></td><td data-label="활동일" class="num">5</td><td data-label="회차" class="num">285</td><td data-label="릴리즈" class="num">145</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$56.63</td><td data-label="시간" class="num">2시간 17분</td></tr></tbody></table></div>

## 프로젝트별 현황

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">프로젝트 이름을 누르면 원장 전체와 회차 이력을 볼 수 있습니다.</caption><thead><tr><th class="primary">프로젝트 · 건강</th><th>자율화</th><th>마지막 회차</th><th>결과</th><th>최근 릴리즈</th><th class="num">누적 비용</th></tr></thead><tbody><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a> <span class="pill pill-merged" title="14일: 릴리즈 9, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 22:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/AgentHub/releases/tag/v0.234.0">v0.234.0</a> <span class="meta">자산 0</span> <span class="pill pill-failed">❌ 누락</span></td><td data-label="누적 비용" class="num">$10.75</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ai-admin/">ai-admin</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 18:20</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ai-admin/releases/tag/v1.2.9">v1.2.9</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 18:40</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 18:30</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 18:50</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 19:00</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/appstore/">appstore</a> <span class="pill pill-released" title="14일: 릴리즈 5, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 19:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/appstore/releases/tag/v2.5.1">v2.5.1</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Clustara/">Clustara</a> <span class="pill pill-released" title="14일: 릴리즈 10, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-06 00:17</td><td data-label="결과"><span class="pill pill-other">• 기타</span> review held, PR open <a href="https://github.com/hkjang/clustara/pull/11">PR #11</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/clustara/releases/tag/v0.9.271">v0.9.271</a> <span class="meta">자산 3</span></td><td data-label="누적 비용" class="num">$19.30</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/dataworks/">dataworks</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 19:20</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/dataworks/releases/tag/v0.9.42">v0.9.42</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/git-ctx/">git-ctx</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 19:30</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/git-ctx/releases/tag/v0.77.5">v0.77.5</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a> <span class="pill pill-failed" title="14일: 릴리즈 5, 실패 0, 경고 4, 회귀 0">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 19:40</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/igame/releases/tag/v0.7.6">v0.7.6</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a> <span class="pill pill-merged" title="14일: 릴리즈 7, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 17:27</td><td data-label="결과"><span class="pill pill-other">• 기타</span> verify failed: 실패한 검증: cargo test --quiet (exit 127)</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/invenqor/releases/tag/v0.2.25">v0.2.25</a></td><td data-label="누적 비용" class="num">$7.84</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a> <span class="pill pill-released" title="14일: 릴리즈 1, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 19:50</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/jikim/releases/tag/v0.2.2">v0.2.2</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 20:00</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/kanpic/">kanpic</a> <span class="pill pill-merged" title="14일: 릴리즈 8, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 20:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/kanpic/releases/tag/v0.236.0">v0.236.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 20:20</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/moina/releases/tag/v0.1.21">v0.1.21</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/moyro/">moyro</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 20:30</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/moyro/releases/tag/v0.2.15">v0.2.15</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/muni/">muni</a> <span class="pill pill-released" title="14일: 릴리즈 6, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 05:43</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/muni/pull/6">PR #6</a>, released <a href="https://github.com/hkjang/muni/releases/tag/v0.27.0">v0.27.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/muni/releases/tag/v0.27.0">v0.27.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/pii-masker/">pii-masker</a> <span class="pill pill-released" title="14일: 릴리즈 8, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 20:40</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/pii-masker/releases/tag/v1.0.12">v1.0.12</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$5.65</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ptium/">ptium</a> <span class="pill pill-merged" title="14일: 릴리즈 7, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 21:00</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ptium/releases/tag/v1.69.27">v1.69.27</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Quantoss/">Quantoss</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 17:43</td><td data-label="결과"><span class="pill pill-other">• 기타</span> CI no-ci, PR open <a href="https://github.com/hkjang/Quantoss/pull/12">PR #12</a></td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$7.80</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/releasedock/">releasedock</a> <span class="pill pill-merged" title="14일: 릴리즈 7, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 21:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/releasedock/releases/tag/v0.5.8">v0.5.8</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> <span class="pill pill-merged" title="14일: 릴리즈 10, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 21:20</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/relio/releases/tag/v1.11.17">v1.11.17</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ReSSO/">ReSSO</a> <span class="pill pill-merged" title="14일: 릴리즈 5, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 18:02</td><td data-label="결과"><span class="pill pill-other">• 기타</span> verify failed: secrets in diff</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ReSSO/releases/tag/v0.9.70">v0.9.70</a></td><td data-label="누적 비용" class="num">$5.29</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/umm/">umm</a> <span class="pill pill-released" title="14일: 릴리즈 7, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 21:30</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/umm/releases/tag/v0.71.2">v0.71.2</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Vendra/">Vendra</a> <span class="pill pill-released" title="14일: 릴리즈 6, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 18:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.41">v0.7.41</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-coders/">vibe-coders</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 21:40</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/vibe-coders/releases/tag/v0.83.0">v0.83.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/visitflow/">visitflow</a> <span class="pill pill-merged" title="14일: 릴리즈 5, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 21:50</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/visitflow/releases/tag/v2.6.5">v2.6.5</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$0.00</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a> <span class="pill pill-merged" title="14일: 릴리즈 6, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-05 22:00</td><td data-label="결과"><span class="pill pill-other">• 기타</span> hold: budget</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/weekly/releases/tag/v0.289.0">v0.289.0</a></td><td data-label="누적 비용" class="num">$0.00</td></tr></tbody></table></div>

## 오늘 상한

<dl class="kv">
<dt>비용</dt><dd><span class="meta">6.62 / 80</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:8%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>회차</dt><dd><span class="meta">1 / 60</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:1%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>릴리즈</dt><dd><span class="meta">0 / 30</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:0%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>휴면 규칙</dt><dd>변경 없음 3회 연속이면 7일 제외</dd>
<dt>수동 실행</dt><dd><a href="https://github.com/hkjang/aidev/issues/new?labels=run&title=run%3A+%3C%ED%94%84%EB%A1%9C%EC%A0%9D%ED%8A%B8%3E">이슈 만들기</a> — 제목 <code>run: &lt;프로젝트&gt;</code>, 라벨 <code>run</code> → 다음 회차 우선 실행</dd>
</dl>

설정: `state/caps.env`. 상한에 닿으면 그날은 새 회차를 시작하지 않고 알린다.

## 품질 지표 (최근 14일)

<ul class="stats"><li title="관찰 24h 지난 머지 175건 중 회귀 없음 175건"><b>100%</b><span>검증된 개선 완료율</span></li><li title="릴리즈 시도 145건 중 자산 검증까지 133건"><b>92%</b><span>완전한 릴리즈 비율</span></li><li title="되돌림·롤백 0건 / 관찰 머지 175건"><b>0%</b><span>사람의 재작업률</span></li><li title="회귀 0건 / 관찰 머지 175건"><b>0%</b><span>변경 후 회귀율</span></li><li title="비용 확인된 유효 개선 0건, 미확인 세션 1"><b>—</b><span>유효 개선당 비용</span></li><li title="해결된 경고 0건"><b>—</b><span>예외 처리 소요 시간(중앙값)</span></li><li title="최근 14일"><b>26</b><span>실행 오류</span></li></ul>

'검증된 개선'은 머지 후 24시간 관찰에서 회귀(main CI 실패·되돌림·롤백)가 없는 변경. '완전한 릴리즈'는 태그·Release·필수 자산 검증까지 끝난 것. 비용이 확인되지 않은 세션은 0이 아니라 '미확인'으로 뺀다.

## 비용·사용량

<ul class="stats"><li><b>$6.62</b><span>오늘 비용</span></li><li><b>16분</b><span>오늘 에이전트 시간</span></li><li><b>2</b><span>오늘 세션</span></li><li><b>$56.63</b><span>누적 비용</span></li><li><b>2시간 17분</b><span>누적 시간</span></li><li><b>57.0M/504K</b><span>누적 토큰 입력/출력</span></li></ul>

claude -p 가 세션마다 보고한 추정값(정액제에서는 참고값). 회차별 내역은 각 일일 보고와 프로젝트 페이지, 원본은 [usage.jsonl](https://hkjang.github.io/aidev/data/usage.jsonl).

## FAQ

<details><summary>이 페이지는 무엇인가요?</summary><p>hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다.</p></details>
<details><summary>얼마나 자주 갱신되나요?</summary><p>에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다. Atom 피드(/feed.xml)를 구독하면 일일 보고를 받아볼 수 있습니다.</p></details>
<details><summary>한 회차에서 에이전트는 무엇을 하나요?</summary><p>저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열고 CI 통과를 확인한 뒤 머지하며, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로·첨부 자산)을 확인해 같은 방식으로 다음 버전을 냅니다.</p></details>
<details><summary>릴리즈 워크플로가 실패하면 어떻게 되나요?</summary><p>러너가 한 번 자동으로 재실행합니다. 그래도 실패하면 실패 단계와 로그 요지를 수정 과제 큐에 넣고, 다음 회차에서 그 프로젝트를 우선 배정해 에이전트가 원인을 고칩니다(검증을 느슨하게 만드는 것은 금지). 큐에 있는 동안 &#x27;주의 필요&#x27;에 표시됩니다.</p></details>
<details><summary>어떤 프로젝트가 대상인가요?</summary><p>최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다.</p></details>
<details><summary>&#x27;머지(릴리즈 없음)&#x27;는 무슨 뜻인가요?</summary><p>개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다.</p></details>
<details><summary>&#x27;주의 필요&#x27;에는 무엇이 뜨나요? 알림은 어디로 오나요?</summary><p>최근 2일 회차 중 CI 실패로 머지되지 않은 PR, 릴리즈 실패, 이전 릴리즈에는 있던 첨부 자산이 빠진 릴리즈, 수정 과제 대기처럼 사람이 확인해야 할 항목입니다. 새 경고가 생기면 GitHub Issue(라벨 alert)에 기록하고, 설정돼 있으면 Slack·이메일·Windows 알림으로도 보냅니다.</p></details>
<details><summary>머지 전에 검토는 없나요?</summary><p>있습니다. 구현과 다른 세션의 리뷰 에이전트가 diff 만 읽고 &#x27;머지하면 안 되는 이유&#x27;(검증하지 않는 테스트, 논리 오류, 설명과 다른 동작, 위험한 변경)를 찾습니다. 거절하면 PR 에 사유를 달고 열어 둡니다. 또 워크플로·마이그레이션·인증·결제·배포 파일(보호 파일)을 건드린 PR 은 자동 머지하지 않습니다.</p></details>
<details><summary>깨진 변경은 어떻게 되나요?</summary><p>머지 2시간 뒤부터 main CI 실패와 되돌림 커밋을 확인해 &#x27;교훈&#x27;으로 기록하고, 그 프로젝트의 다음 회차 프롬프트에 주입해 같은 실수를 피하게 합니다. 릴리즈 워크플로가 반복 실패하고 수정 회차도 실패하면 원래 머지를 되돌리는 롤백 PR 을 자동으로 엽니다(머지는 사람). 프로젝트마다 최근 14일의 실패·경고·회귀로 건강 등급 A~D 를 매깁니다.</p></details>
<details><summary>자율화 단계란 무엇인가요?</summary><p>프로젝트마다 러너 권한을 &#x27;분석만 → PR 생성 → 승인 후 병합 → 저위험 자동 병합 → 검증된 릴리즈 게시&#x27; 다섯 단계로 나눕니다. 단계를 올리는 것은 사람이 정책 파일(state/&lt;프로젝트&gt;.policy.json 의 autonomy)을 고쳐야 하고, 롤백이나 회귀가 생기면 러너가 한 단계 내립니다(⬇ 표시). 작업함에서 승인(aidev-approved)·반려(aidev-rejected) 라벨로 개별 PR 을 처리할 수 있습니다.</p></details>
<details><summary>긴급히 멈추려면?</summary><p>bin/stop.sh all|merge|release|&lt;프로젝트&gt; on &quot;사유&quot; 또는 aidev 저장소에 라벨 stop, 제목 stop: &lt;범위&gt; 이슈를 만들면 됩니다. 새 회차 시작뿐 아니라 진행 중인 회차도 에이전트 시작 전·머지 전·릴리즈 전 경계에서 멈춥니다.</p></details>
<details><summary>비용은 어떻게 계산되나요?</summary><p>각 회차의 claude -p 세션이 보고한 추정 비용(USD)과 소요 시간·턴 수·토큰을 그대로 합산합니다. 정액제 구독에서는 실제 청구가 아닌 참고값입니다.</p></details>
<details><summary>원본 데이터는 어디서 보나요?</summary><p>회차 기록은 runs.jsonl, 사용량은 usage.jsonl, 요약은 data/summary.json, 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다.</p></details>

---
러너·프롬프트·원장은 [https://github.com/hkjang/aidev](https://github.com/hkjang/aidev) 에서 관리한다. 이 페이지는 회차가 끝날 때마다 `bin/report.py` 가 다시 만든다.

{% endraw %}
