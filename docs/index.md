---
title: "aidev 자율 개선 대시보드"
description: "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 63회차·릴리즈 46건, 누적 113회차·릴리즈 74건."
last_modified_at: 2026-09-03 14:47:45 +0900
---

<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "WebSite",
 "name": "aidev 자율 개선 대시보드",
 "url": "https://hkjang.github.io/aidev/",
 "description": "Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 63회차·릴리즈 46건, 누적 113회차·릴리즈 74건.",
 "inLanguage": "ko",
 "author": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-03T14:47:45"
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
    "text": "에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다."
   }
  },
  {
   "@type": "Question",
   "name": "한 회차에서 에이전트는 무엇을 하나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열어 머지하고, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로)을 확인해 같은 방식으로 다음 버전을 냅니다."
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
   "name": "원본 데이터는 어디서 보나요?",
   "acceptedAnswer": {
    "@type": "Answer",
    "text": "회차 기록은 runs.jsonl(한 줄에 한 회차, JSON), 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다."
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
   "name": "일일 보고 2026-09-03",
   "url": "https://hkjang.github.io/aidev/reports/2026-09-03/"
  },
  {
   "@type": "ListItem",
   "position": 2,
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
  }
 ]
}
</script>

# aidev 자율 개선 대시보드

<p class="tldr"><strong>한 줄 요약.</strong> Claude Code 자율 개선 에이전트가 hkjang 의 프로젝트를 개선·테스트·머지·릴리즈한 일일 보고. 오늘 63회차·릴리즈 46건, 누적 113회차·릴리즈 74건. 회차가 끝날 때마다 자동 갱신됩니다 (마지막 갱신 2026-09-03 14:47).</p>

[운영 문서](https://github.com/hkjang/aidev#readme) · [원장](https://github.com/hkjang/aidev/tree/main/state) · [실행 이력](https://github.com/hkjang/aidev/commits/main) · [원본 데이터](https://hkjang.github.io/aidev/data/runs.jsonl)

## 오늘 (2026-09-03)

<ul class="stats"><li><b>63</b><span>회차</span></li><li><b>27</b><span>프로젝트</span></li><li><b>46</b><span>릴리즈</span></li><li><b>17</b><span>머지(릴리즈 없음)</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li></ul>

[2026-09-03 보고 자세히 보기 →](https://hkjang.github.io/aidev/reports/2026-09-03/)

| 시각 | 프로젝트 | 결과 |
|---|---|---|
| 12:04 | Clustara | 🚀 릴리즈 merged [PR #5](https://github.com/hkjang/clustara/pull/5), released [v0.9.266](https://github.com/hkjang/clustara/releases/tag/v0.9.266) |
| 12:20 | Invenqor | 🚀 릴리즈 merged [PR #5](https://github.com/hkjang/invenqor/pull/5), released [v0.2.22](https://github.com/hkjang/invenqor/releases/tag/v0.2.22) |
| 12:49 | ReSSO | ✅ 머지 merged [PR #6](https://github.com/hkjang/ReSSO/pull/6), release missing |
| 13:03 | Vendra | 🚀 릴리즈 merged [PR #105](https://github.com/hkjang/Vendra/pull/105), released [v0.7.39](https://github.com/hkjang/Vendra/releases/tag/v0.7.39) |
| 13:20 | ai-admin | 🚀 릴리즈 merged [PR #4](https://github.com/hkjang/ai-admin/pull/4), released [v1.2.4](https://github.com/hkjang/ai-admin/releases/tag/v1.2.4) |
| 13:35 | aiportal-front-admin | ✅ 머지 merged [PR #5](https://github.com/hkjang/aiportal-front-admin/pull/5), release skipped |
| 13:47 | aiportal-front | ✅ 머지 merged [PR #1](https://github.com/hkjang/aiportal-front/pull/1), release skipped |
| 13:56 | aiportal-java | ✅ 머지 merged [PR #6](https://github.com/hkjang/aiportal-java/pull/6), release skipped |
| 14:06 | aiportal-py | ✅ 머지 merged [PR #4](https://github.com/hkjang/aiportal-py/pull/4), release skipped |
| 14:18 | dataworks | 🚀 릴리즈 merged [PR #5](https://github.com/hkjang/dataworks/pull/5), released [v0.9.40](https://github.com/hkjang/dataworks/releases/tag/v0.9.40) |
| 14:30 | git-ctx | 🚀 릴리즈 merged [PR #16](https://github.com/hkjang/git-ctx/pull/16), released [v0.77.3](https://github.com/hkjang/git-ctx/releases/tag/v0.77.3) |
| 14:47 | igame | 🚀 릴리즈 merged [PR #4](https://github.com/hkjang/igame/pull/4), released [v0.7.4](https://github.com/hkjang/igame/releases/tag/v0.7.4) |

## 일일 보고

| 날짜 | 회차 | 릴리즈 | 머지 | 변경 없음 | 실패 |
|---|---|---|---|---|---|
| [2026-09-03](https://hkjang.github.io/aidev/reports/2026-09-03/) | 63 | 46 | 17 | 0 | 0 |
| [2026-09-02](https://hkjang.github.io/aidev/reports/2026-09-02/) | 50 | 28 | 16 | 3 | 0 |

## 프로젝트별 현황

| 프로젝트 | 마지막 회차 | 결과 | 최근 릴리즈 |
|---|---|---|---|
| [AgentHub](https://github.com/hkjang/AgentHub) | 2026-09-03 00:17 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/AgentHub/pull/3), released [v0.228.0](https://github.com/hkjang/AgentHub/releases/tag/v0.228.0) | [v0.230.0](https://github.com/hkjang/AgentHub/releases/tag/v0.230.0) |
| [ai-admin](https://github.com/hkjang/ai-admin) | 2026-09-03 01:36 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/ai-admin/pull/3), released [v1.2.3](https://github.com/hkjang/ai-admin/releases/tag/v1.2.3) | [v1.2.4](https://github.com/hkjang/ai-admin/releases/tag/v1.2.4) |
| [aiportal-front](https://github.com/hkjang/aiportal-front) | 2026-09-03 13:47 | ✅ 머지 merged [PR #1](https://github.com/hkjang/aiportal-front/pull/1), release skipped | skipped |
| [aiportal-front-admin](https://github.com/hkjang/aiportal-front-admin) | 2026-09-03 01:47 | ✅ 머지 merged [PR #3](https://github.com/hkjang/aiportal-front-admin/pull/3), release skipped | skipped |
| [aiportal-java](https://github.com/hkjang/aiportal-java) | 2026-09-03 01:55 | ✅ 머지 merged [PR #3](https://github.com/hkjang/aiportal-java/pull/3), release skipped | skipped |
| [aiportal-py](https://github.com/hkjang/aiportal-py) | 2026-09-03 02:07 | ✅ 머지 merged [PR #3](https://github.com/hkjang/aiportal-py/pull/3), release skipped | skipped |
| [appstore](https://github.com/hkjang/appstore) | 2026-09-03 02:16 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/appstore/pull/3), released [v2.1.3](https://github.com/hkjang/appstore/releases/tag/v2.1.3) | [v2.1.4](https://github.com/hkjang/appstore/releases/tag/v2.1.4) |
| [Clustara](https://github.com/hkjang/clustara) | 2026-09-03 00:29 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/clustara/pull/3), released [v0.9.264](https://github.com/hkjang/clustara/releases/tag/v0.9.264) | [v0.9.266](https://github.com/hkjang/clustara/releases/tag/v0.9.266) |
| [dataworks](https://github.com/hkjang/dataworks) | 2026-09-03 02:29 | 🚀 릴리즈 merged [PR #2](https://github.com/hkjang/dataworks/pull/2), released [v0.9.37](https://github.com/hkjang/dataworks/releases/tag/v0.9.37) | [v0.9.40](https://github.com/hkjang/dataworks/releases/tag/v0.9.40) |
| [git-ctx](https://github.com/hkjang/git-ctx) | 2026-09-03 02:40 | ✅ 머지 merged [PR #15](https://github.com/hkjang/git-ctx/pull/15), release missing | [v0.77.3](https://github.com/hkjang/git-ctx/releases/tag/v0.77.3) |
| [igame](https://github.com/hkjang/igame) | 2026-09-03 02:56 | 🚀 릴리즈 merged [PR #2](https://github.com/hkjang/igame/pull/2), released [v0.7.2](https://github.com/hkjang/igame/releases/tag/v0.7.2) | [v0.7.4](https://github.com/hkjang/igame/releases/tag/v0.7.4) |
| [Invenqor](https://github.com/hkjang/invenqor) | 2026-09-03 00:39 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/invenqor/pull/3), released [v0.2.20](https://github.com/hkjang/invenqor/releases/tag/v0.2.20) | [v0.2.22](https://github.com/hkjang/invenqor/releases/tag/v0.2.22) |
| [kanpic](https://github.com/hkjang/kanpic) | 2026-09-03 03:06 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/kanpic/pull/3), released [v0.230.0](https://github.com/hkjang/kanpic/releases/tag/v0.230.0) | [v0.231.0](https://github.com/hkjang/kanpic/releases/tag/v0.231.0) |
| [moina](https://github.com/hkjang/moina) | 2026-09-03 03:22 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/moina/pull/3), released [v0.1.18](https://github.com/hkjang/moina/releases/tag/v0.1.18) | [v0.1.19](https://github.com/hkjang/moina/releases/tag/v0.1.19) |
| [moyro](https://github.com/hkjang/moyro) | 2026-09-03 03:44 | 🚀 릴리즈 merged [PR #4](https://github.com/hkjang/moyro/pull/4), released [v0.2.13](https://github.com/hkjang/moyro/releases/tag/v0.2.13) | [v0.2.14](https://github.com/hkjang/moyro/releases/tag/v0.2.14) |
| [muni](https://github.com/hkjang/muni) | 2026-09-03 04:00 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/muni/pull/3), released [v0.24.0](https://github.com/hkjang/muni/releases/tag/v0.24.0) | [v0.25.0](https://github.com/hkjang/muni/releases/tag/v0.25.0) |
| [pii-masker](https://github.com/hkjang/pii-masker) | 2026-09-03 04:16 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/pii-masker/pull/3), released [v1.0.6](https://github.com/hkjang/pii-masker/releases/tag/v1.0.6) | [v1.0.7](https://github.com/hkjang/pii-masker/releases/tag/v1.0.7) |
| [ptium](https://github.com/hkjang/ptium) | 2026-09-03 04:28 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/ptium/pull/3), released [v1.69.21](https://github.com/hkjang/ptium/releases/tag/v1.69.21) | [v1.69.22](https://github.com/hkjang/ptium/releases/tag/v1.69.22) |
| [Quantoss](https://github.com/hkjang/Quantoss) | 2026-09-03 00:46 | ✅ 머지 merged [PR #1](https://github.com/hkjang/Quantoss/pull/1), release skipped | skipped |
| [releasedock](https://github.com/hkjang/releasedock) | 2026-09-03 04:36 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/releasedock/pull/3), released [v0.5.3](https://github.com/hkjang/releasedock/releases/tag/v0.5.3) |  |
| [relio](https://github.com/hkjang/relio) | 2026-09-03 04:47 | 🚀 릴리즈 merged [PR #3](https://github.com/hkjang/relio/pull/3), released [v1.11.10](https://github.com/hkjang/relio/releases/tag/v1.11.10) | [v1.11.11](https://github.com/hkjang/relio/releases/tag/v1.11.11) |
| [ReSSO](https://github.com/hkjang/ReSSO) | 2026-09-03 01:04 | 🚀 릴리즈 merged [PR #4](https://github.com/hkjang/ReSSO/pull/4), released [v0.9.67](https://github.com/hkjang/ReSSO/releases/tag/v0.9.67) |  |
| [ReSSO merged PR #2; weekly released v0.281.0 (수동 동기화](https://github.com/hkjang/ReSSO merged PR #2; weekly released v0.281.0 (수동 동기화) | 2026-09-02 12:43 | • 러너 git 식별자 미설정 수정) |  |
| [umm](https://github.com/hkjang/umm) | 2026-09-03 05:00 | 🚀 릴리즈 merged [PR #133](https://github.com/hkjang/umm/pull/133), released [v0.66.1](https://github.com/hkjang/umm/releases/tag/v0.66.1) | [v0.66.3](https://github.com/hkjang/umm/releases/tag/v0.66.3) |
| [Vendra](https://github.com/hkjang/Vendra) | 2026-09-03 01:22 | 🚀 릴리즈 merged [PR #103](https://github.com/hkjang/Vendra/pull/103), released [v0.7.37](https://github.com/hkjang/Vendra/releases/tag/v0.7.37) | [v0.7.39](https://github.com/hkjang/Vendra/releases/tag/v0.7.39) |
| [Vendra/ai-admin/pii-masker/ptium](https://github.com/hkjang/Vendra/ai-admin/pii-masker/ptium) | 2026-09-02 16:32 | • 태그 사후 푸시 및 Release 생성 |  |
| [vibe-coders](https://github.com/hkjang/vibe-coders) | 2026-09-03 05:19 | 🚀 릴리즈 merged [PR #9](https://github.com/hkjang/vibe-coders/pull/9), released [v0.82.0](https://github.com/hkjang/vibe-coders/releases/tag/v0.82.0) | [v0.82.0](https://github.com/hkjang/vibe-coders/releases/tag/v0.82.0) |
| [visitflow](https://github.com/hkjang/visitflow) | 2026-09-03 05:26 | ✅ 머지 merged [PR #3](https://github.com/hkjang/visitflow/pull/3), release missing |  |
| [weekly](https://github.com/hkjang/weekly) | 2026-09-03 06:06 | 🚀 릴리즈 merged [PR #4](https://github.com/hkjang/weekly/pull/4), released [v0.284.0](https://github.com/hkjang/weekly/releases/tag/v0.284.0) | [v0.285.0](https://github.com/hkjang/weekly/releases/tag/v0.285.0) |

## FAQ

<details><summary>이 페이지는 무엇인가요?</summary><p>hkjang 의 GitHub 프로젝트들을 Claude Code 자율 개선 에이전트가 스스로 분석해 개선하고, 테스트를 통과시킨 뒤 PR 을 main 에 머지하고, 각 저장소의 기존 관례대로 릴리즈한 결과를 회차마다 자동으로 갱신하는 일일 보고 대시보드입니다.</p></details>
<details><summary>얼마나 자주 갱신되나요?</summary><p>에이전트는 10분 간격 스케줄로 사실상 연속 실행되며, 회차가 하나 끝날 때마다 이 사이트가 다시 만들어집니다. 보통 1~2분 안에 반영됩니다.</p></details>
<details><summary>한 회차에서 에이전트는 무엇을 하나요?</summary><p>저장소를 파악하고 개선 아이디어 5개를 가치·위험·작업량으로 채점해 하나를 고른 뒤 구현하고, 테스트·린트·빌드를 실제로 실행해 통과한 경우에만 커밋합니다. 러너가 PR 을 열어 머지하고, 릴리즈 에이전트가 이전 릴리즈 방식(태그·버전 파일·CHANGELOG·워크플로)을 확인해 같은 방식으로 다음 버전을 냅니다.</p></details>
<details><summary>어떤 프로젝트가 대상인가요?</summary><p>최근 30일 안에 커밋이 있고, 작업트리가 깨끗하며, GitHub 원격이 있는 저장소만 후보가 됩니다. 사람이 작업 중인(미커밋 변경이 있는) 저장소는 자동으로 제외됩니다.</p></details>
<details><summary>'머지(릴리즈 없음)'는 무슨 뜻인가요?</summary><p>개선은 머지됐지만 릴리즈가 만들어지지 않은 회차입니다. 릴리즈 이력이 전혀 없는 신규 저장소(관례를 새로 정하지 않음)이거나, 릴리즈 단계가 실패·미완료된 경우입니다.</p></details>
<details><summary>원본 데이터는 어디서 보나요?</summary><p>회차 기록은 runs.jsonl(한 줄에 한 회차, JSON), 프로젝트별 원장은 GitHub 저장소 hkjang/aidev 의 state/ 폴더, 러너와 프롬프트는 같은 저장소의 bin/ 과 prompt.md 에 있습니다.</p></details>

---
러너·프롬프트·원장은 [https://github.com/hkjang/aidev](https://github.com/hkjang/aidev) 에서 관리한다. 이 페이지는 회차가 끝날 때마다 `bin/report.py` 가 다시 만든다.
