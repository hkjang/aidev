import json
from playwright.sync_api import sync_playwright, expect

BASE = 'http://127.0.0.1:9911'
def detail(i):
    return dict(id=i, db_seq=i+10, db_name=f'DB{i}', sql=f'SELECT {i}', row_count=1,
                columns=[dict(name='n', type='INT')], rows=[dict(n=i)])

with sync_playwright() as p:
    browser = p.chromium.launch(args=['--no-sandbox'])
    page = browser.new_page()
    page.goto(BASE + '/login')
    page.locator('input[type=email]').fill('admin@example.com')
    page.locator('input[type=password]').fill('Smoke-Test-1234')
    page.locator('button[type=submit]').click()
    page.wait_for_url(BASE + '/')
    pending = []
    entries = [detail(1), detail(2)]
    def route_saved(route):
        if '?' in route.request.url:
            route.fulfill(json=dict(data=dict(items=entries)))
        else:
            pending.append(route)
    page.route('**/api/v1/query/saved**', route_saved)
    page.goto(BASE + '/saved')
    expect(page.locator('.resource-card')).to_have_count(2)
    page.on('dialog', lambda dialog: dialog.accept())
    def select(i):
        count = len(pending)
        page.locator('.resource-card').filter(has_text=f'#{i} ·').click()
        for _ in range(100):
            if len(pending) > count:
                return pending[-1]
            page.wait_for_timeout(10)
        raise AssertionError('missing detail request')
    def ready(i):
        expect(page.locator('#saved-title')).to_contain_text(f'결과 #{i} ·')
        expect(page.locator('#saved-sql')).to_have_text(f'SELECT {i}')
        expect(page.locator('#saved-result tbody td').nth(1)).to_have_text(str(i))
        expect(page.locator('.resource-card.active')).to_contain_text(f'#{i} ·')
        expect(page.locator('#saved-delete')).to_be_visible()
    def blocked():
        for selector in ['#saved-sql', '#saved-tsv', '#saved-open', '#saved-delete']:
            expect(page.locator(selector)).to_be_hidden()
        expect(page.locator('#saved-result table')).to_have_count(0)
    select(1).fulfill(json=dict(data=detail(1))); ready(1)
    b = select(2); blocked()
    b.fulfill(status=500, json=dict(detail='조회 실패'))
    expect(page.locator('#saved-title')).to_contain_text('실패'); blocked()
    for failure in [False, True]:
        a = select(1)
        select(2).fulfill(json=dict(data=detail(2))); ready(2)
        if failure:
            a.fulfill(status=500, json=dict(detail='old failure'))
        else:
            a.fulfill(json=dict(data=detail(1)))
        page.wait_for_timeout(100); ready(2)
    select(1).fulfill(json=dict(data=detail(1))); ready(1)
    page.locator('#saved-delete').click()
    expect(page.locator('#saved-delete')).to_be_disabled()
    page.wait_for_timeout(100)
    deletion = pending[-1]
    assert deletion.request.method == 'DELETE' and deletion.request.url.endswith('/1')
    select(2).fulfill(json=dict(data=detail(2))); ready(2)
    entries[:] = [detail(2)]
    deletion.fulfill(json={})
    expect(page.locator('#saved-delete')).to_be_enabled(); ready(2)
    expect(page.locator('.resource-card')).to_have_count(1)
    # A mocked list reset permits a separate delete/reselection race without touching real data.
    entries[:] = [detail(1), detail(2)]
    page.reload(); expect(page.locator('.resource-card')).to_have_count(2)
    select(1).fulfill(json=dict(data=detail(1))); ready(1)
    page.locator('#saved-delete').click(); page.wait_for_timeout(100)
    deletion = pending[-1]
    a = select(1)
    entries[:] = [detail(2)]
    deletion.fulfill(json={})
    expect(page.locator('#saved-title')).to_have_text('결과를 선택하세요')
    a.fulfill(json=dict(data=detail(1))); page.wait_for_timeout(100)
    blocked()
    print('PASS: actual embedded /saved page clicks; loading/failure, late success/failure, delete target/B preservation, deleted A resurrection blocked. Saved APIs mocked; real saved data untouched.')
    browser.close()
