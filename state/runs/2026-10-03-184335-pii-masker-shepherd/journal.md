# PR 처리기 노트 2026-10-03-184335-pii-masker-shepherd — pii-masker PR #31
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-03-181743-pii-masker-improve)
# 회차 노트 2026-10-03-181743-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:17] base pinned — main@5b17cd6
- [러너 18:17] autonomy release — 
- [러너 18:22] scout done — 마스킹된 이미지의 실제 인코딩을 응답이 광고하는 MIME 타입과 일치시키기 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 바꾼 것: `internal/masking/engine.go`의 인코더 선택 switch만 — 선언 `mimeType`(png → jpeg/jpg)을 먼저 보고, 디코딩된 `format == "png"` 분기는 선언이 없는 호출자용 fallback으로 남겼다. 응답의 `output.mime_type`·결과 다운로드 `Content-Type`(nosniff)·파일명 확장자가 모두 같은 선언값에서 나오므로 선언이 이겨야 한다. 프로덕션 파일 1개 + README 한 문장 + 테스트 2개.
- 확신 없는 곳: (1) HTTP 회귀는 `startAppServerWithUpstream`를 쓰므로 `httptest.NewServer(application.Handler())`를 지나고 `app.Serve`+직접 만든 리스너는 지나지 않는다(이 파일의 기존 관례). (2) 선언 MIME이 png/jpeg 이외(bmp/tiff 등)로 `MaskImageFile`에 도달하는 경로는 여전히 **미확인** — `validateAttachment`의 `ValidateMIMEType`이 막는다고 보고 `format` fallback을 그대로 남겼을 뿐 실측하지 않았다. (3) JPEG 재인코딩이 마스킹 품질(검은 박스 경계의 블로킹)에 주는 영향은 보지 않았다 — 기존 선언 jpeg + 내용 jpeg 경로와 같은 Quality 95라 동등하다고 판단만 했다.
- 일부러 하지 않은 것: `internal/mock`의 `isMockSupportedMIMEType`(별 과제로 ideas.json에 적음), `Output.MIMEType`/`MaskedFilename`을 실제 포맷에 맞추는 반대 방향 수정(service.go+internal/document까지 번지고 다운로드 파일명 계약이 바뀜), `buildMultipartBody` 헬퍼의 미사용 `contentType` 인자 수정(호출부 20여 곳의 declared MIME이 전부 바뀌므로 별 과제).
- 다음 역할이 조심할 것: 새 HTTP 회귀는 내장 mock을 쓰지 않는다 — mock은 `image/jpeg`를 415로 거부하므로, 이 테스트를 `startAppServerWithConfig`로 옮기면 마스킹에 도달하지 못하고 실패한다. 테스트는 외부 네트워크·DB 없이 돈다.
- [러너 18:26] brief accepted — 채택 — 과제서의 근거(engine.go의 `format == "png" ||` 선행 조건과 service.go의 선언 MIME 그대로 대입)가 현재 HEAD와 정확히 일치
- [러너 18:26] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인함: 두 신규 테스트를 HEAD 복사본에서 engine.go만 main으로 되돌려 실제 실패를 재현(원장에 `실패 재현:` 줄이 없었다) — 증상이 과제와 일치. 미확신 (2)는 해소(SupportedMIMEs는 config.go:155 하드코딩, env 오버라이드 없음 → jpg·format fallback 분기는 도달 불가하지만 무해). 미확신 (3)은 프로브로 측정.
- 거절 사유 1건: README.md:17과 engine.go:500-501이 선언 MIME이 "파일 이름 확장자"까지 정한다고 하지만 MaskedFilename은 업로드 이름만 쓴다. 이름 `sample.png` + 선언 `image/jpeg` 업로드 → `masked_sample.png`에 JPEG 바이트(실측). 수리가 먼저 볼 파일: README.md:17, internal/masking/engine.go:498-502. 프로덕션 switch 로직은 옳으므로 문구만 고치면 된다.
- 승인 쪽 우려(릴리즈 노트): 폭 65535px 초과 PNG를 image/jpeg로 선언하면 이제 `jpeg: image is too large to encode`로 실패한다(변경 전 PNG로 성공). ValidateImageDimensions는 총 5천만 픽셀만 보므로 70000x100이 검증을 통과한다. 조용한 손상은 아니다.
- 못 본 것: Docker 빌드(Go 1.25.0), -race, 32비트/Windows, JPEG 재인코딩의 시각 품질.
- 보안·법무 차단 없음: 인가·비밀값·삭제 경로 미접촉, 새 PII 수집 없음, 검은 박스는 인코딩 전에 픽셀을 덮으므로 손실 압축으로 복원 불가.
- [러너 18:30] review rejected — 리뷰 거절: README.md:17 그리고 internal/masking/engine.go:500-501 — 둘 다 선언 MIME이 마스킹 결과 '파일 이름 확장자'까지 정한다고 주장하지만 코드는 그렇지 �
- [러너 18:30] pr created — https://github.com/hkjang/pii-masker/pull/31

## 수리 노트
- 지적은 전부 맞았다. 임시 프로브로 실측 재현: 이름 `sample.png` + 선언 `image/jpeg` + PNG 바이트 → `file_name="masked_sample.png" mime_type="image/jpeg" actual_format="jpeg"`. engine.go만 origin/main으로 되돌린 같은 프로브는 `actual_format="png"`였으므로, 이름 확장자와 바이트의 어긋남은 이번 변경이 새로 만든 것이라는 지적도 사실이다(틀린 지적 없음).
- 고친 방법: 문구만. README.md:17의 '·파일 이름 확장자' 절 제거 + `output.file_name`은 업로드 이름을 그대로 쓰므로 확장자가 선언 형식과 다를 수 있다고 명시. engine.go 주석의 "it also picks the masked file name's extension"도 실제 동작으로 교체. 프로덕션 switch와 신규 테스트 2개는 미수정. 프로브는 커밋하지 않고 삭제했다.
- 여전히 확신 없는 곳: (1) 이름 확장자와 선언 MIME의 불일치를 README에 '다를 수 있다'로 문서화만 했고 코드로 일치시키지는 않았다 — 비평가가 그쪽(MaskedFilename을 MIME에 맞추기)을 원하면 service.go+internal/document와 다운로드 파일명 계약까지 번지는 별 과제다. (2) 앞선 비평의 65535px 초과 PNG→jpeg 인코딩 실패 우려는 이번 수리 범위 밖이라 손대지 않았다. (3) Docker 빌드·-race·32비트/Windows는 이번에도 미확인.
- 검증: `go build ./...`, `go vet ./...`, `gofmt -l ./cmd ./internal`(무출력), `go test -count=1 ./...` 전 패키지 ok, `git diff --check` 통과. 커밋 dca7436 (README.md, internal/masking/engine.go 둘뿐), push 안 함.

## 심사 노트
- 확인한 것: 이전 거절 사유는 해소됐다 — 임시 프로브(POST /v1/jobs → GET result, 프로덕션 배선)로 sample.png + 선언 image/jpeg + PNG 바이트 → `file_name="masked_sample.png" mime_type="image/jpeg" 다운로드 Content-Type="image/jpeg"+nosniff 실제 바이트=jpeg`, README.md:17 문구와 정확히 일치. 프로브 삭제 후 트리 clean.
- 확인한 것: 두 신규 테스트는 engine.go만 origin/main으로 되돌리면 실제로 실패한다(engine_test.go:264 서브케이스, integration_test.go:1546). HEAD에서 `go test ./...`·`-race`(masking·httpapi)·build·vet·gofmt·diff --check 모두 통과.
- 확인한 것: 동작 변화는 '선언 jpeg + PNG 바이트' 한 분기뿐(나머지 세 분기 불변). SupportedMIMEs는 config.go:155 하드코딩·env 오버라이드 없음 + normalizeMIMEType의 jpg→jpeg 접기로 선언값은 png·jpeg뿐. 선언 MIME 우선은 upstage/client.go:336과 같은 관례. 보호 파일·라우트·권한·의존성 미접촉, 보안·법무 차단 없음.
- 못 본 것: Dockerfile Go 1.25.0 빌드, 32비트·Windows, JPEG 재인코딩의 시각 품질.
- 권고 근거: approve/merge/low — 65535px 초과 PNG를 jpeg로 선언하면 400으로 실패하지만(실측) 변경 전에도 그 응답은 nosniff로 렌더되지 않았고 조용한 손상 없이 큰 소리로 실패하며, 전체가 코드·문서뿐이라 revert로 완전히 되돌아온다. 릴리즈 노트에 65535px 건과 알파 소실 건을 적을 것.
