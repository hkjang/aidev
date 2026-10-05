# 회차 노트 2026-10-05-202733-relio-improve — relio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:27] base pinned — main@deaf151
- [러너 20:27] autonomy low-risk — 롤백 PR 

## 정찰 노트
- 2026-10-04 에 머지된 enum 그물의 가장 큰 구멍(제외된 status·severity = CHECK 42개 중 14개)을 닫는 것을 골랐다. parameters.go 와 migrations/ 를 직접 열어 13쌍 전부의 표를 확정했고 걷기 코드(migrationCheckConstraints 가 이미 (표,컬럼) 키 맵)를 그대로 재사용해 테스트 1파일로 끝난다 — 지난 세 회차 중 두 건이 이 유형으로 머지됐다.
- 제친 후보: release.yml·esbuild 는 보호 경로/네트워크 lock 재생성이라 단독 회차가 맞고, ClientIP·serviceError 는 설정축이나 354 호출부가 붙어 M 을 넘는다. from/to 는 6파일을 넘길 위험이 있어 쪼개야 한다.
- 추측으로 적은 것: 13쌍 표 매핑은 **눈으로 대조**했고 비교를 코드로 돌려 본 것은 아니다(변경 전 기존 테스트가 green 인 것만 실행 확인). 착수 전 임시 t.Logf 로 실제 (operation,파라미터) 쌍 목록을 출력해 표와 맞는지 확인하고 지울 것.
- 구현자 주의: 매핑은 effectiveCheckValues 의 컬럼명 접기를 **대체하지 말고 우회**만 하게 둘 것(6개 키가 그 경로에 얹혀 있다). GET /voices 와 /voices/export 는 voiceFilters() 를 공유하므로 매핑 항목이 각각 필요하다. migrations 섭동은 반드시 되돌리고 knownEnumQueries=26·knownCheckConstraints=42 는 그대로여야 한다.
- 프로필은 0일 전 것이고 이번에 본 코드와 어긋나지 않아 새로 쓰지 않았다(migrations 마지막 017, internal/api 구성 일치 확인).
- [러너 20:33] scout done — `status`·`severity` enum 을 operation→(표, 컬럼) 손 매핑으로 대조해 `enumParametersNotBackedByACheck` 의 두 제외 항목�

## 구현 노트
- enum 그물의 가장 큰 구멍을 닫았다: `enumParameterTables`(operation→파라미터→표) 13쌍을 추가해 `status`·`severity` 를 제외에서 빼고, 매핑된 쌍은 컬럼명 접기를 **우회**해 `migrationCheckConstraints` 의 `(표,컬럼)` 키로 직접 조회한다. 접기 경로는 모호하지 않은 6개 파라미터를 위해 손대지 않았다(대체 아님). 테스트 1파일(+85/−12), 프로덕션·`migrations/`·workflows 0줄. 커밋 909de96.
- **확신 없는 곳**: 표 매핑의 1차 증거는 임시 프로브로 확인한 (a) 공표 집합 = (b) 그 표의 CHECK 집합 일치다. 집합만으로는 "집합이 우연히 같은 다른 표" 를 배제하지 못하므로, 이름이 어긋나는 두 자리를 **실제 필터 SQL 까지 열어** 교차 확인했다: `internal/mail/service.go:277` 이 `FROM mail_deliveries WHERE ($1='' OR status=$1)`, `internal/approval/service.go:399` 가 `FROM approval_requests ar … AND ($3='' OR ar.status=$3)` — 둘 다 매핑한 표 그대로다. 나머지 6개 표도 엔드포인트를 소유한 패키지에서 질의되는 것을 확인했다(voices→`internal/voice`, signals·risks·insights·recommendations→`internal/intelligence`, opportunities→`internal/crm`). 다만 **11쌍은 필터 SQL 을 한 줄씩 다 읽은 것은 아니다**(집합 일치 + 소유 패키지 일치까지). 집합이 겹치는 자리는 `signals`/`risks` severity 뿐이고 둘은 각자 제 표를 가리킨다.
- 일부러 하지 않은 것: 접기 로직을 표 매핑으로 통일하지 않았다(6개를 위한 매핑 항목이 늘고 범위가 터진다). 공표되지 않는 `status` CHECK 3개(`revenue_schedules`·`account_plans`·`personal_keys`)는 질의 파라미터가 없어 매핑에 넣지 않았다 — 이 그물은 문서→migrations 한 방향이다. `forecastCategory`·`sort`·`prompt` 제외 3개와 제외 staleness 가드는 그대로 남겼다.
- 다음 역할이 조심할 것: `make test` 는 돌리지 않았다(과제서 지시 — Go 테스트 1파일이고 `npm ci` 가 네트워크를 탄다). 프런트·Docker·업그레이드 검증은 이번에 미실행. `knownEnumQueries=26`·`knownCheckConstraints=42` 는 의도대로 무변경이고, 이 둘이 바뀌면 걷기가 망가졌다는 신호다. 섭동 8종은 전부 되돌렸고 최종 `git status --short` 는 테스트 1파일이었다. DB 는 필요 없다 — 전부 소스·embed 파싱이다.
- [러너 20:39] brief accepted — 채택 — 과제서의 실측 13행(operation·파라미터·표·공표 집합·CHECK 자리)이 임시 프로브 출력과 **전부 일치**했고, 지정한
- [러너 20:40] verify passed — 검증 10개 통과 (auto)

## 비평 노트
- 구현자가 의심한 자리를 독립 확인: 13쌍 표 매핑을 **전부 실제 필터 SQL 까지** 열어 대조했고 오매핑 없음(voice/service.go:305,307 · crm/service.go:559 · intelligence/queries.go:50-51,106,191,249 · approval/service.go:399 · mail/service.go:277). 새 매핑 분기가 정말 실행되는지 섭동 7종(문서 값 추가 / 다른 실존 표로 교체 / CHECK 없는 표 / 매핑 항목 삭제 / operation 키 개명 / migrations CHECK 확대)으로 확인해 전부 red 였고 되돌린 뒤 baseline PASS, git status 깨끗.
- 주석 검증: ":56 접기가 6개를 담당" 을 임시 프로브로 재현(distinct 6개, 쌍 9개), "status 11표·severity 3표"·"42개 중 14개" 도 섭동 출력과 일치. 독립 실행 go build·vet·gofmt 무출력, `go test ./...` **23 ok / FAIL 0** — 원장 주장과 일치. 범위 이탈·되돌리기 문제 없음.
- 검토 부서: security 차단 없음(인증·인가·비밀값·의존성 무변경, 입력은 embed migrations 와 OpenAPI() 뿐, 공격 경로 없음), legal 차단 없음(개인정보 수집·전송 없음, 새 의존성·외부 약속 문구 없음).
- **승인이어도 남는 우려 — 다음 회차가 알 것**: 가드 3종은 "틀린 표인데 같은 컬럼·같은 값 집합" 을 잡지 못한다. `signals.severity`·`risks.severity` 는 집합이 동일하므로 두 매핑 항목을 서로 바꿔도 green 이다. 매핑에 항목을 추가하는 사람은 집합 일치로 만족하지 말고 필터 SQL 을 읽어야 한다. 또 이 그물은 문서→migrations 한 방향이라 질의 파라미터 없는 status CHECK 3개(revenue_schedules·account_plans·personal_keys)는 여전히 미비교다.
- 못 본 것: make test·프런트(npm)·Docker·업그레이드 검증은 이번에 미실행(테스트 전용 변경이라 적절한 생략이나 릴리즈 단계에서 별도 필요). 판정 **approve** / risk low / blocking 없음.
- [러너 20:44] review approved — 리뷰 승인 (risk=low)
- [러너 20:44] pr created — https://github.com/hkjang/relio/pull/43
- [러너 20:48] ci passed — 검사 2개 모두 success
- [러너 20:48] merge done — 909de96
- [러너 20:48] release skipped — 자율화 단계 low-risk — 릴리즈는 사람이
