# 개선 로드맵

## 목표

`vibe-code`의 목표는 VS Code 안에서 한국어를 기본으로 쓰는 바이브 코딩 도구가 되는 것입니다. 단순 챗봇이 아니라 코드 이해, 수정, 검증, 작업 기록, 팀 공유, 폐쇄망 대응까지 한 흐름으로 이어지는 확장 프로그램을 지향합니다.

## 우선순위 1: 릴리즈 안정화

- VSIX 검증 자동화 강화: CLI 기반 격리 설치 smoke test와 Extension Host command smoke test는 추가됨. 다음 단계로 웹뷰 로드 테스트를 추가합니다.
- 원본 소스 복원: 자체 기능은 `src/` TypeScript로 복원되어 esbuild로 빌드됨(1.2.0). 코어는 `vendor/extension.core.js`로 분리되어 훅 세 개(`beforeCore`, `mergeLocaleOverrides`, `onCommand`)만 호출함.
- 웹뷰 UI 원본 소스: 확보 불가로 판단(2026-09-22). 코어/웹뷰 번들에는 공개 상류(Roo-Code)에 없는 메모리 뱅크, 문서 검색, 그래프 시각화 기능이 포함되어 있고 저장소 표식이 없어 비공개 포크로 보입니다. 화면 확장은 자체 웹뷰 패널(예: `vibe-code.openUsageDashboard`)로 추가합니다.
- CI 파이프라인: `.github/workflows/ci.yml`이 check → Windows 패키징 → 검증 → SHA256 → artifact 업로드까지 수행함. 다음 단계로 태그 push 시 GitHub release 자동 생성.
- 설정 마이그레이션: 기존 사용자의 오래된 설정 키를 `vibe-code.*`로 안전하게 옮깁니다.
- Provider 복원 안전장치: `vibe-coders` 프록시 적용 전 profile 이름 저장과 이전 provider 복원 명령은 추가됨.

## 우선순위 2: 한국어 UX 완성도

- 첫 실행 온보딩: `Vibe Code: 시작 설정` 마법사(프로바이더/네트워크/자율성 3단계)가 추가됨. 다음 단계로 언어·톤 선택과 팀 preset 가져오기를 같은 흐름에 붙입니다.
- 프롬프트 프리셋 관리: 리뷰, 리팩터링, 테스트, 문서화, 커밋 메시지용 한국어 프롬프트를 UI에서 편집합니다.
- 한국어 코드 리뷰 모드 강화: 존댓말/친근체/간결체 톤을 작업별로 적용합니다.
- KST 작업 일지 고도화: 완료 작업, 실패 원인, 다음 액션을 자동 요약해 journal에 남깁니다.
- `/goal` 자율 개발 루프: 목표 파일, 작업 큐, 검증 로그, handoff 파일, 현재 목표 열기/목표 상태 보기/핸드오프 생성 명령, 진행률 포함 active goal 상태바, `현재 목표` 트리 뷰, `현재 계획` 트리 뷰, 최신 계획 열기/계획 진행/계획 상태 변경/goal 연결/완료 archive/restore/active plan 선택 명령, workspace별 audit JSONL은 추가됨. 다음 단계로 자동 handoff 강화와 plan 관계 시각화를 붙입니다.

## 우선순위 3: 개발 생산성

- 작업 계획 보드: `.vibe-code/plans/` 최신 파일을 UI에서 보고 여는 흐름과 상태 변경/단계 이동 명령, plan-to-goal 링크 편집, 완료 plan archive 분리, archive 복원, active plan 선택은 추가됨. 다음 단계로 다중 계획 선택, archive 복원 히스토리, goal-plan 관계 시각화를 붙입니다.
- 컨텍스트 품질 점수: 현재 선택한 파일/검색 결과/임베딩 컨텍스트가 답변에 충분한지 표시합니다.
- 테스트 우선 실행: 변경 파일과 연관된 테스트 명령을 추천하고 결과를 대화에 반영합니다.
- PR/커밋 보조: diff 기반 한국어 요약, conventional commit, PR 본문 생성을 통합합니다.

## 우선순위 4: 팀과 폐쇄망

- 팀 설정 프로파일: `.vibemodes`, `.vibeignore`, MCP 추천, 프롬프트를 하나의 profile로 관리합니다.
- 내부 모델 게이트웨이 preset: Ollama, LM Studio, vLLM, 사내 OpenAI-compatible endpoint 설정을 빠르게 선택합니다.
- 오프라인 문서 검색: 로컬 README, wiki, ADR, journal, plans만 대상으로 검색하는 전용 흐름을 강화합니다.
- 사내 VSIX 업데이트: `vibe-code.updateSourcePath` 기반 버전 비교와 설치 UX를 더 명확히 만듭니다.
- `vibe-coders` 팀 preset: base URL, 기본 모델, `X-Proxy-Provider` 헤더를 팀 설정 zip에 포함해 여러 워크스페이스에 배포합니다.
- `vibe-coders` 호출 감사 보기: Output 명령으로 현재 profile/base URL/model/header/마지막 적용 시각 확인, workspace별 audit JSONL 저장, 상태바 표시까지 추가됨. 다음 단계로 provider 복원 히스토리 UI를 붙입니다.

## 우선순위 5: 관측성과 안전

- 명령 실행 감사 로그: `onCommand` 훅으로 승인/거부/종료 코드가 `.vibe-code/audit/`에 저장되고 `명령 실행 이력 보기` 명령과 journal 세션 요약에 표시됨. 다음 단계로 명령 출력 요약과 실패 원인 분류.
- 위험 작업 정책 UI: destructive command, secrets, protected path 규칙을 설정 화면에서 관리합니다.
- 비용/토큰 대시보드: vibe-coders `/me/report` 기반 사용량 리포트 명령, 웹뷰 대시보드, 상태바 주간 비용 표시가 추가됨. 다음 단계로 세션(task)별 토큰과 자동 압축 시점 표시.
- 장애 복구 리포트: 실패한 작업을 원인/시도/다음 조치로 구조화해 재시도 품질을 높입니다.

## 목표 기반 지속 개발 (1.4.0에서 추가됨)

- 확장이 루프를 굴림: 프롬프트 컨텍스트 훅, 작업 완료 확인, 자동 재개, 자동 handoff(컨텍스트 압축·세션 종료).
- 검증: 검증 계획 CodeLens 실행기, 변경 파일 테스트 추천, 계획/목표 완료 게이트.
- 상태 품질: 정체 감지(상태바 경고), 목표/계획 린트 Diagnostics.
- 다중 목표: 목표 목록 뷰, 전환, 프리셋 기반 새 목표, 계획→완료 기준 연결.
- 회고/지표: 7일 지표, 주간 회고 파일.
- 안전장치: 파괴적 명령 전 git 체크포인트, 주간 예산 가드.

다음 후보: 세션(task)별 토큰·비용 표시, 명령 출력 요약과 실패 원인 분류, 태그 push 시 GitHub release 자동 생성, 팀 preset과 온보딩 마법사 통합, 목표 템플릿 사용자 정의.

## 다음 개발 제안

다음으로 진행할 만한 작업은 다중 plan 선택 UI, archive 복원 히스토리, 자동 handoff 강화입니다. 현재는 목표/계획 보드, active plan 선택, plan 상태 전환, 단계 승급, goal 연결, 완료 plan archive/restore, 프록시 적용/상태 보기/점검/복원이 가능하므로 다음 단계에서는 plan 수명주기 시각화와 팀 preset 배포를 묶는 편이 적절합니다.
