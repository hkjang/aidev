# aidev 회사 조직도 — headcount 부서를 붙인 자율 개선 회사

2026-09-19 부터 aidev 는 [cbrock84/headcount](https://github.com/cbrock84/headcount)(MIT, Chris Brock) 의
부서 스킬을 역할마다 싣고, 그 조직 원칙 — **빌더/리뷰어 두 계급, 배타적 쓰기 표면, 권한 세 단계
(autonomous / proposes / escalates), 검토 부서의 차단 소견은 검토받는 부서가 뒤집을 수 없음** — 로 운영한다.
headcount 는 형제 폴더 `../headcount` 에 클론돼 있고, 러너는 역할의 부서 플러그인을 `--plugin-dir` 로
세션에 싣는다(전역 설치 없음). 명부는 `agents/registry.json`, 러너 배선은 `bin/run.sh` 의 `agent_plugin_args`.

## 조직도

| 부서 (headcount) | aidev 역할 | 계급 | 권한 | 쓰기 표면 | 먼저 부르는 스킬 |
|---|---|---|---|---|---|
| Office of the CEO · Finance · PMO · People · Data & Analytics · Corporate Strategy | **이사회** `bin/board.sh` (주 1회) | reviewer | proposes | 없음 — `state/board/<주>.md·json` 에 제안만 | chief-executive, capital-allocation, cost-accounting, portfolio-governance, performance-management, quantitative-analysis |
| Technology · PMO | **정찰** | reviewer | autonomous | 없음 (`brief.md`·`ideas.json`·`profile.md`) | implementation-planning, solution-exploration, estimating-and-contingency |
| Technology | **구현** | builder | proposes (PR) | 그 회차의 worktree | test-driven-development, systematic-debugging, completion-verification |
| Technology · **Security** · **Legal & Risk** | **비평** | reviewer | autonomous | 없음 | code-review, security-architecture-review, privacy-and-data-protection |
| Technology | **수리** | builder | proposes | worktree (비평이 지목한 결함만) | code-review, systematic-debugging, completion-verification |
| Technology | **중재** | reviewer | autonomous (security·legal 차단은 뒤집을 수 없음) | 없음 | code-review |
| Technology · Marketing | **릴리즈** | builder | autonomous | 릴리즈 worktree·태그·자산 | release-and-deployment, product-launch |
| Technology · Security · Legal & Risk | **PR 심사** (처리기) | reviewer | autonomous (recommend 를 러너가 따름) | 없음 | code-review, security-architecture-review, privacy-and-data-protection |
| Technology | **PR 수리** (처리기) | builder | proposes | PR 브랜치 | systematic-debugging, completion-verification |
| PMO · Technology | **캠페인 설계** | builder | escalates (사람이 activate) | `drafts/<slug>/` | program-management, solution-architecture |
| Operations | **진단** `ask-claude.sh` | reviewer | autonomous | 없음 | incident-management |
| — | **기록·학습** | reviewer | autonomous | `state/campaign-lessons/`, `state/operator-preferences.md` | — |
| — | **코파일럿** | orchestrator interface | escalates | `bin/ops.sh` 동사만 | — |
| — | **운영자 hkjang** | CEO · 유일한 사람 | — | 정책·자율화·위험 수용 | — |

러너(`bin/run.sh`)는 headcount 의 오케스트레이터에 해당한다. 어떤 표면도 소유하지 않고, 유일하게 머지·태그·푸시를 한다.

## 세 가지 규칙

1. **만드는 자와 검사하는 자는 다른 세션이다.** 정찰·비평·중재·심사·이사회는 편집할 수 없다(구조적으로 — 도구가 없다). 구현·수리·릴리즈는 자기 표면 밖을 쓰지 않는다.
2. **검토 부서의 차단 소견은 뒤집을 수 없다.** 비평·PR 심사가 `blocking: ["security"|"legal"]` 을 내면 수리도 중재도 처리기도 그것을 풀지 않는다. PR 은 열린 채 운영자에게 간다. 운영자가 위험을 수용하려면 PR 에 `risk-accepted` 라벨을 달고 **이름·사유·만료**를 코멘트로 남긴다 — 조용한 하향은 없다. 그때만 처리기가 다시 심사한다. 공격 경로가 없는 우려는 차단이 아니라 `notes` 다.
3. **권한은 표면과 별개다.** autonomous 는 결과를 그대로 받는다. proposes 는 일은 하되 러너·사람이 diff 를 보고 들인다(PR). escalates 는 시키지 않으면 하지 않는다(캠페인 시작, 코파일럿의 바꾸는 동사).

## 주간 운영 리듬

| 언제 | 누가 | 무엇을 |
|---|---|---|
| 10분마다 | 러너 | 회차(정찰→구현→검증→비평→수리→중재→PR→CI→머지→릴리즈) |
| 매시간 | PR 처리기 | 멈춘 PR 진단·수리·심사·승인 |
| 30분마다 | 헬스체크·기록·학습 | 건강, 캠페인 교훈, 운영자 취향, 머지 뒤 수정률(6h), 실험 표(6h) |
| 월요일(주 1회) | **이사회** | 회사 상태를 읽고 다음 주 결정 3~7개를 ops.sh 동사로 제안 → 텔레그램. 운영자가 `board apply 1 3` 으로 적용 |
| 월말 | 운영자 | 비교 실험 판독, 자율화·예산 정책 조정, 논문 갱신 |

## 끄기와 설정

- 부서 스킬 전체: `state/NO-HEADCOUNT` 또는 정책 `agents.headcount: false`.
- 이사회: `state/NO-BOARD`. 예산 `AIDEV_BOARD_BUDGET`(기본 $4/주).
- 역할별 부서·스킬: `agents/registry.json` 의 `departments`·`skills`.
- headcount 갱신: `git -C ../headcount pull`. 스킬 이름이 바뀌면 명부를 맞춘다.

## 출처

부서·스킬·조직 방법은 headcount(MIT) 의 것이다. 이 문서의 조직도는 그 부서를 aidev 의 역할에 배정한 것이고, 검토 부서의 차단·위험 수용 규칙은 headcount 의 `security-review`·`legal-risk-review` 헌장을 따른다.
