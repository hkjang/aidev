from pathlib import Path
import json
root = Path(__file__).resolve().parent
prior = root.parent / '2026-09-21-113403-aiportal-front-improve' / 'ideas.json'
a = json.loads((root / 'ideas.json').read_text())
b = json.loads(prior.read_text())
assert len(a) == len(b) + 2
assert [x['title'] for x in a[:len(b)]] == [x['title'] for x in b]
assert [x['status'] for x in a[:len(b)]] == [x['status'] for x in b]
for x in a:
    assert set(x) == {'title','value','risk','size','status','note','updated'}
    assert 1 <= x['value'] <= 5 and 1 <= x['risk'] <= 5
    assert x['size'] in ('S','M','L') and x['status'] in ('pending','done','rejected')
    assert x['updated'] == '2026-09-21'
assert len((root / 'profile.md').read_text().splitlines()) <= 60
assert (root / 'journal.md').read_text().count('## 정찰 노트') == 1
assert (root / 'ledger-entry.md').read_text().count('- 수정 과제:') == 1
assert (root / 'brief.md').is_file()
print('PASS: records only; 29 preserved + 2 new. Release acceptance remains unverified.')
