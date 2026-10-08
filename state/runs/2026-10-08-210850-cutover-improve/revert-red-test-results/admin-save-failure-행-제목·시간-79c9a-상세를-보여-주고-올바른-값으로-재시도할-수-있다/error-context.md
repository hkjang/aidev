# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: admin-save-failure.spec.ts >> 행 제목·시간의 실제 검증 실패는 모든 상세를 보여 주고 올바른 값으로 재시도할 수 있다
- Location: e2e/admin-save-failure.spec.ts:202:5

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: getByTestId('admin-save-error')
Expected substring: "title은 300자 이하여야 합니다."
Received string:    "activity.json 데이터 형식이 올바르지 않습니다."
Timeout: 5000ms

Call log:
  - Expect "soft toContainText" with timeout 5000ms
  - waiting for getByTestId('admin-save-error')
    14 × locator resolved to <div role="alert" data-testid="admin-save-error" class="mb-6 flex items-start gap-2 rounded-2xl border border-red-200 bg-red-50 p-4 font-medium text-red-800">…</div>
       - unexpected value "activity.json 데이터 형식이 올바르지 않습니다."

```

```yaml
- alert: activity.json 데이터 형식이 올바르지 않습니다.
```

```
Error: expect(locator).toContainText(expected) failed

Locator: getByTestId('admin-save-error')
Expected substring: "time은 100자 이하여야 합니다."
Received string:    "activity.json 데이터 형식이 올바르지 않습니다."
Timeout: 5000ms

Call log:
  - Expect "soft toContainText" with timeout 5000ms
  - waiting for getByTestId('admin-save-error')
    14 × locator resolved to <div role="alert" data-testid="admin-save-error" class="mb-6 flex items-start gap-2 rounded-2xl border border-red-200 bg-red-50 p-4 font-medium text-red-800">…</div>
       - unexpected value "activity.json 데이터 형식이 올바르지 않습니다."

```

```yaml
- alert: activity.json 데이터 형식이 올바르지 않습니다.
```

# Test source

```ts
  134 |   expect(confirmMessages).toHaveLength(1);
  135 |   expect(confirmMessages[0]).toContain('[E2E] 하위 작업');
  136 |   expect(confirmMessages[0]).not.toContain('[E2E] 최상위 작업');
  137 | 
  138 |   const remaining = (await readData(page.request)).activities.map((a) => a.id);
  139 |   expect(remaining).toEqual(['root']);
  140 | });
  141 | 
  142 | test('인라인 편집 저장이 실패하면 입력이 닫히지 않고 입력한 값이 남는다', async ({ page }) => {
  143 |   await loginThroughUi(page);
  144 | 
  145 |   await page.getByTitle('내용 편집').nth(1).click();
  146 |   const titleInput = page.getByTestId('activity-edit-title');
  147 |   await titleInput.fill('[E2E] 하위 작업 수정본');
  148 | 
  149 |   await failEveryPut(page);
  150 |   await page.getByTitle('편집 저장').click();
  151 |   await expect(page.getByTestId('admin-save-error')).toContainText('상황판 데이터를 저장하지 못했습니다.');
  152 |   await expect(titleInput).toHaveValue('[E2E] 하위 작업 수정본');
  153 | 
  154 |   // 가로채기를 풀고 다시 저장하면 편집이 닫히고 값이 반영된다.
  155 |   await page.unroute('**/api/activities');
  156 |   const putDone = waitForPut(page);
  157 |   await page.getByTitle('편집 저장').click();
  158 |   expect((await putDone).status()).toBe(200);
  159 | 
  160 |   await expect(page.getByTestId('admin-save-error')).toHaveCount(0);
  161 |   await expect(page.getByTestId('activity-edit-title')).toHaveCount(0);
  162 |   await expect(page.getByText('[E2E] 하위 작업 수정본', { exact: true })).toBeVisible();
  163 | });
  164 | 
  165 | test('상황판 제목 저장이 실패하면 편집이 닫히지 않고 실패 메시지가 보인다', async ({ page }) => {
  166 |   await loginThroughUi(page);
  167 | 
  168 |   await page.getByTitle('제목 수정').click();
  169 |   const titleInput = page.getByTestId('dashboard-title-input');
  170 |   await titleInput.fill('[E2E] 상황판 제목');
  171 | 
  172 |   await failEveryPut(page);
  173 |   await page.getByTitle('저장').click();
  174 |   await expect(page.getByTestId('admin-save-error')).toContainText('상황판 데이터를 저장하지 못했습니다.');
  175 |   await expect(titleInput).toHaveValue('[E2E] 상황판 제목');
  176 | 
  177 |   await page.unroute('**/api/activities');
  178 |   const putDone = waitForPut(page);
  179 |   await page.getByTitle('저장').click();
  180 |   expect((await putDone).status()).toBe(200);
  181 | 
  182 |   await expect(page.getByTestId('admin-save-error')).toHaveCount(0);
  183 |   await expect(page.getByTestId('dashboard-title-input')).toHaveCount(0);
  184 |   await expect(page.getByText('[E2E] 상황판 제목', { exact: true })).toBeVisible();
  185 | });
  186 | 
  187 | 
  188 | const validationSummary = 'activity.json 데이터 형식이 올바르지 않습니다.';
  189 | 
  190 | /** 실제 서버에서 거절받아 다음 실패가 이전 상세를 지우는지도 확인한다. */
  191 | async function rejectLongDashboardTitle(page: Page) {
  192 |   await page.getByTitle('제목 수정').click();
  193 |   await page.getByTestId('dashboard-title-input').fill('가'.repeat(101));
  194 |   const putDone = waitForPut(page);
  195 |   await page.getByTitle('저장', { exact: true }).click();
  196 |   const response = await putDone;
  197 |   expect(response.status()).toBe(400);
  198 |   expect((await response.json()).code).toBe('INVALID_DATA');
  199 |   await expect(page.getByTestId('admin-save-error')).toContainText(validationSummary);
  200 | }
  201 | 
  202 | test('행 제목·시간의 실제 검증 실패는 모든 상세를 보여 주고 올바른 값으로 재시도할 수 있다', async ({ page }) => {
  203 |   await loginThroughUi(page);
  204 |   const before = await readData(page.request);
  205 |   await page.getByTitle('내용 편집').nth(1).click();
  206 |   const titleInput = page.getByTestId('activity-edit-title');
  207 |   const timeInput = page.getByTestId('activity-edit-time');
  208 |   const longTitle = '가'.repeat(301);
  209 |   const longTime = '나'.repeat(101);
  210 |   await titleInput.fill(longTitle);
  211 |   await timeInput.fill(longTime);
  212 | 
  213 |   const rejected = waitForPut(page);
  214 |   await page.getByTitle('편집 저장').click();
  215 |   const response = await rejected;
  216 |   expect(response.status()).toBe(400);
  217 |   expect(await response.json()).toMatchObject({
  218 |     error: validationSummary,
  219 |     code: 'INVALID_DATA',
  220 |     details: [
  221 |       { path: 'activities[1].time', message: 'time은 100자 이하여야 합니다.' },
  222 |       { path: 'activities[1].title', message: 'title은 300자 이하여야 합니다.' },
  223 |     ],
  224 |   });
  225 |   // 수정 전에도 성립하는 저장 불변·입력 유지를 상세 노출보다 먼저 단정한다.
  226 |   expect(await readData(page.request)).toMatchObject({
  227 |     activities: before.activities, dashboardTitle: before.dashboardTitle, lastUpdated: before.lastUpdated,
  228 |   });
  229 |   await expect(titleInput).toHaveValue(longTitle);
  230 |   await expect(timeInput).toHaveValue(longTime);
  231 |   const banner = page.getByTestId('admin-save-error');
  232 |   await expect(banner).toContainText(validationSummary);
  233 |   await expect.soft(banner).toContainText('title은 300자 이하여야 합니다.');
> 234 |   await expect.soft(banner).toContainText('time은 100자 이하여야 합니다.');
      |                             ^ Error: expect(locator).toContainText(expected) failed
  235 | 
  236 |   await titleInput.fill('[E2E] 검증 후 수정본');
  237 |   await timeInput.fill('10:00 ~ 11:00');
  238 |   const accepted = waitForPut(page);
  239 |   await page.getByTitle('편집 저장').click();
  240 |   expect((await accepted).status()).toBe(200);
  241 |   const saved = await readData(page.request);
  242 |   expect(saved.activities).toEqual([tree[0], { ...tree[1], title: '[E2E] 검증 후 수정본', time: '10:00 ~ 11:00' }]);
  243 |   expect(saved.dashboardTitle).toBe(before.dashboardTitle);
  244 |   await expect(banner).toHaveCount(0);
  245 |   await expect(titleInput).toHaveCount(0);
  246 |   await expect(timeInput).toHaveCount(0);
  247 | });
  248 | 
  249 | test('상황판 제목의 실제 검증 실패는 제한 원인을 보여 주고 재저장하면 상세와 편집을 닫는다', async ({ page }) => {
  250 |   await loginThroughUi(page);
  251 |   const before = await readData(page.request);
  252 |   await rejectLongDashboardTitle(page);
  253 |   expect(await readData(page.request)).toMatchObject({
  254 |     activities: before.activities, dashboardTitle: before.dashboardTitle, lastUpdated: before.lastUpdated,
  255 |   });
  256 |   const titleInput = page.getByTestId('dashboard-title-input');
  257 |   await expect(titleInput).toHaveValue('가'.repeat(101));
  258 |   const banner = page.getByTestId('admin-save-error');
  259 |   await expect.soft(banner).toContainText('dashboardTitle은 100자 이하여야 합니다.');
  260 | 
  261 |   await titleInput.fill('[E2E] 검증 후 상황판');
  262 |   const accepted = waitForPut(page);
  263 |   await page.getByTitle('저장', { exact: true }).click();
  264 |   expect((await accepted).status()).toBe(200);
  265 |   const saved = await readData(page.request);
  266 |   expect(saved.dashboardTitle).toBe('[E2E] 검증 후 상황판');
  267 |   expect(saved.activities).toEqual(before.activities);
  268 |   await expect(banner).toHaveCount(0);
  269 |   await expect(titleInput).toHaveCount(0);
  270 | });
  271 | 
  272 | for (const [name, details] of [
  273 |   ['null', null],
  274 |   ['문자열', '잘못된 상세'],
  275 |   ['객체', { path: 'dashboardTitle', message: '배열이 아님' }],
  276 |   ['잘못된 항목', [null, 123, '문자열', {}, [], { path: 1, message: '잘못된 경로' }, { path: 'title', message: {} }, { message: '경로 없음' }]],
  277 | ] as const) {
  278 |   test(`잘못된 details(${name})는 무시하고 다음 실패의 일반 오류만 표시한다`, async ({ page }) => {
  279 |     await loginThroughUi(page);
  280 |     await rejectLongDashboardTitle(page);
  281 |     await page.route('**/api/activities', async (route) => {
  282 |       if (route.request().method() !== 'PUT') return route.fallback();
  283 |       await route.fulfill({
  284 |         status: 500,
  285 |         contentType: 'application/json',
  286 |         body: JSON.stringify({ error: '상황판 데이터를 저장하지 못했습니다.', details }),
  287 |       });
  288 |     });
  289 |     const rejected = waitForPut(page);
  290 |     await page.getByTitle('저장', { exact: true }).click();
  291 |     expect((await rejected).status()).toBe(500);
  292 |     await expect(page.getByTestId('admin-save-error')).toHaveText('상황판 데이터를 저장하지 못했습니다.');
  293 |     await expect(page.getByTestId('dashboard-title-input')).toHaveValue('가'.repeat(101));
  294 |   });
  295 | }
  296 | 
  297 | test('비JSON 실패 응답은 이전 상세 대신 HTTP 상태 fallback을 표시한다', async ({ page }) => {
  298 |   await loginThroughUi(page);
  299 |   await rejectLongDashboardTitle(page);
  300 |   await page.route('**/api/activities', async (route) => {
  301 |     if (route.request().method() !== 'PUT') return route.fallback();
  302 |     await route.fulfill({ status: 502, contentType: 'text/html', body: '<h1>Bad Gateway</h1>' });
  303 |   });
  304 |   const rejected = waitForPut(page);
  305 |   await page.getByTitle('저장', { exact: true }).click();
  306 |   expect((await rejected).status()).toBe(502);
  307 |   await expect(page.getByTestId('admin-save-error')).toHaveText('변경 사항을 저장하지 못했습니다. (HTTP 502)');
  308 |   await expect(page.getByTestId('dashboard-title-input')).toHaveValue('가'.repeat(101));
  309 | });
  310 | 
```