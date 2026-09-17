# PR #3 fix summary (commit e772d79)

- 문제 1: settings.html 의 bool 컨트롤이 미설정 tracking_momento_proxy 를 체크 해제로 그리고 save() 가 모든 체크박스를 'true'/'false' 로 보내, 마스터 토큰만 저장해도 proxy='false' 가 영속화됨 → 서버는 빈값=켜짐이라 문서가 약속한 기본 동작에 화면으로 도달 불가.
- 고침 1: SettingDef 에 `Default`("true" for momento_proxy) 를 두고 /api/settings 뷰로 노출, 화면은 미설정 bool 을 서버가 읽는 대로(default true = 명시적 부정만 꺼짐) 그리며 체크박스는 그려진 상태(data-initial)에서 실제로 바뀐 것만 보냄.
- 문제 2: PUT /api/settings 에서 per-key Validate(validateOptionalURL)·알 수 없는 키 검사가 저장 루프 안에서 실행되어 `{"tracking_enabled":"false","tracking_matomo_url":"ftp://x"}` 가 map 순서에 따라 일부만 저장됨(테스트로 재현 확인).
- 고침 2: meta.Service.CheckSetting(키·형식 검증만) 을 추가하고 저장 루프 앞에서 모든 키를 먼저 검사 → 하나라도 실패하면 아무것도 저장하지 않음. 옛 코드에서 실패하고 새 코드에서 통과하는 회귀 테스트(16회 반복, 잘못된 URL·알 수 없는 키 두 경우 + 뷰의 default 노출) 추가.
- 검증: `go build ./...`, `go vet ./internal/...`, `go test ./... -count=1` 전부 통과(CI 워크플로 파일은 저장소에 없음).
