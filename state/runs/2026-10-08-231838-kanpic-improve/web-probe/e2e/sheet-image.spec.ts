import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

// 1×1 PNG. 바이트로 종류를 읽으므로 진짜 그림이어야 한다.
const PNG=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==','base64')

async function seed(request:APIRequestContext,title:string){
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(r=>r.json())
  return {workbook,sheet:workbook.sheets[0].id as string}
}

async function upload(request:APIRequestContext,sheet:string,key:string){
  return request.post(`/api/v1/sheets/${sheet}/images`,{headers:{'Idempotency-Key':key},multipart:{file:{name:'dot.png',mimeType:'image/png',buffer:PNG}}})
}

async function openEditor(page:Page,workbookId:string){
  await page.goto(`/workbooks/${workbookId}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.waitForTimeout(500)
}

test('an uploaded picture floats over the sheet and is served as what it is', async ({ page, request }) => {
  const {workbook,sheet}=await seed(request,`그림 ${Date.now()}`)
  const created=await upload(request,sheet,`img-${workbook.id}`)
  expect(created.status()).toBe(201)
  const image=await created.json()
  expect(image.content_type).toBe('image/png')
  expect(image.natural_width).toBe(1)

  const content=await request.get(`/api/v1/images/${image.id}/content`)
  expect(content.status()).toBe(200)
  expect(content.headers()['content-type']).toBe('image/png')
  expect(content.headers()['x-content-type-options']).toBe('nosniff')
  expect(content.headers()['content-disposition']).toBe('inline')

  await openEditor(page,workbook.id)
  const figure=page.locator('.sheet-image')
  await expect(figure).toHaveCount(1)
  await expect(figure.locator('img')).toHaveAttribute('src',new RegExp(`/api/v1/images/${image.id}/content`))

  // 같은 멱등 키로 다시 올리면 같은 그림이다.
  const replay=await upload(request,sheet,`img-${workbook.id}`).then(r=>r.json())
  expect(replay.id).toBe(image.id)
  expect((await request.get(`/api/v1/workbooks/${workbook.id}/images`).then(r=>r.json())).items).toHaveLength(1)
})

test('a picture on the clipboard is pasted as a picture, not as text', async ({ page, request }) => {
  const {workbook}=await seed(request,`그림 붙여넣기 ${Date.now()}`)
  await openEditor(page,workbook.id)
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+80,box.y+42)
  await page.evaluate(async(base64)=>{
    const bytes=Uint8Array.from(atob(base64),c=>c.charCodeAt(0))
    const file=new File([bytes],'shot.png',{type:'image/png'})
    const data=new DataTransfer();data.items.add(file);data.setData('text/plain','shot.png')
    document.querySelector('.grid-viewport')?.dispatchEvent(new ClipboardEvent('paste',{clipboardData:data,bubbles:true,cancelable:true}))
  },PNG.toString('base64'))
  await expect(page.locator('.sheet-image')).toHaveCount(1,{timeout:10_000})
  // 글자 'shot.png' 가 셀에 들어가지 않았다.
  const cells=(await request.get(`/api/v1/sheets/${workbook.sheets[0].id}/ranges/A1:A1`).then(r=>r.json())).items
  expect(cells).toHaveLength(0)
})

test('what is not a picture is refused by its bytes, whatever its name says', async ({ request }) => {
  const {sheet}=await seed(request,`가짜 그림 ${Date.now()}`)
  const svg=Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>')
  const refused=await request.post(`/api/v1/sheets/${sheet}/images`,{headers:{'Idempotency-Key':'svg-1'},multipart:{file:{name:'safe.png',mimeType:'image/png',buffer:svg}}})
  expect(refused.status()).toBe(400)
})

test('someone without access to the workbook cannot fetch its picture', async ({ request }) => {
  const owner={'X-Kanpic-Actor':'image.owner@corp.example'},stranger={'X-Kanpic-Actor':'image.stranger@corp.example'}
  const workbook=await request.post('/api/v1/workbooks',{headers:owner,data:{title:`남의 그림 ${Date.now()}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  const image=await request.post(`/api/v1/sheets/${sheet}/images`,{headers:{...owner,'Idempotency-Key':`img-${workbook.id}`},multipart:{file:{name:'dot.png',mimeType:'image/png',buffer:PNG}}}).then(r=>r.json())
  expect(image.id).toBeTruthy()
  // 공유받지 않은 다른 사용자는 본체도 목록도 못 본다. 그림은 사진일 수 있다.
  expect((await request.get(`/api/v1/images/${image.id}/content`,{headers:stranger})).status()).not.toBe(200)
  expect((await request.get(`/api/v1/workbooks/${workbook.id}/images`,{headers:stranger})).status()).not.toBe(200)
  // 주인은 본다.
  expect((await request.get(`/api/v1/images/${image.id}/content`,{headers:owner})).status()).toBe(200)
})
