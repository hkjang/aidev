- 과제: 자산 관계 생성에서 명시적 confidence: 0 을 저장·조회·감사까지 보존 (가치 3 / 위험 1 / 작업량 S)
- 왜: `createAssetRelation` 은 공개 계약이 허용하는 0을 범위 검사 뒤 1로 덮어써, 신뢰도 0%인 관계를 100%로 저장하고 반환한다. 생략 기본값과 명시적 0을 구분하면 콘솔·외부 REST·감사 기록이 요청한 신뢰도를 일관되게 유지한다.
- 수용 기준: 1) 두 POST 경로에서 명시적 0은 201이며 DB confidence·같은 인증 경로의 GET relations에서 해당 id의 confidence·relation.create 감사 after_json.confidence 모두 숫자 0이다. 2) 생략 시 기존 기본값 1, 명시적 1과 0.8은 그대로이고 null도 기존과 같이 1로 처리한다(null 허용 정책 강화는 별도 과제). 3) 기존 범위 밖 값의 400 INVALID_RELATION·행/감사 무변경, UUID 검증·정규화와 충돌의 409가 유지되며 실제 Runtime/라우터를 통과하는 테스트가 SQLite·PostgreSQL에서 이를 증명한다.
- 건드릴 파일: `server/internal/httpapi/assets.go:622 createAssetRelation` — decodeJSON 이전 Confidence 기본값을 1로 초기화하고 0→1 후처리를 없앤다. `server/internal/httpapi/asset_relation_validation_test.go:16 relationValidationClient, :294 TestRelationCreateStoresConfidenceWithinDeclaredRange` — 기존 0 기대값과 주석 수정, 생략/null/0/1/0.8의 DB·GET·감사 확인. 프로덕션 1파일 + 기존 테스트 1파일, 총 2파일로 제한한다.
- 검증 명령: 아래 실행 단계의 명령을 사용한다. 정찰에서 `cd server && go test ./internal/httpapi/ -run 'Relation' -count=1` 실제 통과(httpapi 6.202초). 이는 수정 전 동작의 기준선이며 수정 성공 증거는 아니다.
- 위험과 피할 것: auth·session·migrations·workflows·web·PDF·버전·릴리즈 문서는 수정하지 않는다. 범위 검사·UUID 검사·감사 필드명과 숫자 형식은 유지한다. 과거 PostgreSQL/SQLite 차이를 놓쳤으므로 양 DB를 검증한다. 감사 details에 요청 원문이나 검출 문자열을 추가하지 않는다. 기존 행의 confidence 일괄 보정은 하지 않는다.
- 차선 후보: 관계 생성 confidence의 JSON 타입 오류 회귀 테스트 보강(가치 2 / 위험 1 / S) — 1순위가 다른 변경으로 이미 해결됐을 때만 기존 테스트 파일에 문자열·bool·배열·객체·1e309 요청이 두 인증 경로에서 400이고 행/감사 무변경임을 검증한다. 프로덕션 동작 변경이나 공통 decodeJSON 수정은 포함하지 않는다.

근거와 현재 상태

- 기준: `main@ab7c89b`(v0.2.45), 정찰 전후 작업 트리 깨끗함. CLAUDE.md·AGENTS.md·별도 ROADMAP/TODO 파일은 저장소 파일 검색에서 발견하지 못했다. README, docs/README, EXECUTIVE_REPORT의 단계별 도입안, git log -30, CI·테스트 구성과 관련 코드를 확인했다. src/httpapi/web의 지정 확장자 TODO/FIXME 검색은 출력 없음(전체 파일 내용 정독을 뜻하지 않음).
- `openapi.yaml:1885,2193`: 두 POST의 confidence는 number, minimum 0, maximum 1, default 1이며 required 항목이 아니다. 스키마 변경은 필요 없다.
- `assets.go:622-678`: float64 입력 → decodeJSON → UUID 정규화 → [0,1] 검사 → `if input.Confidence == 0 { input.Confidence = 1 }` → INSERT → 입력 구조체를 recordAdminAudit에 전달한다. 기존 테스트 :309는 명시적 0의 DB 기대값을 1로 고정하고 있으며, 이번 SQLite 기준선 실행에서도 통과했다.
- `auth.go:417 decodeJSON`은 전달한 구조체에 JSON decoder로 직접 디코딩한다. 이 함수를 수정하지 않고 호출 전에 `input.Confidence = 1`을 설정하면 생략값을 유지하고 명시적 0은 덮어쓰도록 할 수 있다. null은 float64의 사전값을 유지하는 방식을 이용하되 실제 라우터 테스트로 확인한다.
- `assets.go:574 assetRelations`는 DB confidence를 응답 map에 그대로 넣는다. `recordAdminAudit:983`은 After에 입력을 전달한다. DB만 확인하지 말고 GET과 audit_logs.after_json을 파싱해 확인해야 한다.
- `server.go:225-229,421-425`: 콘솔과 외부 API가 같은 핸들러를 공유한다. 외부 GET은 relations.read가 필요하지만 테스트 `relationValidationClient`의 키는 현재 relations.write만 발급한다. 테스트 키에 read를 추가하거나 읽기용 키를 별도로 발급하라. 프로덕션 권한은 바꾸지 않는다.
- `web/src/operationsPages.tsx:1287,1291-1294`: 신뢰도를 백분율로 표시하며 생성 폼도 존재하고 confidence: 1을 전송한다. 이전 프로필의 '콘솔은 생성하지 않는다'는 설명은 틀리므로 따르지 않는다. 이번 변경은 기존 UI 생성의 기본값을 유지한다.

대안 비교와 선택

1. 권장: 기존 float64에 decode 전 기본값 1 설정 + 0 후처리 제거. 프로덕션 몇 줄이고 기존 감사 직렬화·null 처리를 유지한다.
2. *float64로 존재 여부를 표현하고 유효값을 별도로 계산. 생략/숫자 구분은 가능하지만 감사에서 nil이 나가지 않도록 추가 처리가 필요하다. null과 생략을 다르게 규정할 요구가 없는 이번 범위에는 불필요하다.
3. 사용자 문서에 0 대신 작은 양수를 쓰게 안내하거나 현행 유지. 저장 신뢰도의 의미를 훼손하므로 선택하지 않는다. 모든 입력을 공통 파서/DB 기본값으로 통합하는 확대안도 이번 범위에 필요 없다.
핵심 가정: 생략·null의 기존 기본값 1을 유지하면서 명시적 숫자 0만 바로잡는다. null을 400으로 바꿀 별도 요구는 확인되지 않았다.

실행 단계(구현 전부 미착수; 단계별 사람 확인 없음)

1. 기준선과 실패 재현: 위 기존 테스트의 0 사례 이름을 보존 의미로 바꾸고 기대값을 0으로 바꾼 뒤 `cd server && go test ./internal/httpapi/ -run '^TestRelationCreateStoresConfidenceWithinDeclaredRange$' -count=1` 실행. 두 인증 경로에서 stored confidence=1, want 0 실패를 확인한다. 이 실패는 재현 체크포인트이며 그대로 커밋하지 않고 다음 수정과 함께 하나의 정상 단위로 마무리한다. 예상과 다르면 과제서를 보정하고 범위를 확대하지 않는다.
2. 최소 수정과 회귀 증명: `createAssetRelation` 기본값 초기화/후처리를 수정한다. 같은 파일의 `relationConfidenceBody`, `relationConfidence`, `createdRelationID`, `assertRelationResponse`와 감사 파싱 예시 `TestRelationUpperCaseIDsAreCanonicalInRowsAndAudit`를 재사용한다. null은 helper의 nil이 생략을 뜻하므로 raw JSON으로 요청한다. 성공 사례별로 생성 id를 기준으로 GET items를 찾고 감사 confidence의 숫자 값과 존재 여부를 검사한다(숫자 누락을 Go zero value로 오인하지 말 것). `cd server && go test ./internal/httpapi/ -run 'Relation' -count=1` 통과가 다음 단계 체크포인트다.
3. 통합 검증: 저장소 루트에서 `POSTGRES_CONTAINER=invenqor-impl-20261008-043853-confidence POSTGRES_PORT=55548 ./scripts/test-postgres.sh -run 'Relation' -count=1`; 이어 `cd server && go test ./... && go vet ./...` 및 `cd server && gofmt -l internal/httpapi/assets.go internal/httpapi/asset_relation_validation_test.go`. PostgreSQL 스크립트는 실제 존재하고 인자를 go test에 전달한다. 해당 포트의 가용성과 Docker는 정찰 미확인이므로 실행 전에 확인하고 충돌하면 빈 포트로 바꾼다. 스크립트는 지정 컨테이너를 먼저 삭제하므로 기존 서비스 이름을 사용하지 않는다. 최종 diff가 지정한 2파일인지 확인한다.

작업량과 예비 시간

- 상향식 추정: 기준선/실패 재현 4–6분, 최소 수정 3–5분, 양 인증 경로 GET·감사·null 테스트 10–14분, 양 DB/전체 suite/vet/검토 8–12분. 기본 25–37분, 알려진 변동(Docker 시작·키 scope 실수) 예비 3–6분을 별도로 더해 28–43분. 45분 내 가능성은 중간 수준의 판단이며 통계적 보장이나 약속이 아니다.
- 유사 사례 확인: 10/06 confidence 범위 수정도 같은 프로덕션 1파일과 테스트 헬퍼를 사용했다. 완료 소요 시간이 기록에 없어 수치 비율로 교차 산정하지 않았다. 상향식과 과거 파일/검증 범위만 비교했다.
- 전제: 기존 Go 캐시 및 Docker 사용 가능, 외부 서비스 설정 변경 없음. 관리 예비(새 범위용)는 0분이며 다른 결함이 보이면 아이디어로 남기고 이번 작업을 늘리지 않는다.
- 추정 방식 참고: [GAO Cost Estimating and Assessment Guide](https://www.gao.gov/products/gao-20-195g)의 범위·작업 분해·가정·위험·실측 갱신 원칙. 분 단위 수치는 정찰 판단이며 해당 문헌이 보증하지 않는다.

정찰 검증 한계

- 코드 편집/신규 테스트 작성 없이 기존 SQLite Relation suite만 실행했다. 0의 GET·감사 종단 결과, null 라우터 동작, PostgreSQL·전체 suite·vet는 구현자가 확인해야 한다.
- 이전 기록에 성공으로 남은 Liquid 수정은 현재 pinned main에 없다(`scripts/check-docs-liquid.sh` 없음, ci.yml 가드 없음, RELEASE_NOTES_v0.2.45.md:150 원문 유지). 이미 시도한 접근이므로 이번 과제로 재선택하지 않는다.
