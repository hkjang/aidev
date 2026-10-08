import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

const put=(request:APIRequestContext,key:string,value:unknown,value_type:'string'|'number'|'boolean')=>
  request.put(`/api/v1/admin/settings/${key}`,{data:{key,value,value_type}})

test.afterEach(async ({ request }) => {
  await put(request,'analytics.enabled',false,'boolean')
  await put(request,'analytics.provider','none','string')
  await put(request,'analytics.custom_snippet','','string')
  await put(request,'analytics.allowed_hosts','','string')
  await request.delete('/api/v1/admin/analytics/violations')
})

function watchViolations(page:Page){
  const violations:string[]=[]
  page.on('console',message=>{if(/Content Security Policy|Refused to/i.test(message.text()))violations.push(message.text())})
  return violations
}

// A tracker loads from its own host and posts its events back there. Both have
// to work from the pasted code alone.
test('the addresses a pasted tracker uses are allowed without extra setup', async ({ page, request }) => {
  await put(request,'analytics.provider','custom','string')
  await put(request,'analytics.custom_snippet',
    `<script>window.__sent=fetch("https://momento.corp.example/collect/v1/events",{method:"POST",body:"{}"}).then(()=>"sent").catch(error=>String(error))</script>`,'string')
  await put(request,'analytics.enabled',true,'boolean')

  const violations=watchViolations(page)
  await page.goto('/')
  // The request is allowed by the policy; it fails at DNS because the host is
  // imaginary, which is a different failure from being refused.
  const outcome=await page.evaluate(()=>(window as unknown as {__sent:Promise<string>}).__sent)
  expect(outcome).not.toContain('Content Security Policy')
  expect(violations).toEqual([])

  const policy=(await request.get('/')).headers()['content-security-policy']
  expect(policy).toContain('connect-src \'self\' ws: wss: https://momento.corp.example')
  expect(policy).toContain('report-uri /api/v1/analytics/csp-report')
})

// Anything the snippet does not spell out is reported and can be allowed from
// the console in one click.
test('a blocked address is reported and allowed from the console', async ({ page, request }) => {
  await put(request,'analytics.provider','custom','string')
  // The endpoint is assembled at runtime, so no scan could have found it.
  await put(request,'analytics.custom_snippet',
    `<script>var host="momento";window.__sent=fetch("https"+"://"+host+".hidden.example/collect",{method:"POST"}).then(()=>"sent").catch(error=>String(error))</script>`,'string')
  await put(request,'analytics.enabled',true,'boolean')

  await page.goto('/')
  await expect.poll(async()=>{
    const items=(await (await request.get('/api/v1/admin/analytics/violations')).json()).items as Array<{origin:string;allowed:boolean}>
    return items.map(item=>item.origin)
  },{timeout:10_000}).toContain('https://momento.hidden.example')

  await page.goto('/admin?tab=analytics')
  const row=page.locator('.violation-row',{hasText:'momento.hidden.example'})
  await expect(row).toBeVisible()
  await row.getByRole('button',{name:'이 도메인 허용'}).click()
  await expect(row.getByText('허용됨')).toBeVisible()

  // The policy now carries the origin, so the request stops being refused.
  const policy=(await request.get('/')).headers()['content-security-policy']
  expect(policy).toContain('https://momento.hidden.example')
  const violations=watchViolations(page)
  await page.goto('/')
  await page.evaluate(()=>(window as unknown as {__sent:Promise<string>}).__sent)
  expect(violations).toEqual([])
})
