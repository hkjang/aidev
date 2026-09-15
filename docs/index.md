---
title: "aidev 자율 개선 대시보드"
description: "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 16회차·릴리즈 1건, 누적 798회차·릴리즈 315건, 주의 필요 9건."
last_modified_at: 2026-09-16 08:32:11 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "WebSite",
 "name": "aidev 자율 개선 대시보드",
 "url": "https://hkjang.github.io/aidev/",
 "description": "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 16회차·릴리즈 1건, 누적 798회차·릴리즈 315건, 주의 필요 9건.",
 "inLanguage": "ko",
 "author": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-16T08:32:11+09:00"
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
   "name": "일일 보고 2026-09-16",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-16/"
  },
  {
   "@type": "ListItem",
   "position": 2,
   "name": "일일 보고 2026-09-15",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-15/"
  },
  {
   "@type": "ListItem",
   "position": 3,
   "name": "일일 보고 2026-09-14",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-14/"
  },
  {
   "@type": "ListItem",
   "position": 4,
   "name": "일일 보고 2026-09-13",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-13/"
  },
  {
   "@type": "ListItem",
   "position": 5,
   "name": "일일 보고 2026-09-12",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-12/"
  },
  {
   "@type": "ListItem",
   "position": 6,
   "name": "일일 보고 2026-09-11",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-11/"
  },
  {
   "@type": "ListItem",
   "position": 7,
   "name": "일일 보고 2026-09-10",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-10/"
  },
  {
   "@type": "ListItem",
   "position": 8,
   "name": "일일 보고 2026-09-09",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-09/"
  },
  {
   "@type": "ListItem",
   "position": 9,
   "name": "일일 보고 2026-09-08",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-08/"
  },
  {
   "@type": "ListItem",
   "position": 10,
   "name": "일일 보고 2026-09-07",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-07/"
  },
  {
   "@type": "ListItem",
   "position": 11,
   "name": "일일 보고 2026-09-06",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-06/"
  },
  {
   "@type": "ListItem",
   "position": 12,
   "name": "일일 보고 2026-09-05",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-05/"
  },
  {
   "@type": "ListItem",
   "position": 13,
   "name": "일일 보고 2026-09-04",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-04/"
  },
  {
   "@type": "ListItem",
   "position": 14,
   "name": "일일 보고 2026-09-03",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-03/"
  },
  {
   "@type": "ListItem",
   "position": 15,
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

<p class="tldr"><strong>한 줄 요약.</strong> Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 16회차·릴리즈 1건, 누적 798회차·릴리즈 315건, 주의 필요 9건. 회차가 끝날 때마다 자동 갱신됩니다 (마지막 갱신 <time datetime="2026-09-16T08:32:11+09:00" data-rel>2026-09-16 08:32</time> KST).</p>

<div class="alerts" role="alert"><strong>⚠️ 주의 필요 9건</strong> <span class="meta">— 새 경고는 GitHub Issue·Slack·이메일·Windows 알림으로도 보냅니다</span><ul><li><a href="https://hkjang.github.io/aidev/projects/orbit/">orbit</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 01:19)</span></li><li><a href="https://hkjang.github.io/aidev/projects/orbit/">orbit</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 01:44)</span></li><li><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 02:10)</span></li><li><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 03:49)</span></li><li><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 06:12)</span></li><li><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 07:40)</span></li><li><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 08:07)</span></li><li><a href="https://hkjang.github.io/aidev/projects/moyro/">moyro</a> — 보호 파일 변경 — 자동 머지 안 함, 사람 검토 필요 <span class=meta>(2026-09-16 08:32)</span></li><li><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a> — 최신 릴리즈 v0.2.35 자산 0개 (이전 v0.2.34: 25개)</li></ul></div>

<div class="alerts ok" role="status"><strong>🩺 러너 정상</strong> <span class="meta">— 마지막 회차 7분 전 · 스케줄러 실행 중 · 다음 실행 2026-09-16 오전 8:20:00 · 디스크 71% · 최근 7일 회귀 12건 · 점검 08:15</span></div>

<div class="alerts" role="alert"><strong>⛔ 긴급 중지 중:</strong> <code>aiportal-java</code> — <a href="https://hkjang.github.io/aidev/inbox/">작업함</a></div>

<p><a href="https://hkjang.github.io/aidev/inbox/"><strong>📥 작업함</strong></a> — 사람 판단이 필요한 PR·복구·수정 과제 26건</p>

[운영 문서](https://github.com/hkjang/aidev#readme) · [원장](https://github.com/hkjang/aidev/tree/main/state) · [실행 이력](https://github.com/hkjang/aidev/commits/main) · [경고 이슈](https://github.com/hkjang/aidev/issues?q=label%3Aalert) · [교훈 25건](https://hkjang.github.io/aidev/lessons/) · [Atom 피드](https://hkjang.github.io/aidev/feed.xml) · [summary.json](https://hkjang.github.io/aidev/data/summary.json)

## 오늘 (2026-09-16)

<ul class="stats"><li><b>16</b><span>회차</span></li><li><b>11</b><span>프로젝트</span></li><li><b>1</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>11</b><span>검토 대기</span></li><li><b>1</b><span>검증 실패</span></li><li><b>1</b><span>변경 없음</span></li><li><b>2</b><span>실행 오류</span></li><li><b>$111.26</b><span>비용</span></li><li><b>3시간 46분</b><span>에이전트 시간</span></li></ul>

[2026-09-16 보고 자세히 보기 →](https://hkjang.github.io/aidev/reports/2026-09-16/)

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">오늘 전체 회차 (KST)</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="released"><td data-label="시각">00:52</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/ReSSO/">ReSSO</a></td><td data-label="결과"><span class="pill pill-released" title="outcome=release-ready">배포 준비 완료</span> merged <a href="https://github.com/hkjang/ReSSO/pull/21">PR #21</a>, released <a href="https://github.com/hkjang/ReSSO/releases/tag/v0.9.84">v0.9.84</a><div class="meta">7파일 <span style="color:var(--good)">+201</span>/<span style="color:var(--bad)">−3</span> · 테스트 1 — feat: count silent authentications by their answer, so a relying party&#x27;s prompt=none loop shows up somewhere</div></td></tr><tr data-status="other"><td data-label="시각">01:19</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/orbit/">orbit</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/orbit/pull/2">PR #2</a><div class="meta">15파일 <span style="color:var(--good)">+1913</span>/<span style="color:var(--bad)">−11</span> · 테스트 3 — feat(admin): 관리자가 화면에서 방문 추적 스크립트를 붙일 수 있게 한다</div></td></tr><tr data-status="other"><td data-label="시각">01:44</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/orbit/">orbit</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/orbit/pull/3">PR #3</a><div class="meta">17파일 <span style="color:var(--good)">+1148</span>/<span style="color:var(--bad)">−3</span> · 테스트 2 — feat(handoff): 승인된 기억을 다른 사내 서비스로 넘긴다</div></td></tr><tr data-status="other"><td data-label="시각">02:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/igame/pull/18">PR #18</a><div class="meta">27파일 <span style="color:var(--good)">+2105</span>/<span style="color:var(--bad)">−16</span> · 테스트 4 — feat: send approval and ranking notifications through the company SMTP relay</div></td></tr><tr data-status="nochange"><td data-label="시각">02:24</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/cutover/">cutover</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> no change</td></tr><tr data-status="other"><td data-label="시각">02:51</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/sqlon/">sqlon</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> review held, PR open <a href="https://github.com/hkjang/sqlon/pull/3">PR #3</a><div class="meta">19파일 <span style="color:var(--good)">+1830</span>/<span style="color:var(--bad)">−32</span> · 테스트 2 — feat(tracking): 관리자가 화면에서 붙이는 방문 추적 스크립트와 nonce 기반 CSP 를 추가합니다</div></td></tr><tr data-status="other"><td data-label="시각">03:19</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/sqlon/">sqlon</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> CI no-ci, PR open <a href="https://github.com/hkjang/sqlon/pull/4">PR #4</a><div class="meta">10파일 <span style="color:var(--good)">+1530</span>/<span style="color:var(--bad)">−754</span> · 테스트 1 — feat(handoff): DBA 다이제스트를 다른 서비스로 넘기는 단일 사용 표(claim) 발급 — HANDOFF-STANDARD 보내는 쪽</div></td></tr><tr data-status="other"><td data-label="시각">03:49</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/jikim/pull/28">PR #28</a><div class="meta">22파일 <span style="color:var(--good)">+2363</span>/<span style="color:var(--bad)">−6</span> · 테스트 5 — feat: send approval and rotation-failure mail through a company SMTP relay</div></td></tr><tr data-status="other"><td data-label="시각">04:49</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> error: agent produced no result (TIMEOUT )<div class="meta">15파일 <span style="color:var(--good)">+1442</span>/<span style="color:var(--bad)">−8</span> · 테스트 1 — feat: 관리자가 화면에서 방문 추적 스크립트를 붙입니다</div></td></tr><tr data-status="other"><td data-label="시각">05:46</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> error: agent produced no result (TIMEOUT )</td></tr><tr data-status="other"><td data-label="시각">06:12</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/jupiq/pull/16">PR #16</a><div class="meta">24파일 <span style="color:var(--good)">+2143</span>/<span style="color:var(--bad)">−25</span> · 테스트 5 — feat: 사내 SMTP 릴레이로 승인·Hub 상태·키 만료 알림을 보낸다</div></td></tr><tr data-status="other"><td data-label="시각">06:53</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실패한 검증: go test ./... (exit 1)<div class="meta">15파일 <span style="color:var(--good)">+1464</span>/<span style="color:var(--bad)">−8</span> · 테스트 1 — fix: Momento 수집기의 경로를 지키고, 꺼진 동안의 신고는 적지 않습니다</div></td></tr><tr data-status="other"><td data-label="시각">07:15</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/postra/">postra</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> review held, PR open <a href="https://github.com/hkjang/postra/pull/11">PR #11</a><div class="meta">37파일 <span style="color:var(--good)">+1499</span>/<span style="color:var(--bad)">−21</span> · 테스트 5 — feat(handoff): send a mail to another in-house service as markdown</div></td></tr><tr data-status="other"><td data-label="시각">07:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/moina/pull/23">PR #23</a><div class="meta">17파일 <span style="color:var(--good)">+899</span>/<span style="color:var(--bad)">−75</span> · 테스트 4 — feat: 사내 SMTP 릴레이 알림에 발송 기록과 이벤트 스위치를 더합니다</div></td></tr><tr data-status="other"><td data-label="시각">08:07</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/weekly/pull/15">PR #15</a><div class="meta">19파일 <span style="color:var(--good)">+876</span>/<span style="color:var(--bad)">−82</span> · 테스트 3 — feat: 허용된 사내 서비스에서 표(claim)로 PPTX 를 넘겨받습니다</div></td></tr><tr data-status="other"><td data-label="시각">08:32</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/moyro/">moyro</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/moyro/pull/15">PR #15</a><div class="meta">28파일 <span style="color:var(--good)">+2307</span>/<span style="color:var(--bad)">−44</span> · 테스트 5 — feat: notify people who are waiting by mail through the company SMTP relay</div></td></tr></tbody></table></div>

## 최근 14일

<div class="chart"><svg viewBox="0 0 640 200" role="img" aria-labelledby="chart-t chart-d"><title id="chart-t">최근 14일 회차 수</title><desc id="chart-d">날짜별 회차 수를 릴리즈·머지·변경 없음·실패로 나눠 쌓은 막대. 같은 값은 아래 표에 있다.</desc><line class="grid" x1="28" x2="636" y1="172.0" y2="172.0"/><text x="22" y="176.0" text-anchor="end">0</text><line class="grid" x1="28" x2="636" y1="155.8" y2="155.8"/><text x="22" y="159.8" text-anchor="end">10</text><line class="grid" x1="28" x2="636" y1="139.6" y2="139.6"/><text x="22" y="143.6" text-anchor="end">20</text><line class="grid" x1="28" x2="636" y1="123.4" y2="123.4"/><text x="22" y="127.4" text-anchor="end">30</text><line class="grid" x1="28" x2="636" y1="107.2" y2="107.2"/><text x="22" y="111.2" text-anchor="end">40</text><line class="grid" x1="28" x2="636" y1="91.0" y2="91.0"/><text x="22" y="95.0" text-anchor="end">50</text><line class="grid" x1="28" x2="636" y1="74.8" y2="74.8"/><text x="22" y="78.8" text-anchor="end">60</text><line class="grid" x1="28" x2="636" y1="58.6" y2="58.6"/><text x="22" y="62.6" text-anchor="end">70</text><line class="grid" x1="28" x2="636" y1="42.4" y2="42.4"/><text x="22" y="46.4" text-anchor="end">80</text><line class="grid" x1="28" x2="636" y1="26.2" y2="26.2"/><text x="22" y="30.2" text-anchor="end">90</text><line class="grid" x1="28" x2="636" y1="10.0" y2="10.0"/><text x="22" y="14.0" text-anchor="end">100</text><g><title>2026-09-03: 회차 96 · 릴리즈 63 · 머지 25 · 변경 없음 8 · 실패 0</title><rect x="37.6" y="69.9" width="24.0" height="100.1" fill="#0ca30c"/><rect x="37.6" y="29.4" width="24.0" height="38.5" fill="#fab219"/><path d="M37.6,29.4 V20.5 Q37.6,16.5 41.6,16.5 H57.6 Q61.6,16.5 61.6,20.5 V29.4 Z" fill="#9ca3af"/></g><g><title>2026-09-04: 회차 77 · 릴리즈 33 · 머지 9 · 변경 없음 1 · 실패 0</title><rect x="80.7" y="118.5" width="24.0" height="51.5" fill="#0ca30c"/><rect x="80.7" y="104.0" width="24.0" height="12.6" fill="#fab219"/><path d="M80.7,104.0 V103.1 Q80.7,102.3 81.5,102.3 H103.9 Q104.7,102.3 104.7,103.1 V104.0 Z" fill="#9ca3af"/><text x="92.7" y="186" text-anchor="middle">09/04</text></g><g><title>2026-09-05: 회차 36 · 릴리즈 21 · 머지 6 · 변경 없음 1 · 실패 0</title><rect x="123.9" y="138.0" width="24.0" height="32.0" fill="#0ca30c"/><rect x="123.9" y="128.3" width="24.0" height="7.7" fill="#fab219"/><path d="M123.9,128.3 V127.4 Q123.9,126.6 124.7,126.6 H147.0 Q147.9,126.6 147.9,127.4 V128.3 Z" fill="#9ca3af"/></g><g><title>2026-09-06: 회차 47 · 릴리즈 12 · 머지 3 · 변경 없음 1 · 실패 0</title><rect x="167.0" y="152.6" width="24.0" height="17.4" fill="#0ca30c"/><rect x="167.0" y="147.7" width="24.0" height="2.9" fill="#fab219"/><path d="M167.0,147.7 V146.9 Q167.0,146.1 167.8,146.1 H190.2 Q191.0,146.1 191.0,146.9 V147.7 Z" fill="#9ca3af"/><text x="179.0" y="186" text-anchor="middle">09/06</text></g><g><title>2026-09-07: 회차 65 · 릴리즈 11 · 머지 13 · 변경 없음 0 · 실패 0</title><rect x="210.1" y="154.2" width="24.0" height="15.8" fill="#0ca30c"/><path d="M210.1,154.2 V137.1 Q210.1,133.1 214.1,133.1 H230.1 Q234.1,133.1 234.1,137.1 V154.2 Z" fill="#fab219"/></g><g><title>2026-09-08: 회차 53 · 릴리즈 29 · 머지 8 · 변경 없음 0 · 실패 0</title><rect x="253.3" y="125.0" width="24.0" height="45.0" fill="#0ca30c"/><path d="M253.3,125.0 V116.1 Q253.3,112.1 257.3,112.1 H273.3 Q277.3,112.1 277.3,116.1 V125.0 Z" fill="#fab219"/><text x="265.3" y="186" text-anchor="middle">09/08</text></g><g><title>2026-09-09: 회차 78 · 릴리즈 37 · 머지 11 · 변경 없음 0 · 실패 2</title><rect x="296.4" y="112.1" width="24.0" height="57.9" fill="#0ca30c"/><rect x="296.4" y="94.2" width="24.0" height="15.8" fill="#fab219"/><path d="M296.4,94.2 V92.6 Q296.4,91.0 298.0,91.0 H318.8 Q320.4,91.0 320.4,92.6 V94.2 Z" fill="#d03b3b"/></g><g><title>2026-09-10: 회차 62 · 릴리즈 33 · 머지 7 · 변경 없음 0 · 실패 1</title><rect x="339.6" y="118.5" width="24.0" height="51.5" fill="#0ca30c"/><rect x="339.6" y="107.2" width="24.0" height="9.3" fill="#fab219"/><path d="M339.6,107.2 V106.4 Q339.6,105.6 340.4,105.6 H362.8 Q363.6,105.6 363.6,106.4 V107.2 Z" fill="#d03b3b"/><text x="351.6" y="186" text-anchor="middle">09/10</text></g><g><title>2026-09-11: 회차 56 · 릴리즈 5 · 머지 1 · 변경 없음 0 · 실패 4</title><rect x="382.7" y="163.9" width="24.0" height="6.1" fill="#0ca30c"/><rect x="382.7" y="162.3" width="24.0" height="0.0" fill="#fab219"/><path d="M382.7,162.3 V159.0 Q382.7,155.8 386.0,155.8 H403.5 Q406.7,155.8 406.7,159.0 V162.3 Z" fill="#d03b3b"/></g><g><title>2026-09-12: 회차 62 · 릴리즈 8 · 머지 1 · 변경 없음 0 · 실패 6</title><rect x="425.9" y="159.0" width="24.0" height="11.0" fill="#0ca30c"/><rect x="425.9" y="157.4" width="24.0" height="0.0" fill="#fab219"/><path d="M425.9,157.4 V151.7 Q425.9,147.7 429.9,147.7 H445.9 Q449.9,147.7 449.9,151.7 V157.4 Z" fill="#d03b3b"/><text x="437.9" y="186" text-anchor="middle">09/12</text></g><g><title>2026-09-13: 회차 42 · 릴리즈 7 · 머지 3 · 변경 없음 0 · 실패 1</title><rect x="469.0" y="160.7" width="24.0" height="9.3" fill="#0ca30c"/><rect x="469.0" y="155.8" width="24.0" height="2.9" fill="#fab219"/><path d="M469.0,155.8 V155.0 Q469.0,154.2 469.8,154.2 H492.2 Q493.0,154.2 493.0,155.0 V155.8 Z" fill="#d03b3b"/></g><g><title>2026-09-14: 회차 55 · 릴리즈 27 · 머지 0 · 변경 없음 0 · 실패 1</title><rect x="512.1" y="128.3" width="24.0" height="41.7" fill="#0ca30c"/><path d="M512.1,128.3 V127.4 Q512.1,126.6 513.0,126.6 H535.3 Q536.1,126.6 536.1,127.4 V128.3 Z" fill="#d03b3b"/><text x="524.1" y="186" text-anchor="middle">09/14</text></g><g><title>2026-09-15: 회차 3 · 릴리즈 0 · 머지 2 · 변경 없음 0 · 실패 0</title><path d="M555.3,172.0 V170.4 Q555.3,168.8 556.9,168.8 H577.7 Q579.3,168.8 579.3,170.4 V172.0 Z" fill="#fab219"/></g><g><title>2026-09-16: 회차 16 · 릴리즈 1 · 머지 0 · 변경 없음 1 · 실패 0</title><rect x="598.4" y="170.4" width="24.0" height="0.0" fill="#0ca30c"/><path d="M598.4,170.4 V169.6 Q598.4,168.8 599.2,168.8 H621.6 Q622.4,168.8 622.4,169.6 V170.4 Z" fill="#9ca3af"/><text x="610.4" y="164.8" text-anchor="middle">16</text><text x="610.4" y="186" text-anchor="middle">09/16</text></g><line class="grid" x1="28" x2="636" y1="172" y2="172"/></svg><ul class="legend" aria-hidden="true"><li><i style="background:#0ca30c"></i>🚀 릴리즈</li><li><i style="background:#fab219"></i>✅ 머지</li><li><i style="background:#9ca3af"></i>➖ 변경 없음</li><li><i style="background:#d03b3b"></i>❌ 실패</li></ul></div>

## 일일 보고

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">날짜</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">머지</th><th class="num">변경 없음</th><th class="num">실패</th><th class="num">비용</th></tr></thead><tbody><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-16/">2026-09-16</a></td><td data-label="회차" class="num">16</td><td data-label="릴리즈" class="num">1</td><td data-label="머지" class="num">0</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$111.26</td></tr><tr data-status="merged"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-15/">2026-09-15</a></td><td data-label="회차" class="num">3</td><td data-label="릴리즈" class="num">0</td><td data-label="머지" class="num">2</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$35.17</td></tr><tr data-status="failed"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-14/">2026-09-14</a></td><td data-label="회차" class="num">55</td><td data-label="릴리즈" class="num">27</td><td data-label="머지" class="num">0</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">1</td><td data-label="비용" class="num">$294.07</td></tr><tr data-status="failed"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-13/">2026-09-13</a></td><td data-label="회차" class="num">42</td><td data-label="릴리즈" class="num">7</td><td data-label="머지" class="num">3</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">1</td><td data-label="비용" class="num">$348.23</td></tr><tr data-status="failed"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-12/">2026-09-12</a></td><td data-label="회차" class="num">62</td><td data-label="릴리즈" class="num">8</td><td data-label="머지" class="num">1</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">6</td><td data-label="비용" class="num">$286.03</td></tr><tr data-status="failed"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-11/">2026-09-11</a></td><td data-label="회차" class="num">56</td><td data-label="릴리즈" class="num">5</td><td data-label="머지" class="num">1</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">4</td><td data-label="비용" class="num">$364.29</td></tr><tr data-status="failed"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-10/">2026-09-10</a></td><td data-label="회차" class="num">62</td><td data-label="릴리즈" class="num">33</td><td data-label="머지" class="num">7</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">1</td><td data-label="비용" class="num">$356.91</td></tr><tr data-status="failed"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-09/">2026-09-09</a></td><td data-label="회차" class="num">78</td><td data-label="릴리즈" class="num">37</td><td data-label="머지" class="num">11</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">2</td><td data-label="비용" class="num">$300.23</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-08/">2026-09-08</a></td><td data-label="회차" class="num">53</td><td data-label="릴리즈" class="num">29</td><td data-label="머지" class="num">8</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$227.63</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-07/">2026-09-07</a></td><td data-label="회차" class="num">65</td><td data-label="릴리즈" class="num">11</td><td data-label="머지" class="num">13</td><td data-label="변경 없음" class="num">0</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$135.58</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-06/">2026-09-06</a></td><td data-label="회차" class="num">47</td><td data-label="릴리즈" class="num">12</td><td data-label="머지" class="num">3</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$190.48</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-05/">2026-09-05</a></td><td data-label="회차" class="num">36</td><td data-label="릴리즈" class="num">21</td><td data-label="머지" class="num">6</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$50.01</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-04/">2026-09-04</a></td><td data-label="회차" class="num">77</td><td data-label="릴리즈" class="num">33</td><td data-label="머지" class="num">9</td><td data-label="변경 없음" class="num">1</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr><tr data-status="released"><td data-label="날짜" class="primary"><a href="https://hkjang.github.io/aidev/reports/2026-09-03/">2026-09-03</a></td><td data-label="회차" class="num">96</td><td data-label="릴리즈" class="num">63</td><td data-label="머지" class="num">25</td><td data-label="변경 없음" class="num">8</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">—</td></tr></tbody></table></div>

## 주간·월간 보고

### 주간

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">주</th><th class="num">활동일</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">실패</th><th class="num">비용</th><th class="num">시간</th></tr></thead><tbody><tr data-status="released"><td data-label="주" class="primary"><a href="https://hkjang.github.io/aidev/weekly/2026-W38/">2026-W38</a></td><td data-label="활동일" class="num">3</td><td data-label="회차" class="num">74</td><td data-label="릴리즈" class="num">28</td><td data-label="실패" class="num">1</td><td data-label="비용" class="num">$440.49</td><td data-label="시간" class="num">16시간 48분</td></tr><tr data-status="released"><td data-label="주" class="primary"><a href="https://hkjang.github.io/aidev/weekly/2026-W37/">2026-W37</a></td><td data-label="활동일" class="num">7</td><td data-label="회차" class="num">418</td><td data-label="릴리즈" class="num">130</td><td data-label="실패" class="num">14</td><td data-label="비용" class="num">$2018.91</td><td data-label="시간" class="num">83시간 14분</td></tr><tr data-status="released"><td data-label="주" class="primary"><a href="https://hkjang.github.io/aidev/weekly/2026-W36/">2026-W36</a></td><td data-label="활동일" class="num">5</td><td data-label="회차" class="num">306</td><td data-label="릴리즈" class="num">157</td><td data-label="실패" class="num">0</td><td data-label="비용" class="num">$240.48</td><td data-label="시간" class="num">10시간 51분</td></tr></tbody></table></div>

### 월간

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">월</th><th class="num">활동일</th><th class="num">회차</th><th class="num">릴리즈</th><th class="num">실패</th><th class="num">비용</th><th class="num">시간</th></tr></thead><tbody><tr data-status="released"><td data-label="월" class="primary"><a href="https://hkjang.github.io/aidev/monthly/2026-09/">2026-09</a></td><td data-label="활동일" class="num">15</td><td data-label="회차" class="num">798</td><td data-label="릴리즈" class="num">315</td><td data-label="실패" class="num">15</td><td data-label="비용" class="num">$2699.89</td><td data-label="시간" class="num">110시간 53분</td></tr></tbody></table></div>

## 프로젝트별 현황

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">프로젝트 이름을 누르면 원장 전체와 회차 이력을 볼 수 있습니다.</caption><thead><tr><th class="primary">프로젝트 · 건강</th><th>자율화</th><th>마지막 회차</th><th>결과</th><th>최근 릴리즈</th><th class="num">누적 비용</th></tr></thead><tbody><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/AgentHub/">AgentHub</a> <span class="pill pill-failed" title="14일: 릴리즈 19, 실패 0, 경고 7, 회귀 4">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 15:35</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/AgentHub/releases/tag/v0.246.0">v0.246.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/AgentHub/releases/tag/v0.246.0">v0.246.0</a> <span class="meta">자산 8</span></td><td data-label="누적 비용" class="num">$166.19</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ai-admin/">ai-admin</a> <span class="pill pill-failed" title="14일: 릴리즈 12, 실패 0, 경고 9, 회귀 0">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 11:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/ai-admin/pull/23">PR #23</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ai-admin/releases/tag/v1.2.20">v1.2.20</a> <span class="meta">자산 2</span></td><td data-label="누적 비용" class="num">$105.81</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front/">aiportal-front</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 1">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-10 19:15</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front/pull/14">PR #14</a>, release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$37.95</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-13 07:44</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/15">PR #15</a> (approved), release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$35.04</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-java/">aiportal-java</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-08 19:17</td><td data-label="결과"><span class="pill pill-other">• 기타</span> verify failed: 실패한 검증: ./gradlew --quiet test (exit 1)</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$19.39</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-py/">aiportal-py</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 1">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-10 19:13</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-py/pull/17">PR #17</a>, release skipped</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$41.02</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/appstore/">appstore</a> <span class="pill pill-failed" title="14일: 릴리즈 14, 실패 0, 경고 2, 회귀 2">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 20:53</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/appstore/pull/18">PR #18</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/appstore/releases/tag/v2.7.0">v2.7.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$122.23</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Clustara/">Clustara</a> <span class="pill pill-merged" title="14일: 릴리즈 19, 실패 0, 경고 2, 회귀 1">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-10 17:50</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/clustara/pull/20">PR #20</a>, released <a href="https://github.com/hkjang/clustara/releases/tag/v0.9.281">v0.9.281</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/clustara/releases/tag/v0.9.281">v0.9.281</a> <span class="meta">자산 3</span></td><td data-label="누적 비용" class="num">$75.95</td></tr><tr data-status="nochange"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/cutover/">cutover</a> <span class="pill pill-released" title="14일: 릴리즈 2, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 02:24</td><td data-label="결과"><span class="pill pill-nochange">➖ 변경 없음</span> no change</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/cutover/releases/tag/v1.6.0">v1.6.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$31.74</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/DartFly/">DartFly</a> <span class="pill pill-merged" title="14일: 릴리즈 1, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 16:56</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/DartFly/releases/tag/v2.68.0">v2.68.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/DartFly/releases/tag/v2.68.0">v2.68.0</a> <span class="meta">자산 2</span></td><td data-label="누적 비용" class="num">$26.84</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/dataworks/">dataworks</a> <span class="pill pill-merged" title="14일: 릴리즈 17, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 00:07</td><td data-label="결과"><span class="pill pill-other">• 기타</span> review held, PR open <a href="https://github.com/hkjang/dataworks/pull/21">PR #21</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/dataworks/releases/tag/v0.9.54">v0.9.54</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$101.81</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/git-ctx/">git-ctx</a> <span class="pill pill-failed" title="14일: 릴리즈 9, 실패 0, 경고 6, 회귀 2">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-10 19:34</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/git-ctx/pull/28">PR #28</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/git-ctx/releases/tag/v0.77.13">v0.77.13</a> <span class="meta">자산 2</span></td><td data-label="누적 비용" class="num">$41.21</td></tr><tr data-status="merged"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/hunter/">hunter</a> <span class="pill pill-merged" title="14일: 릴리즈 1, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-15 23:32</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/hunter/pull/3">PR #3</a>, release tag held (timeout)</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/hunter/releases/tag/v1.11.0">v1.11.0</a></td><td data-label="누적 비용" class="num">$36.21</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/igame/">igame</a> <span class="pill pill-failed" title="14일: 릴리즈 12, 실패 0, 경고 9, 회귀 1">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 02:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/igame/pull/18">PR #18</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/igame/releases/tag/v0.7.14">v0.7.14</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$97.84</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Invenqor/">Invenqor</a> <span class="pill pill-failed" title="14일: 릴리즈 16, 실패 1, 경고 7, 회귀 0">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 16:08</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/invenqor/releases/tag/v0.2.35">v0.2.35</a>, asset manifest failed, <strong>ASSETS MISSING</strong></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/invenqor/releases/tag/v0.2.35">v0.2.35</a> <span class="meta">자산 0</span> <span class="pill pill-failed">❌ 누락</span></td><td data-label="누적 비용" class="num">$120.30</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a> <span class="pill pill-merged" title="14일: 릴리즈 10, 실패 0, 경고 3, 회귀 1">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 03:49</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/jikim/pull/28">PR #28</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/jikim/releases/tag/v0.2.13">v0.2.13</a> <span class="meta">자산 2</span></td><td data-label="누적 비용" class="num">$93.84</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a> <span class="pill pill-failed" title="14일: 릴리즈 10, 실패 0, 경고 4, 회귀 0">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 06:12</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/jupiq/pull/16">PR #16</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/jupiq/releases/tag/v1.7.0">v1.7.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$84.02</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/kanpic/">kanpic</a> <span class="pill pill-merged" title="14일: 릴리즈 16, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 14:10</td><td data-label="결과"><span class="pill pill-other">• 기타</span> release-only, release tag held (failed)</td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/kanpic/releases/tag/v0.246.0">v0.246.0</a></td><td data-label="누적 비용" class="num">$59.54</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Kkiit/">Kkiit</a> <span class="pill pill-merged" title="14일: 릴리즈 1, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 17:02</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/Kkiit/releases/tag/v0.4.0">v0.4.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/Kkiit/releases/tag/v0.4.0">v0.4.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$43.27</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/madi/">madi</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 20:19</td><td data-label="결과"><span class="pill pill-other">• 기타</span> review held, PR open <a href="https://github.com/hkjang/madi/pull/3">PR #3</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/madi/releases/tag/v0.3.0">v0.3.0</a></td><td data-label="누적 비용" class="num">$22.89</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/moina/">moina</a> <span class="pill pill-failed" title="14일: 릴리즈 11, 실패 3, 경고 7, 회귀 0">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 07:40</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/moina/pull/23">PR #23</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/moina/releases/tag/v0.1.30">v0.1.30</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$93.76</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Momento/">Momento</a> <span class="pill pill-failed" title="14일: 릴리즈 1, 실패 8, 경고 9, 회귀 0">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 11:26</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/Momento/releases/tag/v0.34.40">v0.34.40</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/Momento/releases/tag/v0.34.40">v0.34.40</a> <span class="meta">자산 2</span></td><td data-label="누적 비용" class="num">$38.67</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/moyro/">moyro</a> <span class="pill pill-failed" title="14일: 릴리즈 9, 실패 0, 경고 5, 회귀 1">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 08:32</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/moyro/pull/15">PR #15</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/moyro/releases/tag/v0.2.30">v0.2.30</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$95.29</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/muni/">muni</a> <span class="pill pill-merged" title="14일: 릴리즈 12, 실패 0, 경고 3, 회귀 1">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 15:02</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/muni/releases/tag/v0.39.0">v0.39.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/muni/releases/tag/v0.39.0">v0.39.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$88.90</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/nexabuilder/">nexabuilder</a> <span class="pill pill-merged" title="14일: 릴리즈 1, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 12:52</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/nexabuilder/releases/tag/v1.16.0">v1.16.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/nexabuilder/releases/tag/v1.16.0">v1.16.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$4.14</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/orbit/">orbit</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 01:44</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/orbit/pull/3">PR #3</a></td><td data-label="최근 릴리즈"></td><td data-label="누적 비용" class="num">$18.50</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/pii-masker/">pii-masker</a> <span class="pill pill-released" title="14일: 릴리즈 16, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-10 14:39</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/pii-masker/pull/17">PR #17</a>, released <a href="https://github.com/hkjang/pii-masker/releases/tag/v1.0.20">v1.0.20</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/pii-masker/releases/tag/v1.0.20">v1.0.20</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$29.27</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/postra/">postra</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 2, 경고 2, 회귀 1">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 07:15</td><td data-label="결과"><span class="pill pill-other">• 기타</span> review held, PR open <a href="https://github.com/hkjang/postra/pull/11">PR #11</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/postra/releases/tag/v0.20.0">v0.20.0</a> <span class="meta">자산 5</span></td><td data-label="누적 비용" class="num">$40.06</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ptium/">ptium</a> <span class="pill pill-failed" title="14일: 릴리즈 12, 실패 0, 경고 3, 회귀 2">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 17:43</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/ptium/releases/tag/v1.69.39">v1.69.39</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ptium/releases/tag/v1.69.39">v1.69.39</a> <span class="meta">자산 7</span></td><td data-label="누적 비용" class="num">$96.36</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Quantoss/">Quantoss</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-07 11:43</td><td data-label="결과"><span class="pill pill-other">• 기타</span> approved <a href="https://github.com/hkjang/Quantoss/pull/13">PR #13</a>, rebase conflict</td><td data-label="최근 릴리즈">skipped</td><td data-label="누적 비용" class="num">$16.11</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/qurio/">qurio</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 22:03</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/qurio/pull/13">PR #13</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/qurio/releases/tag/v1.4.0">v1.4.0</a></td><td data-label="누적 비용" class="num">$26.07</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/releasedock/">releasedock</a> <span class="pill pill-merged" title="14일: 릴리즈 14, 실패 0, 경고 2, 회귀 1">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-10 15:54</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/releasedock/pull/16">PR #16</a>, released <a href="https://github.com/hkjang/releasedock/releases/tag/v0.5.15">v0.5.15</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/releasedock/releases/tag/v0.5.15">v0.5.15</a> <span class="meta">자산 2</span></td><td data-label="누적 비용" class="num">$38.25</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/relio/">relio</a> <span class="pill pill-failed" title="14일: 릴리즈 13, 실패 0, 경고 7, 회귀 3">건강 D</span></td><td data-label="자율화"><span class="pill pill-merged" title="자율화 단계 low-risk — 강등: 롤백 PR  (2026-09-06)">저위험 자동 병합</span> ⬇</td><td data-label="마지막 회차">2026-09-14 11:42</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/relio/releases/tag/v1.11.20">v1.11.20</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/relio/releases/tag/v1.11.20">v1.11.20</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$70.34</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/ReSSO/">ReSSO</a> <span class="pill pill-failed" title="14일: 릴리즈 15, 실패 0, 경고 5, 회귀 1">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 00:52</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/ReSSO/pull/21">PR #21</a>, released <a href="https://github.com/hkjang/ReSSO/releases/tag/v0.9.84">v0.9.84</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/ReSSO/releases/tag/v0.9.84">v0.9.84</a> <span class="meta">자산 2</span></td><td data-label="누적 비용" class="num">$127.29</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/seaton/">seaton</a> <span class="pill pill-released" title="14일: 릴리즈 2, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-13 19:41</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/seaton/pull/27">PR #27</a>, released <a href="https://github.com/hkjang/seaton/releases/tag/v1.4.3">v1.4.3</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/seaton/releases/tag/v1.4.3">v1.4.3</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$33.10</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/SecCheck/">SecCheck</a> <span class="pill pill-failed" title="14일: 릴리즈 1, 실패 0, 경고 4, 회귀 1">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 13:25</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/SecCheck/releases/tag/v1.0.145">v1.0.145</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/SecCheck/releases/tag/v1.0.145">v1.0.145</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$90.11</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/sqlon/">sqlon</a> <span class="pill pill-released" title="14일: 릴리즈 1, 실패 0, 경고 0, 회귀 0">건강 A</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 03:19</td><td data-label="결과"><span class="pill pill-other">• 기타</span> CI no-ci, PR open <a href="https://github.com/hkjang/sqlon/pull/4">PR #4</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/sqlon/releases/tag/v0.1.5">v0.1.5</a> <span class="meta">자산 6</span></td><td data-label="누적 비용" class="num">$21.52</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/trace/">trace</a> <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0">건강 B</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 19:18</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/trace/pull/1">PR #1</a></td><td data-label="최근 릴리즈"></td><td data-label="누적 비용" class="num">$2.64</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/umm/">umm</a> <span class="pill pill-merged" title="14일: 릴리즈 12, 실패 0, 경고 3, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 13:46</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/umm/releases/tag/v0.75.0">v0.75.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/umm/releases/tag/v0.75.0">v0.75.0</a> <span class="meta">자산 3</span></td><td data-label="누적 비용" class="num">$80.02</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/Vendra/">Vendra</a> <span class="pill pill-failed" title="14일: 릴리즈 13, 실패 0, 경고 4, 회귀 1">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 09:31</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/Vendra/pull/123">PR #123</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/Vendra/releases/tag/v0.7.53">v0.7.53</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$146.11</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/vibe-coders/">vibe-coders</a> <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-13 23:09</td><td data-label="결과"><span class="pill pill-other">• 기타</span> review held, PR open <a href="https://github.com/hkjang/vibe-coders/pull/17">PR #17</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/vibe-coders/releases/tag/v0.85.3">v0.85.3</a></td><td data-label="누적 비용" class="num">$38.42</td></tr><tr data-status="released"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/visitflow/">visitflow</a> <span class="pill pill-failed" title="14일: 릴리즈 7, 실패 1, 경고 6, 회귀 0">건강 D</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-14 12:17</td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> release-only, released <a href="https://github.com/hkjang/visitflow/releases/tag/v2.8.0">v2.8.0</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/visitflow/releases/tag/v2.8.0">v2.8.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$41.72</td></tr><tr data-status="other"><td data-label="프로젝트 · 건강" class="primary"><a href="https://hkjang.github.io/aidev/projects/weekly/">weekly</a> <span class="pill pill-merged" title="14일: 릴리즈 10, 실패 0, 경고 2, 회귀 0">건강 C</span></td><td data-label="자율화"><span class="pill pill-released" title="자율화 단계 release">검증된 릴리즈 게시</span></td><td data-label="마지막 회차">2026-09-16 08:07</td><td data-label="결과"><span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/weekly/pull/15">PR #15</a></td><td data-label="최근 릴리즈"><a href="https://github.com/hkjang/weekly/releases/tag/v0.301.0">v0.301.0</a> <span class="meta">자산 1</span></td><td data-label="누적 비용" class="num">$100.14</td></tr></tbody></table></div>

## 오늘 상한

<dl class="kv">
<dt>비용</dt><dd><span class="meta">111.26 / 300</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:37%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>회차</dt><dd><span class="meta">16 / 100</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:16%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>릴리즈</dt><dd><span class="meta">1 / 60</span> <span style="display:inline-block;width:6rem;height:.5rem;background:var(--card-2);border-radius:4px;vertical-align:middle"><span style="display:block;width:1%;height:100%;background:var(--good);border-radius:4px"></span></span></dd>
<dt>휴면 규칙</dt><dd>변경 없음 3회 연속이면 7일 제외</dd>
<dt>수동 실행</dt><dd><a href="https://github.com/hkjang/aidev/issues/new?labels=run&title=run%3A+%3C%ED%94%84%EB%A1%9C%EC%A0%9D%ED%8A%B8%3E">이슈 만들기</a> — 제목 <code>run: &lt;프로젝트&gt;</code>, 라벨 <code>run</code> → 다음 회차 우선 실행</dd>
</dl>

설정: `state/caps.env`. 상한에 닿으면 그날은 새 회차를 시작하지 않고 알린다.

## 품질 지표 (최근 14일)

<ul class="stats"><li title="관찰 24h 지난 머지 427건 중 회귀 없음 386건"><b>90%</b><span>검증된 개선 완료율</span></li><li title="릴리즈 시도 323건 중 자산 검증까지 298건"><b>92%</b><span>완전한 릴리즈 비율</span></li><li title="되돌림·롤백 0건 / 관찰 머지 427건"><b>0%</b><span>사람의 재작업률</span></li><li title="회귀 0건 / 관찰 머지 427건"><b>0%</b><span>변경 후 회귀율</span></li><li title="비용 확인된 유효 개선 184건, 미확인 세션 10"><b>$14.67</b><span>유효 개선당 비용</span></li><li title="해결된 경고 93건"><b>7.0시간</b><span>예외 처리 소요 시간(중앙값)</span></li><li title="최근 14일"><b>70</b><span>실행 오류</span></li></ul>

'검증된 개선'은 머지 후 24시간 관찰에서 회귀(main CI 실패·되돌림·롤백)가 없는 변경. '완전한 릴리즈'는 태그·Release·필수 자산 검증까지 끝난 것. 비용이 확인되지 않은 세션은 0이 아니라 '미확인'으로 뺀다.

## 비용·사용량

<ul class="stats"><li><b>$111.26</b><span>오늘 비용</span></li><li><b>3시간 46분</b><span>오늘 에이전트 시간</span></li><li><b>21</b><span>오늘 세션</span></li><li><b>$2699.89</b><span>누적 비용</span></li><li><b>110시간 53분</b><span>누적 시간</span></li><li><b>2865.1M/21.4M</b><span>누적 토큰 입력/출력</span></li></ul>

claude -p 가 세션마다 보고한 추정값(정액제에서는 참고값). 회차별 내역은 각 일일 보고와 프로젝트 페이지, 원본은 [usage.jsonl](https://hkjang.github.io/aidev/data/usage.jsonl).

## 개선 캠페인

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">캠페인</th><th>목표</th><th>대상</th><th class="num">예산</th><th>기한</th><th class="num">회차</th><th>상태</th></tr></thead><tbody><tr data-status="released"><td data-label="캠페인" class="primary">silent-sso-2026-09</td><td data-label="목표">Keycloak(또는 ReSSO)에 이미 로그인한 사람이 이 서비스를 열면 로그인 화면 없이 바로 본 화면으로
들어가게 만든다. OIDC 의 prompt=none 을 쓰는 silent authentication 이다.

표준은 aidev 저장소의 SILENT-SSO-STANDARD.md</td><td data-label="대상">AgentHub, Invenqor, Kkiit, ReSSO, SecCheck, Vendra, ai-admin, appstore, cutover, dataworks, igame, jikim, jupiq, moina, moyro, muni, ptium, relio, seaton, umm, vibe-coders, visitflow, weekly, postra, DartFly, madi, qurio, ssak, kanvas, hunter, orbit, trace, sqlon</td><td data-label="예산" class="num">$228.28 / $300</td><td data-label="기한">2026-12-15</td><td data-label="회차" class="num">32</td><td data-label="상태">진행</td></tr><tr data-status="released"><td data-label="캠페인" class="primary">tracking-2026-09</td><td data-label="목표">관리자가 화면에서 방문 추적 스크립트를 붙일 수 있는 체계를 이 저장소에 만든다.

표준과 참조 구현은 aidev 저장소의 TRACKING-STANDARD.md 에 있다. 먼저 읽고 그대로 따른다:
  /mnt/c/Users/USER/projects/aidev/TRACKING-STAND</td><td data-label="대상">AgentHub, Invenqor, Kkiit, ReSSO, SecCheck, Vendra, ai-admin, appstore, cutover, dataworks, igame, jikim, jupiq, moina, moyro, muni, ptium, relio, seaton, umm, vibe-coders, visitflow, weekly, postra, DartFly, madi, qurio, ssak, kanvas, hunter, orbit, trace, sqlon</td><td data-label="예산" class="num">$360.09 / $500</td><td data-label="기한">2026-10-31</td><td data-label="회차" class="num">33</td><td data-label="상태">진행</td></tr><tr data-status="released"><td data-label="캠페인" class="primary">handoff-2026-09</td><td data-label="목표">이 서비스가 다른 사내 서비스와 문서를 주고받을 수 있게 만든다.

표준은 aidev 저장소의 HANDOFF-STANDARD.md 에 있다. 먼저 읽고 그대로 따른다.
엔드포인트 이름·요청 모양·응답 코드를 임의로 바꾸지 마라 — 여섯 서비스가 서로 맞물려야
하므로, 한 곳이라도 다르게</td><td data-label="대상">umm, muni, kanpic, ptium, weekly, postra, DartFly, madi, qurio, ssak, kanvas, hunter, orbit, trace, sqlon</td><td data-label="예산" class="num">$128.48 / $250</td><td data-label="기한">2026-11-15</td><td data-label="회차" class="num">15</td><td data-label="상태">진행</td></tr><tr data-status="released"><td data-label="캠페인" class="primary">mail-2026-09</td><td data-label="목표">사내 SMTP 릴레이로 이벤트 알림을 보내는 체계를 이 저장소에 만든다.

표준과 참조 구현은 aidev 저장소의 MAIL-STANDARD.md 에 있다. 먼저 읽고 그대로 따른다:
  /mnt/c/Users/USER/projects/aidev/MAIL-STANDARD.md

참조 구현</td><td data-label="대상">AgentHub, Invenqor, Kkiit, ReSSO, SecCheck, Vendra, ai-admin, appstore, cutover, dataworks, igame, jikim, jupiq, moina, moyro, muni, ptium, relio, seaton, umm, vibe-coders, visitflow, weekly, postra, DartFly, madi, qurio, ssak, kanvas, hunter, orbit, trace, sqlon</td><td data-label="예산" class="num">$177.71 / $600</td><td data-label="기한">2026-11-30</td><td data-label="회차" class="num">15</td><td data-label="상태">진행</td></tr><tr data-status="nochange"><td data-label="캠페인" class="primary">guides-2026-09</td><td data-label="목표">이 저장소의 사용자 가이드와 관리자 가이드를 화면 캡처가 들어간 완성본으로 만든다.

표준은 aidev 저장소의 GUIDE-STANDARD.md 에 있다. 먼저 읽고 그대로 따른다:
  /mnt/c/Users/USER/projects/aidev/GUIDE-STANDARD.md

산출물은</td><td data-label="대상">AgentHub, Invenqor, Kkiit, Momento, ReSSO, SecCheck, Vendra, ai-admin, appstore, cutover, dataworks, igame, jikim, jupiq, kanpic, moina, moyro, muni, ptium, relio, seaton, umm, vibe-coders, visitflow, weekly</td><td data-label="예산" class="num">$454.53 / $700</td><td data-label="기한">2026-10-15</td><td data-label="회차" class="num">83</td><td data-label="상태">완료</td></tr></tbody></table></div>

설정: `state/campaigns.json`. 캠페인 대상은 수정·수동 큐 다음 순서로 우선 배정되고, 예산·기한이 다하면 자동 종료된다.

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
