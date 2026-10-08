import { expect, test } from '@playwright/test'

// 시트의 범위를 골라 프레젠테이션으로 만든다. kanpic 이 값의 뜻을 먼저
// 판단하고, 프레젠테이션 서비스는 그 뜻을 그림으로 만든다.
//
// 이 시험은 프레젠테이션 서비스가 실제로 붙어 있을 때만 돈다. 서비스 없이
// 도는 CI 에서는 기능이 꺼져 있고, 꺼져 있을 때 메뉴에 나오지 않는 것 자체가
// 확인할 값어치가 있는 동작이다.
test('a range becomes a deck, and the menu stays hidden when nobody set one up', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`덱 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`deck-${stamp}`,cells:[
    {row:1,column:1,value:'부서'},{row:1,column:2,value:'매출'},{row:1,column:3,value:'목표달성률'},
    {row:2,column:1,value:'영업1'},{row:2,column:2,value:'120억'},{row:2,column:3,value:'108%'},
    {row:3,column:1,value:'영업2'},{row:3,column:2,value:'95억'},{row:3,column:3,value:'91%'},
    {row:4,column:1,value:'영업3'},{row:4,column:2,value:'110억'},{row:4,column:3,value:'103%'},
  ]}})
  const configured=await request.get('/api/v1/presentation/config').then(r=>r.json())

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  const group=page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'프레젠테이션',exact:true})

  if(!configured.enabled){
    // 설정되지 않은 기능은 메뉴에 없어야 한다.
    await expect(group).toHaveCount(0)
    await request.delete(`/api/v1/workbooks/${workbook.id}`)
    return
  }

  await group.click()
  await page.getByRole('menuitem',{name:'프레젠테이션 만들기…'}).click()
  const dialog=page.getByRole('dialog',{name:'프레젠테이션 만들기'})
  await expect(dialog).toBeVisible()
  await dialog.getByLabel('프레젠테이션 제목').fill(`영업실적 ${stamp}`)

  // 범위를 어떻게 읽었는지 먼저 보여 준다. 서버가 만든 미리보기라서 여기
  // 보이는 것과 실제로 만들어지는 것이 다를 수 없다.
  await expect(dialog.getByText('이 범위를 이렇게 읽었습니다')).toBeVisible()
  await expect(dialog.locator('.presentation-slide')).not.toHaveCount(0)
  const previewed=await dialog.locator('.presentation-slide').count()

  await dialog.getByRole('button',{name:'프레젠테이션 만들기'}).click()
  await expect(dialog.getByRole('status')).toContainText(`${previewed}장을 만들었습니다`,{timeout:30000})

  // 만든 덱은 워크북에 매여 있다. 그래야 볼 수 있는 사람만 내려받는다.
  const records=await request.get(`/api/v1/workbooks/${workbook.id}/presentations`).then(r=>r.json())
  expect(records.items).toHaveLength(1)
  expect(records.items[0]).toMatchObject({range:'A1:C4',stale:false})

  const download=await request.get(`/api/v1/presentations/${records.items[0].id}/export`)
  expect(download.status()).toBe(200)
  expect(download.headers()['content-type']).toContain('presentationml')
  expect((await download.body()).byteLength).toBeGreaterThan(10_000)

  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 덱을 만든 뒤 원본이 바뀌면 덱은 옛말을 한다. 목록이 그것을 말해 주고,
// 다시 만들기는 **같은 덱** 을 지금 값으로 고쳐 쓴다 — 새 덱을 만들면 이미
// 보낸 링크가 계속 옛 숫자를 보여 준다.
test('the panel says when a deck has fallen behind, and refreshes it in place', async ({ page, request }) => {
  const configured=await request.get('/api/v1/presentation/config').then(r=>r.json())
  test.skip(!configured.enabled,'프레젠테이션 서비스가 설정되지 않았습니다')

  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`새로고침 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`seed-${stamp}`,cells:[
    {row:1,column:1,value:'부서'},{row:1,column:2,value:'매출'},
    {row:2,column:1,value:'영업1'},{row:2,column:2,value:120},
    {row:3,column:1,value:'영업2'},{row:3,column:2,value:95},
    {row:4,column:1,value:'영업3'},{row:4,column:2,value:110},
  ]}})
  const made=await request.post(`/api/v1/sheets/${sheet}/presentations`,{data:{range:'A1:B4',title:`실적 ${stamp}`}}).then(r=>r.json())
  const deckId=made.presentation.id as string

  const current=await request.get(`/api/v1/workbooks/${workbook.id}`).then(r=>r.json())
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:current.version,idempotency_key:`bump-${stamp}`,cells:[{row:3,column:2,value:400}]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  await page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'프레젠테이션',exact:true}).click()
  await page.getByRole('menuitem',{name:'만든 프레젠테이션 목록'}).click()

  const card=page.locator('.presentation-panel-list article')
  await expect(card).toHaveCount(1)
  await expect(card.getByText('원본 변경됨')).toBeVisible()

  await card.getByRole('button',{name:/다시 만들기/}).click()
  await expect(card.getByText('원본 변경됨')).toHaveCount(0,{timeout:30000})

  // 같은 덱이어야 하고, 새 숫자가 들어가 있어야 한다.
  const records=await request.get(`/api/v1/workbooks/${workbook.id}/presentations`).then(r=>r.json())
  expect(records.items).toHaveLength(1)
  expect(records.items[0].id).toBe(deckId)
  expect(records.items[0].stale).toBe(false)

  const download=await request.get(`/api/v1/presentations/${deckId}/export`)
  expect(download.status()).toBe(200)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 잴 것이 없는 표 — 로드맵, 절차, 일정 — 도 슬라이드로 만들 값어치가 있다.
// 그 값어치는 순서에 있으므로 표로 떨어뜨리면 잃는다.
test('a roadmap becomes a process, not a table', async ({ page, request }) => {
  const configured=await request.get('/api/v1/presentation/config').then(r=>r.json())
  test.skip(!configured.enabled,'프레젠테이션 서비스가 설정되지 않았습니다')

  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`로드맵 ${stamp}`}}).then(r=>r.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`road-${stamp}`,cells:[
    {row:1,column:1,value:'단계'},{row:1,column:2,value:'내용'},{row:1,column:3,value:'기한'},
    {row:2,column:1,value:'준비'},{row:2,column:2,value:'조직·예산 확정'},{row:2,column:3,value:'2026-07'},
    {row:3,column:1,value:'이행'},{row:3,column:2,value:'1차 이관'},{row:3,column:3,value:'2026-10'},
    {row:4,column:1,value:'안정화'},{row:4,column:2,value:'운영 이관'},{row:4,column:3,value:'2026-11'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('menubar',{name:'워크북 메뉴'}).getByRole('menuitem',{name:'데이터',exact:true}).click()
  await page.getByRole('menu',{name:'데이터 메뉴'}).getByRole('menuitem',{name:'프레젠테이션',exact:true}).click()
  await page.getByRole('menuitem',{name:'프레젠테이션 만들기…'}).click()
  const dialog=page.getByRole('dialog',{name:'프레젠테이션 만들기'})
  await expect(dialog).toBeVisible()

  // 숫자가 없어도 머리글은 머리글이다. 자료로 삼키면 "단계" 가 첫 단계가 된다.
  await expect(dialog.getByText(/3행/)).toBeVisible()
  await expect(dialog.locator('.presentation-slide',{hasText:'진행 순서'})).toBeVisible()
  await expect(dialog.locator('.presentation-component span',{hasText:'steps'}).first()).toBeVisible()
  // 표지의 첫 줄은 개수가 아니라 어디서 어디까지인지다.
  await expect(dialog.locator('.presentation-slide').first()).toContainText('준비부터 안정화까지 3단계입니다.')

  await dialog.getByRole('button',{name:'프레젠테이션 만들기'}).click()
  await expect(dialog.getByRole('status')).toContainText('만들었습니다',{timeout:30000})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
