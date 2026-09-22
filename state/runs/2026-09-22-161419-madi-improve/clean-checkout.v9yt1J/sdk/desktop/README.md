# madi 데스크톱

React 로컬 화면과 Tauri 2 네이티브 셸입니다. 서비스 서버는 기존 Go + React 이미지를 그대로 사용합니다. 로컬 화면은 빠른 기록·선택 파일 가져오기/내보내기·암호화 오프라인 보관함을 제공하고, 원격 워크스페이스는 기기 IPC 권한이 없는 별도 창에서 엽니다.

## 빌드와 폐쇄망 배포

Node.js 22+, Rust 1.88+와 [Tauri 운영체제별 필수 도구](https://v2.tauri.app/start/prerequisites/)가 필요합니다. Linux는 WebKitGTK 4.1, GTK3, Ayatana AppIndicator, DBus 개발 패키지를 사용합니다. Windows는 WebView2, macOS는 운영체제 WKWebView를 사용합니다.

```sh
cd sdk/desktop
npm ci
npm run build
npm run tauri build
```

인터넷이 연결된 빌드 환경에서 만든 해당 운영체제 설치물을 폐쇄망으로 반입합니다. 앱 실행 시 npm·Cargo·CDN에 연결하지 않습니다. Windows 폐쇄망 PC에는 조직이 승인한 WebView2 오프라인 런타임을 먼저 준비해야 합니다. macOS/Windows 일반 배포는 조직 인증서로 코드 서명하는 것을 권장합니다. 서명 키를 저장소에 넣지 마세요.

완전히 오프라인에서 소스부터 빌드하려면 npm 패키지 캐시와 `cargo vendor --locked` 결과, 해당 OS 개발 도구를 별도로 준비하고 `npm ci --offline`, Cargo의 vendored source 설정과 `--offline --locked`를 사용합니다. 설치 환경에 없는 시스템 라이브러리를 앱이 인터넷에서 자동 다운로드하지 않습니다.

GitHub 릴리즈 첨부물은 사용자 배포 정책에 따라 **madi 서비스 Docker 이미지 tar.gz만** 사용합니다. 데스크톱 설치물과 클리퍼는 이 SDK에서 조직별로 빌드·서명·배포하며 서비스 릴리즈에 자동 첨부하지 않습니다.

## 서버 연결

1. 로컬 ‘연결과 기기 설정’에 `https://madi.company.local`처럼 경로 없는 서버 주소를 입력합니다.
2. 서비스 개인화 → API 키에서 대상 워크스페이스의 `document:read`, `document:write` 키를 발급해 입력합니다. 비밀번호나 Bootstrap admin 환경변수를 클라이언트에 넣지 않습니다.
3. 기본값은 메모리 세션 키입니다. ‘운영체제 키체인에 API 키 보관’을 명시적으로 선택하면 OS 자격 증명 저장소에 보관합니다. 저장소가 잠겼거나 없으면 오류를 표시하고 평문 파일로 대체하지 않습니다.
4. 서비스 화면 열기에서 일반 서비스 계정 또는 관리자 설정 SSO로 로그인합니다. API 키는 원격 웹뷰에 넘기지 않습니다. HTTPS SSO 리다이렉트를 허용하며 창 제목에 실제 현재 원점을 표시합니다. HTTP SSO는 사내 HTTP 위험 확인이 활성화된 경우에만 허용합니다.

사내 CA는 운영체제 신뢰 저장소에 등록하세요. TLS 검증 비활성화 옵션은 없습니다. 키 회전·권한 축소·만료 후에는 서비스 개인화에서 새 키를 발급하고 기기 설정에서 다시 연결하세요. 연결 해제는 메모리 키와 저장한 키체인 항목을 지우고 원격 창을 닫습니다. OS 키체인 삭제 실패는 따로 안내합니다.

## 빠른 기록·파일·딥 링크

- `Ctrl/⌘ + Shift + Space`: 로컬 빠른 기록 창 표시. 다른 앱의 단축키 충돌·OS 권한 때문에 등록이 실패하면 안내합니다.
- 트레이: 빠른 기록, 워크스페이스 열기, 종료. 트레이 생성 성공 시 닫기 버튼은 창을 숨깁니다. 트레이를 사용할 수 없으면 일반 창 닫기로 종료합니다.
- 가져오기: OS 파일 선택 대화상자로 지정한 UTF-8 Markdown/TXT 파일 하나, 최대 4 MB. 심볼릭 링크·디렉터리는 거부합니다. HTML·스크립트를 실행하지 않습니다.
- 내보내기: OS 저장 대화상자에서 사용자가 지정한 파일만 저장합니다. 기존 파일 교체 여부는 저장 대화상자에서 확인하세요.
- `madi://capture`, `madi://page/문서UUID`: 설치 프로그램이 등록한 딥 링크. 앱에서 확인 후 설정된 사내 서버만 엽니다. 링크로 서버 설정·키를 바꾸거나 명령을 실행할 수 없습니다. [Tauri 딥 링크 설치 조건](https://v2.tauri.app/plugin/deep-linking/).

빠른 기록은 개인 인박스로 전송합니다. 네트워크 오류 시 같은 내용은 같은 `client_request_id`로 재시도해 중복 생성을 방지합니다. 내용을 바꾸면 새 요청 ID를 만듭니다. 파일을 가져왔더라도 ‘저장’ 전에는 서버에 보내지 않습니다.

## 오프라인 보관함

기기 보관함 암호는 12자 이상이며 서버 로그인 암호와 다르게 정하세요. AES-256-GCM, PBKDF2-SHA256 310,000회, 무작위 salt/IV를 사용합니다. 암호화 키는 메모리에만 있고 5분 미사용·명시적 잠금·앱 재실행 시 잠깁니다. 보관함은 서버 주소와 사용자 ID에 연결됩니다. 암호를 잃으면 복구할 수 없습니다.

문서별 최대 1 MB, 문서와 임시 기록 합계 100개·50 MB까지 직접 선택해 보관할 수 있습니다. 서버 API·첨부파일·SSO 세션은 자동 캐시하지 않습니다. 오프라인 편집은 기기 초안이며 서버에 자동 반영하지 않습니다. 전송할 때 현재 키·사용자 ACL과 원본 버전을 다시 확인합니다. 서버가 바뀌었으면 덮어쓰지 않고 초안을 보존하므로 내보내 비교하세요.

오프라인 기기 사본은 이후 권한 회수를 실시간으로 알 수 없습니다. 기밀 문서는 조직의 반출 정책을 확인한 후 보관하세요. 기기 보관함 삭제는 로컬 사본과 미전송 기록을 영구 삭제하지만 서버 원본을 바꾸지 않습니다.

## 검증

```sh
cargo test --manifest-path sdk/desktop/src-tauri/Cargo.toml --no-default-features
cd sdk/desktop && npm run build
```

`tests/desktop.mjs`는 실제 Tauri 바이너리·WebKit WebDriver를 사용해 시작 딥 링크의 명시적 확인/취소, 로컬 화면, 인증 API, 개인 인박스 전송, OS 파일 대화상자의 UTF-8 가져오기/내보내기, WebCrypto 보관함, 전역 빠른 기록 단축키, 원격 창의 IPC 거부와 연결 해제를 검증합니다. 실행 전 별도의 폐기 가능한 madi QA 서버와 `MADI_DESKTOP_BINARY`, `MADI_TAURI_DRIVER`, Linux의 `MADI_WEBKIT_DRIVER`를 지정하세요. 파일/전역 키 검증에는 Xvfb와 xdotool(`MADI_XDOTOOL`로 경로 지정)이 필요합니다. Linux 네이티브 실행을 검증했으며 다른 OS 설치물의 실행·서명 검증은 해당 OS에서 별도로 수행하세요. [Tauri WebDriver](https://v2.tauri.app/develop/tests/webdriver/).

관리자 권한이 없는 Linux 빌드 환경에는 `tests/native-deps.mjs`, `tests/native-fill-libs.mjs`, `tests/native-run.mjs`가 있습니다. 공식 패키지를 사용자 임시 디렉터리에 추출하고 private mount namespace에서만 라이브러리 경로를 구성합니다. 호스트 운영체제에 패키지를 설치하지 않습니다. 이는 개발 검증 도구이며 운영 배포 방식이 아닙니다.

## Rust 의존성 보안 점검

2026-09-08에 `cargo-audit 0.22.2`로 `src-tauri/Cargo.lock`의 570개 패키지를 대상 OS 필터·권고 제외 없이 검사했습니다. 결과는 RustSec 분류상 **취약점 0건, 정보형 경고 17건**입니다. 경고가 없다는 뜻은 아닙니다. glib 0.18.5의 안전성(`unsound`) 경고 1건과 GTK3 계열 10개·proc-macro-error 1개·Unicode 계열 5개의 유지보수 중단 경고가 남아 있습니다.

[RUSTSEC-2024-0429](https://rustsec.org/advisories/RUSTSEC-2024-0429.html)는 `VariantStrIter`의 C 출력 포인터 처리로 인해 최적화 빌드에서 정의되지 않은 동작·충돌이 발생할 수 있다는 경고입니다. SDK 및 내려받은 Rust 의존성 소스에서 해당 반복자와 `array_iter_str` 호출은 glib 자체 정의·예제·테스트 외에는 발견되지 않았습니다. 원격 서비스 창에는 기기 IPC 권한이 없으며 로컬 명령은 창·원점과 API 경로를 확인합니다. 다만 이 정적 문자열 검사는 전체 프로그램의 비도달성 증명이 아닙니다.

공식 수정 버전은 glib 0.20.0 이상이지만 현재 GTK3 의존성은 0.18 계열을 요구합니다. 확인 당시 crates.io의 마지막 호환 버전은 0.18.5이고 공식 0.18 브랜치에도 해당 수정이 없어 단순 버전 갱신으로 해결할 수 없습니다. [GTK3 바인딩 유지보수 중단](https://rustsec.org/advisories/RUSTSEC-2024-0415.html)에도 수정 버전은 없습니다. 조직별 데스크톱 배포에서는 이 잔여 경고를 검토하고 Tauri/Wry의 전환 상황, 운영체제 GLib·GTK·WebKitGTK 보안 업데이트를 계속 확인하세요. Cargo 검사는 운영체제 공유 라이브러리를 검사하지 않습니다.

저장소 루트에서 연결된 검증 환경에 설치한 공식 도구로 다시 검사할 수 있습니다.

```sh
cargo audit --file sdk/desktop/src-tauri/Cargo.lock
cargo audit --file sdk/desktop/src-tauri/Cargo.lock --deny unsound
```

두 번째 명령은 현재 안전성 경고를 실패로 처리합니다. 검사 때문에 제품 소스·잠금 파일을 수정하거나 권고를 무시하도록 설정하지 않았습니다. 실제 원문은 `test-results/desktop-security/cargo-audit.json`, 엄격 검사 원문은 `cargo-audit-strict.json`, 도구·DB 커밋·잠금 파일 해시는 같은 폴더의 `provenance.json`에 보관합니다. 상세 재현 방법은 [보안 점검 문서](../../docs/security-verification.md#데스크톱-rust-의존성-별도-검사)를 참고하세요. 이 Rust SDK와 네이티브 실행물은 서비스 Docker 런타임에 포함되지 않습니다.
