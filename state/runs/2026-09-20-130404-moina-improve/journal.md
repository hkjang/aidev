# 회차 노트 2026-09-20-130404-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:04] base pinned — main@a5ca9bc
- [러너 13:04] autonomy release — 
## 정찰 노트
- 고른 이유: main v0.1.33에 mail 캠페인(014)이 아직 없어 메일·가이드 재캡처 후보는 막혀 있고, image-orientation은 브라우저 기본값과 같아 효과 없는 변경(운영자 규칙)이라 기각. `safeFilename` MIME 불일치는 실제 응답(`filename`·`Content-Disposition`)이 바뀌고 S 크기에 실제 배선 integration 테스트로 증명 가능해 선택. Makefile `-race`는 한 줄이라 차선으로.
- 확신 없는 곳: `frontend/API_CONTRACT.md`에 Media.filename 설명이 있는지 미확인(있으면 OpenAPI와 같은 문장으로). `safeFilename` 호출 자리는 social.go:655 하나뿐으로 보이나 구현자가 grep으로 재확인할 것. 확장자 표(jpg/png/gif/webp/mp4/webm)는 내가 정한 것이며 저장소 관례는 없음.
- 조심할 것: 200 rune 절단을 확장자 붙이기 전 stem에 적용해야 절단 뒤에도 확장자가 남음. 기존 행은 고치지 말 것(migrations 추가만). integration 테스트는 대역 mediastore 없이 `New(repository…)`+세션 cookie+CSRF로 실제 `POST /api/v1/media`를 지날 것(다른 media integration 테스트는 DB 직접 INSERT라 이 경로의 첫 테스트가 됨). 프로필(2026-09-19)은 하루 전이고 코드와 어긋난 곳 없어 다시 쓰지 않음.
- [러너 13:07] scout done — 업로드 미디어의 저장 파일 이름 확장자를 판정한 MIME에 맞추고, 이름이 비면 종류에 맞는 기본 이름을 준
## 구현 노트
- 무엇·왜: `safeFilename(header, mimeType)`로 바꿔 저장·다운로드 이름의 확장자를 서버가 판정한 MIME(6개, `mediaFilenameExtensions`)에 맞췄고 빈 이름·`.`·`..`은 `image.jpg`/`video.mp4`처럼 종류+확장자로 짓습니다(commit 09cbdec). 부수 발견: `filepath.Base("")`가 `"."`라 기존 `"image"` 기본값은 닿을 수 없던 코드였습니다.
- 확신 없는 곳·검증 못 한 것: 과제서 (b)의 `filename=""` 케이스는 Go net/http 파서가 빈 filename part를 값으로 읽어 400 `file_required`가 되므로 end-to-end로는 도달 불가 — integration은 `filename=".."`로 대신했고 `""`은 단위 테스트만 덮습니다(Go 1.26 `mime/multipart/formdata.go:142`에서 확인). 확장자 표는 정찰이 정한 6개 그대로이며 `.m4v`·`.jpe` 같은 별칭은 인정하지 않아 `clip.m4v`(MP4)는 `clip.mp4`로 바뀝니다 — 의도한 동작이지만 저장소 관례는 없습니다. 프런트·e2e는 응답 형식이 문자열 그대로라 돌리지 않았습니다.
- 일부러 하지 않은 것: 기존 `media_assets.filename` 행 재작성(migrations 추가만·가치 없음, 새 업로드부터 적용), `detectMediaType`·`contentDisposition` 변경, `frontend/API_CONTRACT.md`(filename 언급 0건이라 두지 않음), 차선 후보 Makefile `-race`(본 과제가 끝나 범위 밖으로 둠).
- 다음 역할이 조심할 것: `TestPostgreSQLUploadMediaFilenameFollowsDetectedMIME`는 `MOINA_TEST_POSTGRES_DSN`이 있어야 돌고 없으면 skip — 이 환경은 55432·55433이 다른 세션 컨테이너에 잡혀 있어 `-p 127.0.0.1::5432`로 자동 포트를 받아 돌렸습니다(컨테이너는 지웠음). 검증 명령: `cd backend && MOINA_TEST_POSTGRES_DSN=… go test -race ./... -count=1`(FAIL 0·SKIP 0), `go vet`·staticcheck 2025.1.1, `make fmt`·`make check`(route 120개) 모두 통과.
- [러너 13:13] brief accepted — 채택 — 과제서의 근거(social.go:732·호출 자리 하나·`filepath.Base` 결과)가 코드와 맞았고 수용 기준 1·2·3(a)·(b)를 그대로 �
- [러너 13:13] verify passed — 검증 7개 통과 (auto)
## 비평 노트
- 확인한 것: diff(a5ca9bc..09cbdec) 4개 파일 전부 읽음. `safeFilename` 호출 자리는 social.go:655 하나뿐이고 `media.Put`도 uploadMedia 한 곳뿐임을 grep으로 재확인. 경계값(`.`·`..`·`...`·`.png`만·공백·역슬래시·200 rune 절단·대소문자 확장자) 손으로 따라가 결함 없음. 단위 테스트는 시그니처가 바뀌어 옛 코드에서 컴파일조차 안 되고 `photo.png`→`photo.jpg` 단언이 있어 진짜 검증함.
- 직접 돌린 것: 단위 테스트·`go vet` 통과, throwaway `postgres:17-alpine`(자동 포트, 컨테이너 삭제)으로 `TestPostgreSQLUploadMediaFilenameFollowsDetectedMIME`을 `-race`로 돌려 3 케이스 PASS. 응답 filename·DB 행·Content-Disposition·Content-Type 모두 단언됨.
- 못 본 것: `make check`·frontend vitest·e2e는 돌리지 않음(프런트는 filename을 표시 fallback으로만 쓰고 e2e에 filename 단언 없음을 grep으로 확인). 보안·법무 관점: 새 수집·전송 없음, 사용자 이름 입력의 폴리글롯(x.html 저장) 표면이 오히려 줄어듦 — 차단 사유 없음.
- 승인이어도 남는 우려: (1) 허용 MIME 목록이 social.go:642 인라인 슬라이스와 `mediaFilenameExtensions` 맵 두 곳에 있어 형식을 추가할 때 어긋날 수 있음(맵에 없으면 확장자 없이 저장). (2) `.hidden`처럼 점으로 시작하는 이름은 stem이 비어 `image.jpg`로 바뀜(정보 손실이지만 무해). (3) 릴리즈 노트: 새 업로드부터 `clip.mov`→`clip.mp4`, `.m4v`·`.jpe` 별칭은 교체됨; 기존 행은 그대로.
- [러너 13:15] review approved — 리뷰 승인 (risk=low)
- [러너 13:15] pr created — https://github.com/hkjang/moina/pull/27
- [러너 13:23] ci passed — 검사 2개 모두 success
- [러너 13:23] merge done — 09cbdec
- [러너 13:36] release published — v0.1.34
- [러너 13:41] assets verified — v0.1.34 자산 1개 (이전 v0.1.33: 1)
