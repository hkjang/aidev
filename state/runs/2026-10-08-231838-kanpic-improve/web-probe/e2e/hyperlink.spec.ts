import { expect, test } from '@playwright/test'

// 링크가 든 셀은 눌러서 열 수 있어야 하고, 셀 텍스트는 남의 워크북에서도
// 오므로 아무 주소나 열어 주면 안 된다.
test('a linked cell offers its target and refuses an unsafe one', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`링크 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`hl-seed-${stamp}`,cells:[
    {row:2,column:2,formula:'=HYPERLINK("https://example.com/guide","가이드 열기")'},
    {row:3,column:2,value:'https://example.com/report'},
    {row:4,column:2,formula:'=HYPERLINK("javascript:alert(1)","눌러보세요")'},
  ]}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const select=async(address:string)=>{
    await page.getByRole('combobox',{name:'이름 상자'}).fill(address)
    await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  }
  await select('B2')
  const chip=page.locator('.cell-link a')
  await expect(chip).toHaveAttribute('href','https://example.com/guide')
  await expect(chip).toHaveAttribute('target','_blank')

  // 그냥 붙여넣은 주소도 링크로 취급한다.
  await select('B3')
  await expect(chip).toHaveAttribute('href','https://example.com/report')

  // 스크립트 주소는 열지 않는다.
  await select('B4')
  await expect(chip).toHaveCount(0)
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
