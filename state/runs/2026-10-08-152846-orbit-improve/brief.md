- 과제: 전체 내보내기의 세션 전용 인증과 완결 판정을 OpenAPI·API 문서에 맞추기 (가치 3 / 위험 1 / 작업량 S)
- 왜: 실제 GET /api/v1/personal/export는 세션 쿠키만 허용하지만 openapi.go는 people:read Bearer 키로 호출할 수 있다고 안내하며, docs/API.md에는 내보내기 설명 자체가 없다. 이 한 엔드포인트의 문서를 실제 동작에 맞추면 클라이언트가 잘못된 인증으로 403을 받거나 HTTP 200인 부분 실패를 온전한 백업으로 오인하는 일을 줄일 수 있다.
- 수용 기준:
  1) GET /openapi.json의 components.securitySchemes에 sessionCookie(type=apiKey, in=cookie, name=orbit_session)를 추가하고, /personal/export의 get.security만 [{"sessionCookie":[]}]로 명시한다. 해당 operation 설명은 세션 전용이며 API 키로 호출할 수 없음을 알리고 people:read를 요구한다는 문구를 제거한다. 최상위 bearerAuth 보안 요구와 다른 엔드포인트는 유지한다. security:[] 또는 빈 객체는 인증 불필요로 읽히므로 쓰지 않는다.
  2) /personal/export 응답 설명 및 docs/API.md의 새 절에 HTTP 200만으로 성공을 판단할 수 없고, JSON 파싱 성공 + complete === true가 필요하다고 쓴다. 정상은 complete:true, 섹션 처리 실패는 complete:false와 failed_section(people/interactions/memories/links 중 하나)이다. 실패한 배열은 일부 또는 빈 배열로 남고 이후 섹션 키는 없으며, 네트워크/쓰기 실패는 유효한 JSON이나 실패 마커조차 보장하지 않는다. 기존 서버 동작을 고치려 하지 않는다.
  3) docs/API.md의 영어 문체를 유지한다. 첫 Bearer 안내가 모든 엔드포인트에 해당하는 것처럼 읽히지 않도록 이 세션 전용 예외를 안내한다. 전체 경로 GET /api/v1/personal/export, 로그인한 브라우저의 Personalization 내보내기 링크, attachment 파일명 orbit-export-YYYY-MM-DD.json, 최상위 exported_at/service_version/user 및 네 배열을 설명한다. 정상/people 실패 JSON 예시 두 개는 주석·말줄임표 없는 파싱 가능한 형태로 적는다. 빈 배열 예시로 충분하며 user 객체는 exportData가 실제 내보내는 여섯 필드를 따른다. Bearer curl 예시를 추가하지 않는다.
  4) 신규 TestOpenAPIExportContract는 New(nil, "test-version", "test", "test")로 만든 실제 라우터에 httptest 요청 GET /openapi.json을 보내 200·JSON Content-Type·파싱·info.version을 확인한다. 위 쿠키 스키마와 export operation의 보안 override, 최상위 bearerAuth 유지, /people/의 기존 people:read 설명 유지도 응답 JSON에서 검증한다. complete/failed_section 설명이 실제 공개 응답에 있는지도 검사한다. 인증 계약 기준은 requiredScope(http.MethodGet, "/api/v1/personal/export") == "session-only"로 연결하되 이것이 실제 인증 통합 시험은 아님을 구분한다. 소스 파일을 읽는 문자열 시험이나 자체 제작 OpenAPI 객체를 주입하는 시험은 금지한다.
- 건드릴 파일:
  - internal/server/openapi.go:openAPI — 쿠키 security scheme와 export 전용 operation 리터럴만 추가/수정. /orbit 선례처럼 한 항목을 펼치고 operation(summary, scope)는 그대로 둔다. 기존 '/orbit만 리터럴' 주석은 두 예외가 된 사실에 맞게 짧게 고친다.
  - internal/server/openapi_test.go:TestOpenAPIExportContract — 신규, 위 실제 HTTP 라우터 응답 회귀 시험. 프로덕션 코드 파일은 총 1개.
  - docs/API.md — Bearer 설명의 예외 안내 및 전체 내보내기 절/예시 추가.
- 검증 명령:
  - go test -race -count=1 ./internal/server -run '^TestOpenAPIExportContract$' (추가할 시험 이름; 정찰 시점에는 아직 없음).
  - go test -race -count=1 ./... (정찰에서 실제 PASS, ORBIT_TEST_DATABASE_URL 미설정으로 기존 DB 시험은 SKIP).
  - go vet ./... ; go build ./... ; git diff --check (구현 후 수행; 정찰에서는 이 세 명령의 실행 결과 미확인).
  - 아래 Python 명령으로 docs/API.md의 모든 json 코드 블록이 파싱되는지 확인한다. 새 문서의 완결 계약은 export.go와 기존 TestExportBrokenSectionStaysParseable의 정상/실패 단언을 대조한다. 이 문서 변경에 DB 환경을 새로 설치할 필요는 없다. 정찰은 실 DB 응답을 이번 회차에 재실행하지 않았으므로 이를 실측했다고 기록하지 않는다.

```sh
python3 - <<'PY'
import json, pathlib, re
text = pathlib.Path('docs/API.md').read_text()
blocks = re.findall(r'```json\s*\n(.*?)\n```', text, re.S)
assert len(blocks) >= 2, '정상/부분 실패 JSON 예시가 필요합니다'
for block in blocks:
    json.loads(block)
print(f'{len(blocks)} JSON examples parsed')
PY
```

- 위험과 피할 것: auth.go/auth_test.go·session·personal.go·export.go·SQL·마이그레이션·.github/workflows는 수정하지 않는다. 실제 접근권한을 API 키에 열어 문서와 맞추지 않는다. OpenAPI 전 경로 확장·공통 operation 리팩터·응답 전체 JSON Schema 설계·내보내기 UI 변경은 범위 밖이다. 성공/실패 본문 예시는 사용자 개인정보가 아닌 가상의 값으로 쓴다. 감사 details에 본문을 넘기지 않는다. 이번 작업에서 쿠키 값·키를 수집할 이유가 없다.
- 차선 후보: docs/API.md만으로 세션 전용 인증·complete/failed_section 계약 문서화 — OpenAPI 표면을 바꿀 수 없는 구체적 제약이 발견될 때만 선택하고 그 이유를 기록한다. 이전부터 계속 보류된 문서 누락은 현재도 실재한다.

근거와 확인 범위:
- main@5c97a1a, VERSION 0.7.12. server.go:New는 /api/v1/personal/export에 exportData를, /openapi.json에 openAPI를 배선한다.
- auth.go:authenticate/requiredScope(176행)는 /api/v1/personal/를 session-only로 분류하고 API 키를 403 insufficient_scope로 거부한다. issueSession의 쿠키 이름은 orbit_session이다. web/src/pages/PersonalPage.tsx:271은 같은 출처의 anchor 링크로 세션을 전달한다.
- openapi.go:openAPI(5행)는 전역 bearerAuth 및 export의 operation("내 기록 전체 내보내기", "people:read")를 사용한다. 현재 잘못된 설명의 직접 근거다. openAPI 호출 시험은 rg로 찾지 못했다.
- export.go:exportData(48행 이후)는 메타데이터 뒤 people→interactions→memories→links 순서로 쓰며, 섹션 실패 분기는 닫기와 complete:false/failed_section을 내보낸다. export_db_test.go:TestExportBrokenSectionStaysParseable가 이미 정상·실패 모양을 검증한다. 이번에는 테스트 소스만 읽었고 DB 실행은 미확인이다.
- OpenAPI 3.1의 operation.security override 및 cookie apiKey 규칙: https://spec.openapis.org/oas/v3.1.0.html#security-scheme-object 와 https://spec.openapis.org/oas/v3.1.0.html#operation-object . 공식 명세를 확인했다.

대안 비교와 결정:
- 선택: 공개 OpenAPI operation + Markdown을 함께 수정. 프로덕션 1파일, 시험 1파일, 문서 1파일로 실제 오안내를 닫는다.
- 문서만 수정: 15~25분, 변경 위험은 더 낮으나 도구가 읽는 /openapi.json은 잘못된 인증을 계속 안내한다. 차선으로만 둔다.
- 현상 유지: 구현 비용 0이나 403의 이유와 불완전 백업 판정법이 전달되지 않는다.
- OpenAPI 자동 생성·전 경로 규격화는 별도 설계가 필요해 이번 45분 범위 밖이며 선택하지 않는다.
- 핵심 가정: 세션 전용이라는 현행 제품 정책을 유지한다. 코드와 브라우저 링크가 같은 정책을 지지한다. New(nil, ...)로 openAPI를 부르는 새 시험의 실제 실행은 구현자가 확인해야 한다(라우트는 DB를 사용하지 않으므로 성립한다고 코드로 판단).

구현 순서와 점검 지점(아직 모두 pending; 사람 승인 대기 없음):
1. openapi.go의 한 operation과 schema를 수정하고 신규 라우터 시험을 함께 작성한다. 증명: 위 TestOpenAPIExportContract 명령. 다음 단계 전 구현자가 자체 점검한다.
2. docs/API.md의 설명·예시를 추가한다. 증명: 위 Python 파싱 명령 + exportData/기존 DB 시험 단언과 수동 대조. 문서 체크를 서버 동작 시험이라고 보고하지 않는다.
3. 전체 race 시험·vet·build·diff 검사를 수행하고 변경 파일이 위 세 개뿐인지 확인한다. 새로운 결함을 발견하면 과제서를 정정하고 별도 아이디어로 남기며 범위를 늘리지 않는다.

추정 근거:
- bottom-up: operation/schema 5~8분 + 실제 HTTP 시험 8~12분 + Markdown/예시 6~9분 + 전체 검사·검토 4~6분 = 기본 23~35분. 정찰의 기존 Go race 시험은 빠르게 통과했고 의존성 추가나 DB 작업은 없다.
- 알려진 불확실성(응답 JSON 타입 단언/문구 정합성 보완) 예비시간 5~8분을 별도로 두어 총 28~43분. 이는 통계적 신뢰구간이 아닌 정찰의 중간 확신 계획 범위이며 45분 내 완료 보장은 아니다.
- 과거 회차에 단일 호출자 수정·회귀 시험이 채택된 기록과 규모를 비교했으나 실제 소요시간 자료가 없어 독립적인 유사 추정 수치는 만들지 않았다. 미지의 범위 증가를 위한 관리 예비분은 0분이며, 생기면 별도 회차로 넘긴다.
- 적용 스킬: pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration. 전용 Skill 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills 아래 해당 SKILL.md를 직접 읽었다. 추정은 스킬의 작업 분해·가정·범위·예비분 분리 원칙을 사용하며 외부 비용 통계나 확률 수치를 인용하지 않는다.
