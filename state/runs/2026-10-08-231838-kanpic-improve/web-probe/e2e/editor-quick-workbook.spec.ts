import { expect, test } from '@playwright/test'

// 편집기의 빠른 이동은 워크북 전체를 받아 놓고 앞의 스무 개만 보여 주고
// 있었다. 그래서 워크북이 많은 곳에서는 열 때마다 큰 목록을 받으면서도
// 스물한 번째부터는 이름을 적어도 찾히지 않았다.
//
// 이제 앞의 스무 개만 받고, 그 밖의 것은 적은 이름으로 서버에 묻는다.
test('quick switch finds a workbook that is not in the first page', async ({ page, request }) => {
  const stamp=Date.now()
  // 목록 앞쪽을 채운 뒤, 마지막에 찾을 워크북을 만든다. 최근 순으로 오므로
  // 방금 만든 것이 앞에 오도록 먼저 대상부터 만들고 뒤를 채운다.
  const target=await request.post('/api/v1/workbooks',{data:{title:`찾을워크북 ${stamp}`}}).then(r=>r.json())
  const filler:string[]=[]
  for(let index=0;index<24;index+=1){
    const made=await request.post('/api/v1/workbooks',{data:{title:`채움 ${stamp} ${index}`}}).then(r=>r.json())
    filler.push(made.id)
  }
  const host=await request.post('/api/v1/workbooks',{data:{title:`편집중 ${stamp}`}}).then(r=>r.json())

  await page.goto(`/workbooks/${host.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.keyboard.press('ControlOrMeta+k')
  const box=page.getByPlaceholder(/워크북/)
  await expect(box).toBeVisible()

  // 앞의 스무 개 밖에 있는 워크북을 이름으로 찾아 연다.
  await box.fill(`찾을워크북 ${stamp}`)
  await expect(page.getByRole('option',{name:new RegExp(`찾을워크북 ${stamp}`)})).toBeVisible()
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL(new RegExp(`/workbooks/${target.id}`))

  for(const id of [...filler,target.id,host.id])await request.delete(`/api/v1/workbooks/${id}`)
})
