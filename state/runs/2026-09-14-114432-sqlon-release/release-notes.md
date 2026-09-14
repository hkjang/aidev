SQLON v0.1.5 Release with a fix for PII exposure false positives on short column-name hints, GitHub Pages landing page improvements, and updated offline Docker image tarballs.

## Changes

- **PII 노출 리포트 오탐 수정** (`get_pii_exposure`): 짧은 ASCII 단서(`pan`·`dob`·`ssn`·`card` 등 네 글자 이하)를 부분 문자열이 아닌 낱말 경계로 대조합니다. `japan_code`가 신용카드로, `adobe_flag`가 생년월일로 잡히던 오탐이 사라지며, `pan`·`dob`·`ssn`·`card_no`·`cust_addr`·`iban`·`rrn`은 그대로 탐지됩니다.
- 실행 중 생기는 로컬 데이터(`data/backups`, `data/sqlon`)와 로컬 데모 스크립트를 `.gitignore`에 추가했습니다.
- GitHub Pages 랜딩 페이지: 한국어 기본·영어 전환, 브라우저 언어 자동 감지, SEO/AEO 메타데이터와 Schema.org JSON-LD, 모바일 반응형 레이아웃, GitHub Sponsor 버튼.

## Assets

| File | Description |
| --- | --- |
| `sqlon-v0.1.5-linux-amd64.tar.gz` | Linux amd64 binary + data/docs |
| `sqlon-v0.1.5-linux-arm64.tar.gz` | Linux arm64 binary + data/docs |
| `sqlon-v0.1.5-windows-amd64.zip` | Windows amd64 binary + data/docs |
| `sqlon-v0.1.5-docker.tar.gz` | Standard offline Docker image (`sqlon/sqlon:v0.1.5`) — PostgreSQL / MySQL / MariaDB |
| `sqlon-oracle-v0.1.5-docker.tar.gz` | Oracle offline Docker image (`sqlon/sqlon-oracle:v0.1.5`) — Oracle Instant Client bundled |
| `SHA256SUMS.txt` | SHA-256 checksums for all assets |

```sh
docker load -i sqlon-v0.1.5-docker.tar.gz
docker run --rm -p 6767:6767 -e SQLON_ADMIN_TOKEN=change-me sqlon/sqlon:v0.1.5
```
