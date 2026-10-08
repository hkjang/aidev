import json
import sys
from pathlib import Path
from urllib.parse import urlparse
from playwright.sync_api import sync_playwright

OUT = Path(__file__).parent
PHASE = sys.argv[1]
BASE = 'http://127.0.0.1:4179'
FIELDS = {
    'sla': ['critical_days', 'high_days', 'medium_days', 'low_days', 'info_days', 'due_soon_days'],
    'risk': ['kev_boost', 'epss_boost', 'stale_after_days'],
}
DEFAULTS = {
    'sla': dict(enabled=False, critical_days=7, high_days=30, medium_days=90, low_days=180, info_days=0, due_soon_days=7),
    'risk': dict(kev_boost=25, epss_threshold=0.1, epss_boost=10, stale_after_days=30),
}
with sync_playwright() as p:
    browser = p.chromium.launch(executable_path='/opt/google/chrome/chrome', headless=True, args=['--no-sandbox'])
    for viewport in [dict(width=1440, height=1000), dict(width=390, height=844)]:
        context = browser.new_context(viewport=viewport, permissions=['clipboard-read', 'clipboard-write'])
        page = context.new_page()
        page.on('dialog', lambda dialog: dialog.accept())
        sent = []
        errors = []
        unknown = set()
        page.on('pageerror', lambda error: (errors.append(str(error)), print('PAGE ERROR:', error, flush=True)))
        page.on('console', lambda msg: print('CONSOLE ERROR:', msg.text, flush=True) if msg.type == 'error' else None)
        def route_api(route):
            request = route.request
            path = urlparse(request.url).path
            if path == '/api/auth/me':
                response = {'user': dict(id='synthetic-admin', username='synthetic', name='합성 자료 · 입력 검증', role='admin')}
            elif path == '/api/settings/public':
                response = dict(version='1.24.0', site_name='합성 자료 · 입력 검증')
            elif path == '/api/auth/config':
                response = {}
            elif path == '/api/admin/tracking':
                response = dict(enabled=False, name='방문 통계', script='', allowed_origins=[], revision=0, updated_at=None)
            elif path == '/api/admin/tracking/violations':
                response = dict(violations=[], limit=100)
            elif path == '/api/settings':
                response = DEFAULTS
            elif path.startswith('/api/settings/') and request.method == 'PUT':
                response = request.post_data_json
                sent.append((path, response))
            else:
                unknown.add(path)
                print('OTHER API:', path, flush=True)
                response = []
            route.fulfill(status=200, content_type='application/json', body=json.dumps(response))
        context.route('**/api/**', route_api)
        def enter(field, value, method):
            field.fill('')
            if method == 'typing':
                field.press_sequentially(value)
            else:
                page.evaluate('(text) => navigator.clipboard.writeText(text)', value)
                field.press('Control+V')
            field.press('Tab')
        def save(group):
            with page.expect_response(lambda res: res.url.endswith('/api/settings/' + group) and res.request.method == 'PUT'):
                page.get_by_role('button', name='설정 저장', exact=True).click()
            page.get_by_role('button', name='설정 저장', exact=True).wait_for(state='visible')
            return sent[-1][1]
        for group, fields in FIELDS.items():
            page.goto(BASE + '/admin/settings?tab=' + group)
            page.locator('#settings-' + group + '-' + fields[0]).wait_for()
            for key in fields:
                field = page.locator('#settings-' + group + '-' + key)
                for method in ['typing', 'paste']:
                    enter(field, '1.5', method)
                    display = field.input_value()
                    payload = save(group)
                    value = payload[key]
                    print(f'{viewport["width"]} {group}.{key} {method} 1.5 -> display={display}, PUT={value}', flush=True)
                    assert isinstance(value, (int, float)) and float(value).is_integer(), f'{group}.{key} {method} submits fractional value {value}'
                for value in ([1, 3650] if key == 'stale_after_days' else [0, 100] if group == 'risk' else [0, 3650]):
                    enter(field, str(value), 'typing')
                    assert save(group)[key] == value
            if group == 'risk':
                epss = page.locator('#settings-risk-epss_threshold')
                for method in ['typing', 'paste']:
                    for value in ['0', '0.1', '0.125', '1']:
                        enter(epss, value, method)
                        payload = save(group)
                        print(f'{viewport["width"]} risk.epss_threshold {method} {value} -> PUT={payload["epss_threshold"]}', flush=True)
                        assert payload['epss_threshold'] == float(value)
                enter(epss, '0.125', 'typing')
                # Visible marking makes the provenance explicit in every capture.
                page.evaluate('''() => { const n = document.createElement('div'); n.textContent = '합성 자료 · 실제 앱 UI / API 모의 응답 · DB 미검증'; Object.assign(n.style, {position:'fixed',top:'0',left:'0',zIndex:'99999',background:'#fff3bf',color:'#000',padding:'8px',fontSize:'12px'}); document.body.append(n); }''')
                page.get_by_role('heading', name='조치 우선순위', exact=True).click()
                page.evaluate('window.scrollTo(0, 0)')
                page.screenshot(path=str(OUT / f'{PHASE}-risk-{viewport["width"]}.png'), full_page=True)
        # A separate group save must preserve the unsaved SLA draft.
        page.goto(BASE + '/admin/settings?tab=sla')
        field = page.locator('#settings-sla-critical_days')
        field.wait_for()
        enter(field, '12', 'typing')
        page.get_by_role('button', name='조치 우선순위', exact=True).click()
        page.get_by_role('button', name='입력 유지하고 이동', exact=True).click()
        epss = page.locator('#settings-risk-epss_threshold')
        epss.wait_for()
        enter(epss, '0.125', 'typing')
        payload = save('risk')
        assert 'critical_days' not in payload
        page.locator('.settings-navigation button').filter(has_text='조치 기한 · SLA').click()
        assert page.locator('#settings-sla-critical_days').input_value() == '12'
        assert not errors, errors
        print(f'{viewport["width"]}: group save/draft preservation PASS; page errors=0; other synthetic API routes={sorted(unknown)}', flush=True)
        context.close()
    browser.close()
print('PASS: desktop/mobile numeric input, clipboard paste, blur, PUT bodies, boundaries, EPSS fractions and group draft preservation')
