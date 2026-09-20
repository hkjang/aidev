## KCB Cutover Dashboard v1.7.0

### 버그 수정
- **삭제·상태 전파의 하위 탐색을 ID 접두사 대신 parentId 관계로**: 지금까지 부모 작업을 지우거나 상태를 바꿀 때 하위를 `id.startsWith('<부모ID>-')`로 찾았습니다. 기본 데이터처럼 `1` → `1-1` → `1-1-1` 형식이면 문제가 없지만, 관리자 콘솔에서 **업로드한 `activity.json`의 id는 아무 문자열이나 될 수 있어**(업로드 검증은 `parentId` 관계만 봅니다) `net` → `vlan` → `vlan-check` 같은 트리에서는 다음 두 가지가 잘못 동작했습니다.
  - **부모를 삭제해도 항목이 사라지지 않음** — 자식이 고아로 남아 저장 검증에서 400으로 거부되고, 화면에는 아무 오류 없이 그대로 남아 있었습니다.
  - **부모 상태를 바꿔도 자식이 따라오지 않음** — 관리 화면 드롭다운으로 `net`을 완료로 바꿔도 `vlan`·`vlan-check`는 대기 그대로였습니다.
  - 이제 두 경로 모두 `parentId` 관계로 하위를 찾습니다(`lib/treeUtils.ts`의 `collectDescendantIds` — `parentId` 인덱스로 너비 우선 탐색, 순환 데이터도 종료). 접두사만 같은 다른 부모의 항목(예: `parentId`가 `db`인 `net-x`)은 건드리지 않고, 기본 데이터의 `1-1` 형식 트리는 이전과 똑같이 동작합니다.
  - id 규칙이 바뀌는 것은 아니므로 **기존 데이터나 업로드 파일을 고칠 필요는 없습니다.**

### 문서
- `docs/ADMIN_GUIDE.md` 4.3절 — 하위 항목 id가 `<부모ID>-`로 시작해야 한다는 제약 문장을 `parentId` 기준으로 고쳤습니다. 6장 문제 해결의 "삭제해도 항목이 사라지지 않음" 행은 이 버그의 증상이었으므로 삭제했습니다.

### 변경 파일
- `lib/treeUtils.ts` — `collectDescendantIds(activities, rootId)` 신설, `deleteActivity`가 사용
- `app/admin/page.tsx` — `onStatusChange`의 하위 상태 전파가 같은 함수를 사용
- `lib/treeUtils.test.ts`(신규), `e2e/tree-ops.spec.ts`(신규)
- `docs/ADMIN_GUIDE.md`

### 검증
- eslint · tsc 통과, `node --test` 단위 테스트 72건(신규 7건: `collectDescendantIds` 3 — 임의 id 3단 트리·접두사만 같은 다른 부모 보존·순환 종료, `deleteActivity` 4 — 초기 데이터 회귀 포함) 통과
- Playwright e2e 10건(신규 2건: API `{action:'delete', targetId:'net'}` 200 후 GET에 `vlan`·`vlan-check` 없음과 `net-x`(parentId `db`) 보존, 관리 화면 드롭다운으로 `net`을 완료로 바꾸면 `vlan`·`vlan-check`도 완료·`db`·`net-x`는 대기) 통과
- 단위·e2e 모두 이전 로직으로 되돌리면 각각 2건이 실패함을 확인한 뒤 복원(회귀 테스트로 고정)
- Docker 멀티스테이지 빌드 내 Next.js 프로덕션 빌드(`npm run build`) 통과, 이미지의 `/api/activities` 번들에 `parentId` 인덱스 기반 탐색이 들어가고 접두사 `startsWith` 분기가 없음을 확인

### 업그레이드
데이터 파일 형식은 이전 버전과 같으므로 `cutover-data` 볼륨을 그대로 물려주면 됩니다. 설정 항목이나 환경 변수는 추가되지 않았습니다.

```bash
docker load -i cutover-v1.7.0.tar.gz
docker stop cutover-app && docker rm cutover-app
docker run -d --name cutover-app --restart always -p 3000:3000 \
  -e ADMIN_PASSWORD='<관리자-비밀번호>' -e USER_PASSWORD='<사용자-비밀번호>' -e AUTH_SECRET='<기존과-같은-값>' \
  -v cutover-data:/app/data cutover:v1.7.0
```

### 자산
- `cutover-v1.7.0.tar.gz` — `docker save cutover:v1.7.0 | gzip` (node:22-alpine, standalone)
