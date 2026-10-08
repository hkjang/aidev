import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

const put=(request:APIRequestContext,key:string,value:unknown,value_type:'string'|'number'|'boolean')=>
  request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type}})

async function reset(request:APIRequestContext){
  await put(request,'analytics.enabled',false,'boolean')
  await put(request,'analytics.provider','none','string')
  await put(request,'analytics.custom_snippet','','string')
  await put(request,'analytics.measurement_id','','string')
  await put(request,'analytics.include_admin',false,'boolean')
  await put(request,'analytics.placement','head','string')
}
test.afterEach(async ({ request }) => { await reset(request) })

// Collects the browser complaints that a blocked script would produce.
function watchViolations(page:Page){
  const violations:string[]=[]
  page.on('console',message=>{if(/Content Security Policy|Refused to/i.test(message.text()))violations.push(message.text())})
  page.on('pageerror',error=>violations.push(String(error)))
  return violations
}

test('a pasted tracking snippet runs on visitor pages under the strict policy', async ({ page, request }) => {
  await put(request,'analytics.provider','custom','string')
  await put(request,'analytics.custom_snippet','<script>window.__kanpicTracked=(window.__kanpicTracked||0)+1</script>','string')
  await put(request,'analytics.enabled',true,'boolean')

  const violations=watchViolations(page)
  await page.goto('/')
  await expect.poll(()=>page.evaluate(()=>(window as unknown as {__kanpicTracked?:number}).__kanpicTracked??0)).toBe(1)
  expect(violations).toEqual([])

  // The nonce in the page matches the one the policy allows, which is what lets
  // an inline snippet run without weakening the policy.
  const response=await request.get('/')
  const policy=response.headers()['content-security-policy']
  const nonce=/nonce="([^"]+)"/.exec(await response.text())?.[1]
  expect(nonce).toBeTruthy()
  expect(policy).toContain(`'nonce-${nonce}'`)
  expect(policy).not.toContain('unsafe-inline')

  // The console is left alone unless an administrator asks for it.
  await page.goto('/admin?tab=analytics')
  expect(await page.evaluate(()=>(window as unknown as {__kanpicTracked?:number}).__kanpicTracked??0)).toBe(0)
  await put(request,'analytics.include_admin',true,'boolean')
  await page.goto('/admin?tab=analytics')
  await expect.poll(()=>page.evaluate(()=>(window as unknown as {__kanpicTracked?:number}).__kanpicTracked??0)).toBe(1)
})

test('the console configures a hosted provider and previews the generated code', async ({ page, request }) => {
  await reset(request)
  await page.goto('/admin?tab=analytics')
  await expect(page.getByRole('heading',{name:'방문자 추적'})).toBeVisible()

  await page.getByRole('radio',{name:/Google Analytics 4/}).click()
  await page.getByLabel('측정 ID').fill('G-E2E12345')
  await page.getByLabel('측정 ID').blur()
  await page.getByLabel('추적 코드 삽입').click()
  await expect(page.getByLabel('추적 코드 삽입')).toBeChecked()
  await expect(page.locator('.tracking-preview')).toContainText('gtag/js?id=G-E2E12345')

  // The generated snippet and the vendor origin reach the served page. The
  // preview above only proves what the console thinks; the checkbox saves in
  // the background, so the served page is asked until it agrees.
  await expect.poll(async()=>(await request.get('/workbooks/none')).text()).toContain('https://www.googletagmanager.com/gtag/js?id=G-E2E12345')
  const response=await request.get('/workbooks/none')
  expect(response.headers()['content-security-policy']).toContain('https://www.googletagmanager.com')

  // An incomplete configuration is reported rather than silently doing nothing.
  await page.getByLabel('측정 ID').fill('')
  await page.getByLabel('측정 ID').blur()
  // 검사는 서버가 들고 있는 설정을 본다. 지우기가 아직 날아가는 중이면
  // 예전 값으로 검사해 "문제 없음" 이라 답하고, 곧이어 도착한 저장 메시지가
  // 그 자리를 덮는다. 서버가 실제로 비워진 뒤에 검사한다.
  await expect.poll(async()=>(await request.get('/workbooks/none')).text()).not.toContain('googletagmanager')
  await page.getByRole('button',{name:'설정 확인'}).click()
  await expect(page.locator('.result-banner')).toContainText('analytics.measurement_id')
})

test('tracking stays out of the page until it is switched on', async ({ request }) => {
  await put(request,'analytics.provider','ga4','string')
  await put(request,'analytics.measurement_id','G-OFF','string')
  const response=await request.get('/')
  expect(await response.text()).not.toContain('googletagmanager')
  expect(response.headers()['content-security-policy']).toBe(
    "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
})
