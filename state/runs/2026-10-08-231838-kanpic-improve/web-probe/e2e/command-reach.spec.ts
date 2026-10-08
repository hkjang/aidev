import { expect, test } from '@playwright/test'

// 기능이 있어도 손이 닿는 곳에 없으면 없는 것과 같다. 최근에 더한 데이터
// 작업들이 셀 오른쪽 클릭과 명령 팔레트 양쪽에서 잡혀야 한다.
test('recent data commands are reachable from the cell menu and the palette', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`메뉴 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','매출'],['부산',80],['부산',95],['서울',120]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`cr-seed-${stamp}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+48+53,box.y+30+40,{button:'right'})
  const cellMenu=page.getByLabel('셀 메뉴')
  await cellMenu.getByRole('menuitem',{name:'데이터'}).click()
  for(const label of ['부분합…','중복 항목 삭제…','텍스트를 열로 분할…'])
    await expect(page.getByRole('menuitem',{name:label})).toBeVisible()

  // 셀 메뉴에서 부분합을 고르면 그 대화상자가 열린다.
  await page.getByRole('menuitem',{name:'부분합…'}).click()
  await expect(page.getByRole('dialog',{name:'부분합'})).toBeVisible()
  await page.getByRole('dialog',{name:'부분합'}).getByRole('button',{name:'취소'}).click()

  // 팔레트에서도 같은 명령을 찾을 수 있다.
  await page.keyboard.press('Control+k')
  const palette=page.getByRole('dialog',{name:'빠른 이동'})
  await palette.getByRole('searchbox').or(palette.getByRole('textbox')).first().fill('부분합')
  await expect(palette.getByRole('option',{name:'부분합 제거'})).toBeVisible()
  await palette.getByRole('option',{name:'부분합',exact:true}).click()
  await expect(page.getByRole('dialog',{name:'부분합'})).toBeVisible()
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
