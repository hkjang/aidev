import json

src = '/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-194424-aiportal-front-improve/ideas.json'
dst = '/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-22-212444-aiportal-front-improve/ideas.json'
items = json.load(open(src, encoding='utf-8'))
print('loaded', len(items))
titles = [i['title'] for i in items]
D = '2026-09-22'

def find(sub):
    return [i for i in items if sub in i['title']]

for i in find('릴리즈 버전 결정 입력'):
    i['note'] = ('21:24 정찰: 교착 원인을 외부 원문에서 확정 — release-prompt.md:21,22(2·3항)은 이전 증가 패턴/이전 방식을 '
                 '요구하나 태그 0개·릴리즈 커밋 0개로 따를 패턴 없음, :24(5항) skipped는 버전 파일 부재를 요구하나 '
                 'package.json:3 version 0.0.0 존재. released·skipped 모두 불가한 구조적 교착이며 해소는 5항이 명시한 '
                 '사람의 일. 저장소 안 합법 수단 없음 → pending 유지, 관례 신설 금지.')
    i['updated'] = D

for i in find('useAppList'):
    i['note'] = ('21:24 정찰이 코드로 검증: useAppList.js:40,53의 `|| []`는 객체가 truthy라 방어 실패 → '
                 'getFormattedAppList의 list.map이 TypeError. 형제 경로 src/utils/appList.js:77은 toAppArray(:18)로 '
                 '이미 방어되고 tests/unit/appList.spec.js:146이 비배열 케이스를 덮음 = 두 경로 중 하나만 방어됨. '
                 '소비자 Home.vue:35, Chat/Index.vue:40. 이번 회차 구현 대상(1순위 진입 조건 미충족 시 기본 실행).')
    i['updated'] = D
    i['status'] = 'pending'

new = [
    {
        'title': '릴리즈 절차 교착 자체를 사람에게 에스컬레이션 (release-prompt 2·3항 vs 5항)',
        'value': 5, 'risk': 1, 'size': 'S', 'status': 'pending',
        'note': ('release-prompt.md:21,22는 이전 릴리즈 증가 패턴/같은 방식을 요구하고 :24의 skipped는 버전 파일도 '
                 '없을 것을 요구한다. 릴리즈 이력 0 + package.json 존재인 저장소는 released도 skipped도 불가 → 영구 '
                 'failed. 수정 지점은 이 저장소가 아니라 외부 aidev 절차이며 사람의 결정이 필요하다. 구현자는 이 '
                 '항목을 코드로 고치려 하지 말 것.'),
        'updated': D,
    },
    {
        'title': 'docs/RELEASE.md 재조회 스크립트의 0.0.0 하드 assert',
        'value': 2, 'risk': 1, 'size': 'S', 'status': 'pending',
        'note': ('docs/RELEASE.md:45가 package/lock 세 필드를 0.0.0으로 assert한다. 실제 릴리즈로 버전이 오르면 문서에 '
                 '적힌 재조회 명령이 즉시 실패한다. 문서 스스로 값이 달라지면 되돌리지 말라고 적었으나 명령은 그대로다. '
                 '릴리즈 관례가 정해지기 전에는 손대지 말 것.'),
        'updated': D,
    },
]
for n in new:
    if n['title'] not in titles:
        items.append(n)

json.dump(items, open(dst, 'w', encoding='utf-8'), ensure_ascii=False, indent=2)
print('wrote', len(items), 'items')
