import { expect, test } from '@playwright/test'

const PNG=Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==','base64')

// 여기서 내보낸 파일을 다시 들여오면 그림도 따라와야 한다. 차트는 이미 그랬다.
test('a picture rides along when a workbook goes out to XLSX and comes back', async ({ request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`그림 왕복 ${Date.now()}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  const image=await request.post(`/api/v1/sheets/${sheet}/images`,{headers:{'Idempotency-Key':`img-${workbook.id}`},multipart:{file:{name:'dot.png',mimeType:'image/png',buffer:PNG}}}).then(r=>r.json())
  expect(image.id).toBeTruthy()

  const exported=await request.post('/api/v1/exports',{data:{workbook_id:workbook.id,format:'xlsx'}})
  expect(exported.ok()).toBe(true)
  // 빠뜨린 그림이 없으면 머리글도 없다. 차트와 같은 규칙이다.
  expect(exported.headers()['x-kanpic-skipped-images']).toBeUndefined()
  const bytes=await exported.body()

  const imported=await request.post('/api/v1/imports',{headers:{'Idempotency-Key':`imp-${workbook.id}`},multipart:{file:{name:'왕복.xlsx',mimeType:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',buffer:bytes}}})
  expect(imported.ok()).toBe(true)
  const copy=await imported.json()
  const images=(await request.get(`/api/v1/workbooks/${copy.id}/images`).then(r=>r.json())).items
  expect(images).toHaveLength(1)
  expect(images[0].content_type).toBe('image/png')
  expect(images[0].natural_width).toBe(1)
  const content=await request.get(`/api/v1/images/${images[0].id}/content`)
  expect(content.status()).toBe(200)
  expect(Buffer.compare(await content.body(),PNG)).toBe(0)
})
