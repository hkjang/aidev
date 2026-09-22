# madi 웹 클리퍼

Chrome/Edge Manifest V3와 Firefox용 확장 프로그램입니다. 별도 온라인 서비스 없이 사용자가 지정한 madi 서버와만 통신합니다. 개인 API 키는 `storage.session`에 보관하며 브라우저 종료 시 삭제됩니다. 전송 전 미리보기를 제공합니다.

## 오프라인 설치

인터넷 연결이 가능한 빌드 환경에서 `node sdk/clipper/build.mjs chrome` 또는 `node sdk/clipper/build.mjs firefox`를 실행합니다. 외부 패키지 설치는 필요하지 않습니다. 생성된 `sdk/clipper/dist/{chrome,firefox}` 디렉터리를 폐쇄망에 복사하세요.

Chrome/Edge: 확장 프로그램 관리 → 개발자 모드 → 압축해제된 확장 프로그램 로드. 조직에서는 브라우저 정책에 맞춰 사내 확장 프로그램 배포 체계를 사용하세요.

Firefox: 개발 검증은 `about:debugging` → 임시 부가 기능 로드 → manifest.json. 일반 배포에는 조직 정책에 맞는 Firefox 확장 서명이 필요합니다. 이 저장소는 스토어 업로드나 서명을 자동 수행하지 않습니다.

## 연결과 사용

1. madi의 개인화 → API 키에서 해당 워크스페이스의 `document:read`, `document:write` 권한 키를 발급합니다.
2. 클리퍼 연결 설정에 서버 원점 주소와 키를 입력합니다. 해당 원점의 접근 권한만 명시적으로 허용합니다.
3. 기본 워크스페이스를 선택합니다.
4. 웹페이지에서 확장 아이콘을 눌러 본문(article/main), 선택 텍스트, 링크, 전체 페이지 텍스트, 현재 보이는 화면의 스크린샷 중 하나를 선택합니다.
5. 제목·본문·이미지를 확인하고 개인 인박스에 저장합니다. 스크린샷은 전체 스크롤 페이지가 아니라 현재 뷰포트입니다. 캡처한 HTML과 스크립트는 실행·전송하지 않습니다.

실패한 전송은 세션 미리보기에 남고 같은 `client_request_id`로 재시도합니다. 이미 저장된 문서에 첨부만 실패한 경우 문서를 중복 생성하지 않습니다. 브라우저를 종료하면 미전송 미리보기는 사라지므로 먼저 저장하거나 직접 복사하세요. 개인 인박스에 저장 후에는 원본 문서에서 분류·공유합니다.

## 보안 범위

설치 시 `activeTab`, `scripting`, `storage`만 요청합니다. `optional_host_permissions`는 사용자가 선택한 madi 서버 원점에 대해서만 요청합니다. 키·본문은 쿠키를 포함하지 않는 인증 API 요청으로 전송하며 리다이렉트를 따르지 않습니다. TLS 검증을 끄는 옵션은 없습니다. 사내 HTTP는 위험 확인을 거쳐 사용할 수 있으나 HTTPS를 권장합니다. 원본 사이트의 외부 이미지 URL을 서버가 대신 가져오지 않습니다.

확장 프로그램 페이지는 로컬 코드만 실행하며 원격 스크립트·iframe·eval을 사용하지 않습니다. 캡처 세션에는 민감한 정보가 포함될 수 있으므로 공용 PC에서는 연결 해제 후 브라우저를 종료하세요.

API: `POST /api/v1/captures`, `GET /api/v1/workspaces`, `POST /api/v1/attachments?document_id=...&client_request_id=...`.

참고: [Chrome activeTab](https://developer.chrome.com/docs/extensions/develop/concepts/activeTab), [Chrome storage.session](https://developer.chrome.com/docs/extensions/reference/api/storage), [Firefox MV3 background](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/manifest.json/background).
