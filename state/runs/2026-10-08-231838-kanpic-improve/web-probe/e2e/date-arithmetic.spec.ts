import { expect, test } from '@playwright/test'

// 두 날짜가 며칠 떨어져 있는지, 그날로부터 일주일 뒤가 언제인지는 어떤
// 스프레드시트에서든 가장 흔한 날짜 계산이다. kanpic 은 날짜를 직렬 번호가
// 아니라 쓰인 그대로 담아 두는데, 연산자가 그것을 몰라 둘 다 #VALUE! 였다.
test('subtracting dates gives days and adding days gives a date', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`날짜 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{idempotency_key:`date-${stamp}`,cells:[
    {row:1,column:1,value:'2026-08-23'},{row:2,column:1,value:'2026-09-01'},
  ]}})
  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B1')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.keyboard.type('=A2-A1')
  await page.keyboard.press('Enter')
  await page.keyboard.type('=A1+7')
  await page.keyboard.press('Enter')

  await expect.poll(async()=>{
    const items=(await request.get(`/api/v1/sheets/${sheet}/ranges/B1:B2`).then(r=>r.json())).items
    return items.map((cell:{value:unknown})=>cell.value)
  }).toEqual([9,'2026-08-30'])

  // 화면에도 오류가 아니라 값이 보여야 한다.
  await page.getByRole('combobox',{name:'이름 상자'}).fill('B1')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await expect(page.locator('.formula-issue')).toHaveCount(0)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
