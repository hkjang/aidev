# 배포판에서 빠진 라이선스 원문

`scripts/licenses.mjs`는 설치된 npm 런타임 의존성과 실제 Go 서비스 import 모듈에서 라이선스·NOTICE를 모아 `web/public/licenses.txt`에 포함합니다. npm README에 포함된 완전한 라이선스도 보존합니다. 아래 두 원문은 설치 패키지에 파일이 없어 공식 저장소의 고정 버전에서 가져왔습니다.

- Go: https://github.com/golang/go/blob/go1.26.0/LICENSE
- react-remove-scroll-bar 2.3.8은 package.json과 README에 MIT를 선언하지만 배포본 및 당시 npm gitHead에 전체 LICENSE 파일은 없습니다. 같은 공식 프로젝트가 추가한 MIT 원문을 고정해 보존합니다: https://github.com/theKashey/react-remove-scroll-bar/blob/7301c160fda44cb8cf2b9fdfde61efad35736196/LICENSE

업스트림 텍스트를 변경하지 않습니다. 생성 manifest는 의존성 잠금 파일과 각 원문의 SHA-256을 기록합니다. 이 모음은 madi 자체의 소프트웨어 라이선스를 지정하지 않습니다.

PDF.js의 `cmaps`, `standard_fonts`, `wasm`에 동봉된 개별 라이선스는 실제 오프라인 정적 파일과 중앙 고지문 양쪽에 보존합니다. PDF.js가 선택적 Node 의존성으로 설치한 `@napi-rs/canvas-*` 플랫폼 패키지에 LICENSE가 없는 경우에는 같은 버전의 `@napi-rs/canvas`가 배포한 LICENSE를 사용합니다. 생성기는 부모의 `optionalDependencies`에 해당 플랫폼과 정확한 버전이 일치하고 선언 라이선스도 같은지 검사하며 다른 패키지의 라이선스를 임의로 대입하지 않습니다. Node의 네이티브 `.node` 바이너리는 최종 Go 서비스 이미지의 웹 정적 파일에 포함하지 않습니다.

PDF.js의 Liberation Sans는 2.x OFL 폰트가 아니라 **1.07.4 GPLv2 + Liberation 예외**입니다. [PDF.js 6.3.289의 공식 원본 기록](https://github.com/mozilla/pdf.js/blob/v6.3.289/external/standard_fonts/README.md)을 기준으로, 별도 수집기 `scripts/bundle-pdf-font-sources.mjs`가 [공식 1.07.4 SFD 소스](https://releases.pagure.org/liberation-fonts/liberation-fonts-1.07.4.tar.gz), Makefile·FontForge 스크립트·라이선스, 실제 포함된 네 폰트와 SHA-256/버전 기록을 같은 이미지의 `/usr/share/madi/sources/pdfjs-liberation`에 넣습니다. APK 원본 목록만으로 이 웹 폰트를 포함했다고 간주하지 않습니다. 폰트/소스/기록 해시가 바뀌면 빌드를 중단해 재검토를 요구합니다. 폰트는 PDF.js 배포본 그대로 유지하며 역사적 빌드와의 바이트 단위 재현성을 주장하지 않습니다.

고지문 안의 원본 공백·탭은 저작권자가 제공한 텍스트의 일부로 유지합니다.
