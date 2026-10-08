# Invenqor Agent 문서

이 디렉터리는 Invenqor Server v0.2.45와 Agent v0.2.45의 역할별 공식 가이드를 제공합니다.

| 문서 | 대상 | Markdown | PDF |
|---|---|---|---|
| 사용자 가이드 | 설치 후 기본 사용과 상태 확인이 필요한 사용자 | [USER_GUIDE.md](USER_GUIDE.md) | [USER_GUIDE.pdf](USER_GUIDE.pdf) |
| 관리자 가이드 | 배포, 설정, 보안, 모니터링, 장애 대응 담당자 | [ADMIN_GUIDE.md](ADMIN_GUIDE.md) | [ADMIN_GUIDE.pdf](ADMIN_GUIDE.pdf) |
| 임원 보고서 | 도입·확대·통제 의사결정자 | [EXECUTIVE_REPORT.md](EXECUTIVE_REPORT.md) | [EXECUTIVE_REPORT.pdf](EXECUTIVE_REPORT.pdf) |
| Server 설치·오프라인·K8s 운영 | 플랫폼 운영자 | [SERVER_INSTALLATION.md](SERVER_INSTALLATION.md) | [SERVER_INSTALLATION.pdf](SERVER_INSTALLATION.pdf) |
| 자산 API·MCP·키 관리 | 연계·AI·보안 담당자 | [API_MCP_GUIDE.md](API_MCP_GUIDE.md) | [API_MCP_GUIDE.pdf](API_MCP_GUIDE.pdf) |

변경 요약과 검증 증적은 [Server·Agent v0.2.45 릴리즈 노트](RELEASE_NOTES_v0.2.45.md)를
참조하십시오.

문서의 기준 릴리즈는 Server `v0.2.45`, Agent `v0.2.45`, 기준일은
2026-08-24입니다. 제품 동작과
문서가 다를 경우 해당 버전의 소스 코드와 설정 검증 결과를 우선하며, 문서 오류는
저장소 이슈로 보고해 주십시오.

v0.2.45는 URL-only Agent 등록과 Windows 자동 업데이트를 강화하고, 사용자별
Windows package·병렬 RPM·stacked mount의 안정 식별을 보완합니다. 멀티 Pod에서는
Keycloak Secret을 공용 암호화 저장소로 이관하고 API Key 변경과 이벤트 최종 상태를
PostgreSQL compare-and-swap으로 보호합니다. MCP `2026-07-28`과 기존 client를 한
endpoint에서 지원하며, 상세 변경과 검증 증적은 릴리즈 노트를 참조하십시오.

임원 보고서·Server 설치 가이드·API·MCP 가이드의 PDF를 다시 생성하려면 저장소
루트에서 다음 명령을 실행합니다.

```bash
./scripts/build-docs.sh
```

빌드에는 Node.js 20 이상, `npx`, Chromium 계열 브라우저가 필요합니다.

사용자 가이드와 관리자 가이드는 화면 캡처를 싣고 공통 가이드 표준을 따르므로
PDF를 공용 도구로 만듭니다. 캡처를 다시 찍는 절차와 함께
[관리자 가이드 22장](ADMIN_GUIDE.md#22-가이드-화면-캡처-다시-만들기)에
정리했습니다.
