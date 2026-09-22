# 폐쇄망 설치·운영·복구 가이드

## 1. 반입 전 준비

madi 서비스 이미지에는 Go 실행 파일과 React 정적 자산이 포함됩니다. 실행 시 npm, Go 모듈, 외부 폰트 또는 CDN을 내려받지 않습니다. 서비스와 연결하는 PostgreSQL, Keycloak, AI API, 컨테이너 런타임 및 TLS 인증서는 해당 망에 별도로 준비합니다. Keycloak과 AI는 선택 사항입니다.

기본 릴리즈 아키텍처는 `linux/amd64`입니다. 다른 CPU 아키텍처는 해당 환경에서 이미지를 별도로 빌드하고 검증해야 합니다. 단일 서비스 복제본과 로컬 파일 볼륨을 기본으로 합니다. 대규모 동시 사용자 수나 수백만 문서 규모의 성능 보장은 부하 시험 전에는 제공하지 않습니다.

운영 이미지는 UID/GID `10001:10001`로 실행하는 Go 단일 바이너리와 React·한국어 폰트, CA 인증서, 시간대 데이터만 포함합니다. Git 동기화와 외부 실행기는 Go/HTTP 기반이므로 `git`, SSH 클라이언트, Node.js, Python 또는 컨테이너 내부 `pg_dump` 설치가 필요 없습니다. 네이티브 PostgreSQL 백업 스크립트의 클라이언트 도구는 **운영 호스트**에 준비합니다.

연결망에서 GitHub Release의 `madi-v0.3.0.tar.gz`를 내려받아 반입합니다. 배포 편의를 위해 저장소의 `compose.yaml`과 `.env.example`, 운영 가이드도 따로 반입합니다. 릴리즈 파일은 서비스 이미지 하나이며 PostgreSQL 이미지는 포함되지 않습니다. 전송 전후 SHA-256 값을 비교하여 파일 동일성을 확인하세요.

```sh
sha256sum madi-v0.3.0.tar.gz
gzip -t madi-v0.3.0.tar.gz
gzip -dc madi-v0.3.0.tar.gz | docker load
docker image inspect madi:v0.3.0
```

## 2. PostgreSQL 준비

서비스 전용 데이터베이스와 계정을 생성합니다. madi는 시작 시 스키마를 준비하므로 해당 데이터베이스에서 테이블·인덱스 생성 권한이 필요합니다. 조직의 백업·인증서·접근통제 기준을 적용하세요.

```sql
CREATE ROLE madi LOGIN PASSWORD '관리도구에서_생성한_안전한_비밀번호';
CREATE DATABASE madi OWNER madi;
```

실제 비밀번호를 SQL 이력에 남기지 않도록 조직의 비밀 관리 절차를 사용하세요. DSN은 `postgres://madi:암호@postgres.internal:5432/madi?sslmode=verify-full` 형태이며 사용자명과 비밀번호의 특수문자는 URL 인코딩이 필요합니다. 폐쇄망 안에서도 가능한 한 PostgreSQL TLS를 사용합니다.

## 3. 네 필수 환경변수

`.env.example`을 `.env`로 복사한 뒤 값을 입력합니다. `.env`에는 서비스 설정 네 개만 필요합니다.

```dotenv
POSTGRES_DSN=postgres://madi:ENCODED_PASSWORD@postgres.internal:5432/madi?sslmode=verify-full
BOOTSTRAP_ADMIN=admin@example.internal
BOOTSTRAP_ADMIN_PASSWORD=LONG_RANDOM_INITIAL_PASSWORD
ENCRYPTION_KEY=BASE64_ENCODED_32_RANDOM_BYTES
```

암호화 키 생성 예: `openssl rand -base64 32`. `.env`의 읽기 권한을 제한하고 별도 보안 저장소에 백업합니다. BOOTSTRAP_ADMIN_PASSWORD는 최초 계정 생성에 사용하며 재시작할 때 기존 비밀번호를 덮어쓰지 않습니다. ENCRYPTION_KEY를 단순히 교체하면 기존 암호화 비밀을 읽지 못합니다. 개인 API 키 회전과 서버 마스터 암호화 키 교체는 별개의 작업입니다.

## 4. 실행

```sh
docker compose up -d
docker compose ps
docker compose logs --tail=100 madi
curl --fail http://127.0.0.1:8080/readyz
```

Compose는 `pull_policy: never`를 사용하므로 외부 레지스트리에서 이미지를 당겨오지 않습니다. `/var/lib/madi` 볼륨은 유지합니다. 호스트 디렉터리를 직접 마운트하면 UID/GID 10001에 필요한 권한을 부여합니다. 데이터 볼륨을 삭제하는 `docker compose down -v`는 정상적인 업데이트 절차에 사용하지 마세요.

기본 읽기 전용 루트 파일시스템과 `no-new-privileges`, 모든 Linux capability 제거를 유지하세요. Compose `/tmp`는 최대 2GiB tmpfs로, 사용한 만큼 RAM/스왑을 소비합니다. 1GiB까지의 논리 복원 스테이징 및 동시 가져오기 작업을 고려해 호스트 여유 메모리를 확보하거나 `/tmp`를 전용 디스크 볼륨으로 교체하세요. Kubernetes 예제는 디스크 기반 2GiB `emptyDir`, 메모리 요청 512MiB/한도 4GiB, CPU 요청 100m/한도 2를 사용합니다. 처리량 보장값이 아닌 시작 설정이며 관리자 작업 동시성·첨부 크기와 함께 조정하고 OOM/디스크 사용량을 관찰합니다.

브라우저에서 로그인한 뒤 관리자 시스템 설정의 서비스 URL을 실제 HTTPS 주소로 지정합니다. 프록시에서 원래 호스트를 전달하고 SSE 버퍼링을 끄세요. Keycloak과 AI URL은 서비스가 접근할 수 있는 내부 주소로 설정합니다. 폐쇄망에서는 인터넷 호스트를 지정하지 않습니다.

## 5. 오프라인 동작 점검

브라우저 개발자 도구의 네트워크 탭에서 외부 CDN·폰트·분석 서비스 호출이 없는지 확인합니다. 외부 인터넷 연결을 끊은 상태에서 로그인, 문서 작성·저장·재조회, 검색, 파일 업로드·다운로드, 새로고침, 프로필과 관리자 설정을 확인하세요. SSO와 AI를 켠 경우 내부 서버만 호출하는지도 확인합니다.

소스 환경에서는 `bash scripts/verify-image.sh madi:v0.3.0`로 외부 egress가 차단된 Docker 네트워크 기동 검사를 할 수 있습니다. 이 스크립트는 테스트용 PostgreSQL 이미지를 먼저 가져오므로 연결망의 릴리즈 검증용입니다. 실제 반입망의 DNS, TLS, 프록시 및 볼륨 접근 검사는 운영 환경에서 추가로 수행해야 합니다.

## 6. 업데이트

1. 현재 버전과 데이터베이스·첨부파일을 백업하고 복원 가능 여부를 확인합니다.
2. 새 서비스 이미지 아카이브를 반입하고 `docker load`합니다.
3. `compose.yaml`의 image를 새 `madi:v버전`으로 변경합니다.
4. `docker compose up -d` 후 준비 상태와 주요 화면을 확인합니다.

마이그레이션 후에는 이전 이미지로만 되돌리면 데이터 형식이 맞지 않을 수 있습니다. 문제가 있으면 같은 시점의 PostgreSQL과 첨부 백업을 복원한 후 해당 버전 이미지로 실행합니다. 데이터 복원은 현재 데이터를 덮어쓸 수 있으므로 격리 환경에서 먼저 시험하세요.

## 7. 전체 백업과 복원

같은 시점의 다음 자료가 필요합니다.

- PostgreSQL 전체 논리 백업: `pg_dump`의 custom 형식
- `/var/lib/madi` 파일 볼륨의 백업
- ENCRYPTION_KEY 및 나머지 네 환경변수의 안전한 별도 사본
- 사용한 이미지 버전, TLS·프록시·배포 구성

일관성을 위해 서비스 쓰기를 중단하거나 서비스를 정지한 상태에서 DB와 파일을 함께 백업합니다. 아래 명령의 연결 정보는 운영 환경에 맞는 비밀 주입 방법으로 설정합니다.

```sh
docker compose stop madi
pg_dump --format=custom --file=madi-postgres.dump 'postgresql://madi@postgres.internal:5432/madi'
# 조직의 백업 도구로 madi-data 볼륨 전체를 같은 백업 세트에 보관합니다.
docker compose start madi
```

비밀번호는 `.pgpass` 또는 운영 비밀 관리 도구를 사용하고 명령행에 넣지 않습니다. 백업 결과와 종료 코드를 확인한 뒤 서비스를 시작하세요. 관리자 ZIP은 아래의 논리 복원에 사용할 수 있으며 대규모 복구와 PITR에는 PostgreSQL 네이티브 백업을 사용합니다.

복원 시험은 새 데이터베이스와 새 볼륨에서 진행합니다.

```sh
createdb --host=postgres.internal --username=madi madi_restore_test
pg_restore --exit-on-error --no-owner --dbname='postgresql://madi@postgres.internal:5432/madi_restore_test' madi-postgres.dump
```

첨부파일을 새 볼륨에 복구하고 같은 ENCRYPTION_KEY 및 백업 당시 서비스 이미지를 사용해 시작합니다. 사용자 로그인, 문서·버전·첨부, OIDC·AI 비밀의 복호화, 키 정책을 점검합니다. 성공한 백업의 시각·복원 소요 시간·점검 결과를 기록합니다. 검증 후 운영 대상 전환은 별도의 변경 관리 절차에 따릅니다.

### 관리자 ZIP 복원

관리자 백업 페이지에서 만든 ZIP은 같은 madi 버전과 같은 ENCRYPTION_KEY를 사용하는 환경에 복원할 수 있습니다. 먼저 빈 시험용 환경을 준비하고 관리자 백업 페이지에서 파일을 선택한 뒤 확인 문구 `RESTORE`를 입력합니다. 이 작업은 대상 DB의 현재 데이터를 교체합니다.

복원은 ZIP 256MB, 압축 해제 1GB, 10,000개 항목 이내로 제한됩니다. 파일을 준비한 뒤 DB 변경을 트랜잭션으로 적용하고 실패 시 현재 DB를 유지합니다. 기존 첨부파일은 복구할 수 있도록 남겨 두며 저장소 설정은 대상 환경의 경로를 사용합니다. 성공하면 세션이 무효화되므로 백업에 있던 계정으로 다시 로그인하세요. 대용량·장기 보관·PITR에는 앞의 PostgreSQL 백업 절차를 사용합니다.

API 연동은 관리자 쿠키 세션으로 `POST /api/v1/admin/restore`에 multipart `file`과 `confirmation=RESTORE`를 보냅니다. API 키는 관리자 복원 권한을 제공하지 않습니다.

### PostgreSQL 백업·복원 스크립트

제공 스크립트는 기본 로컬 경로에 있는 첨부파일에 한정합니다. 프로필 기반 Local/S3/MinIO 첨부가 존재하면 불완전한 백업을 만들지 않도록 거부합니다. 여러 프로필을 쓰는 대규모 환경은 서비스를 정지한 상태에서 `pg_dump`와 **모든 로컬 프로필 루트 및 S3/MinIO 버킷·접두사의 스토리지 스냅샷**을 동일 세트로 보관하세요. DB만 복원하거나 기본 볼륨만 복사하면 원격 첨부는 복구되지 않습니다. 각 스토리지의 조직 표준 백업 도구로 UUID 키·객체 바이트를 그대로 보존하고, 격리 환경에서 전체 파일 체크섬과 권한·복호화를 검증합니다. 암호화 키는 별도 보관합니다.

저장소는 `scripts/backup-postgres.sh`와 `scripts/restore-postgres.sh`를 제공합니다. 실행 환경에는 Bash, 해당 서버 버전과 호환되는 PostgreSQL 클라이언트(`pg_dump`, `pg_restore`, `psql`), GNU tar, sha256sum, realpath가 필요합니다. 비밀번호가 없는 명시적 PostgreSQL URI를 사용하며 비밀번호는 권한을 제한한 `.pgpass` 또는 `PGPASSFILE`로 주입합니다. 비밀번호가 포함된 DSN은 스크립트가 거부합니다.

서비스의 모든 쓰기를 중지한 뒤 다음과 같이 백업합니다. 출력 디렉터리는 새 경로여야 하며 부모 디렉터리를 먼저 준비합니다.

```sh
bash scripts/backup-postgres.sh \
  --dsn 'postgresql://madi@postgres.internal:5432/madi?sslmode=verify-full' \
  --attachments /srv/madi/attachments \
  --source-storage-path /var/lib/madi/attachments \
  --output /backup/madi-20260908 \
  --acknowledge WRITES_STOPPED
```

`--attachments`는 스크립트가 파일을 읽는 호스트 경로이고 `--source-storage-path`는 DB에 기록된 서비스 내부 경로입니다. 두 경로가 같으면 후자를 생략할 수 있습니다. Compose의 이름 있는 볼륨을 사용하는 경우 사내 백업 도구로 해당 볼륨을 읽을 수 있도록 준비하고 정확한 첨부 디렉터리를 지정하세요.

결과는 `postgres.dump`, `attachments.tar.gz`, `backup-info.txt`, `SHA256SUMS`입니다. 암호화된 설정은 DB 덤프에 포함되고 ENCRYPTION_KEY는 포함되지 않습니다. 실패한 세트에는 `INCOMPLETE` 표시가 남으며 복원 스크립트는 이를 거부합니다. 성공 메시지와 체크섬을 확인한 뒤 서비스를 재시작합니다.

복원 스크립트는 기존 운영 DB를 덮어쓰지 않습니다. 운영자가 미리 만든 빈 DB 중 이름이 `_restore_test` 또는 `_restore_verify`로 끝나는 별도 대상만 허용합니다. 첨부파일도 새 디렉터리에 복원합니다.

```sh
createdb --host=postgres.internal --username=madi madi_restore_test
bash scripts/restore-postgres.sh \
  --backup /backup/madi-20260908 \
  --target-dsn 'postgresql://madi@postgres.internal:5432/madi_restore_test?sslmode=verify-full' \
  --attachments /srv/madi-restore/attachments \
  --target-storage-path /var/lib/madi/attachments \
  --acknowledge RESTORE_DISPOSABLE_TARGET
```

체크섬, 아카이브 경로, 서비스 버전, 빈 대상 DB를 확인한 뒤 단일 트랜잭션으로 PostgreSQL을 복원합니다. 첨부 경로와 저장소 설정을 대상 경로에 맞추고 기존 세션은 폐기합니다. 같은 ENCRYPTION_KEY·서비스 버전으로 시험 인스턴스를 시작하고 파일 소유권·로그인·문서·첨부·연동을 확인하세요. 실패 시 생성된 시험 DB와 파일은 진단을 위해 남으며 자동으로 삭제하지 않습니다. 검증한 대상의 운영 전환은 조직의 변경 절차에 따릅니다.

## 8. 문제 해결

| 증상 | 확인 항목 |
| --- | --- |
| 시작 시 DB 오류 | DSN, DNS, 방화벽, DB 사용자 권한, 인증서 |
| 암호화 키 오류 | Base64 형식, 디코딩 길이 32바이트, 기존 배포와 같은 키인지 |
| SSO 리디렉션 실패 | 관리자 서비스 URL, issuer, Keycloak redirect URI |
| AI가 한 번에 출력됨 | 프록시 SSE 버퍼링과 제공자의 streaming 지원 |
| 파일 업로드 실패 | 파일 볼륨 용량, UID 10001 권한, 파일 크기 제한 |
| 403 접근 오류 | 계정 비활성화, 워크스페이스 역할, 문서 공개 범위, 키 범위/IP |
| 409 저장 충돌 | 다른 사용자가 저장한 최신 버전 확인 후 재편집 |
| 429 호출 제한 | 계정/IP/키 호출 속도를 낮추고 정책 확인 |

서버 로그는 요청 ID를 기준으로 추적합니다. 로그 수집 전에 비밀과 문서 내용이 포함되었는지 확인하고 필요한 정보만 공유하세요.
