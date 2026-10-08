import { expect, test } from '@playwright/test'

// 버전 목록은 워크북의 이야기를 하지만, 사람들이 실제로 묻는 것은 "이 숫자
// 누가 바꿨어"다. 셀 하나의 기록이 그 답이어야 한다.
test('a cell reports who changed it and what it said before', async ({ page, request }) => {
  const stamp=Date.now()
  const editor=`history-editor-${stamp}`
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`편집 기록 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  await request.put(`/api/v1/workbooks/${workbook.id}/shares`,{data:{principal_type:'user',principal_id:editor,role:'editor'}})
  let version=workbook.version
  const write=async(cells:unknown[],key:string,actor?:string)=>{
    const result=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{
      headers:actor?{'X-Kanpic-Actor':actor}:{},
      data:{base_version:version,idempotency_key:key,cells},
    }).then(response=>response.json())
    version=result.server_version
    return result
  }
  await write([{row:2,column:2,value:100}],`h1-${stamp}`)
  await write([{row:2,column:2,value:250}],`h2-${stamp}`,editor)
  // 같은 값을 다시 쓴 작업은 기록에 남을 편집이 아니다.
  await write([{row:2,column:2,value:250}],`h3-${stamp}`,editor)
  await write([{row:2,column:2,formula:'=125*3'}],`h4-${stamp}`)

  const history=await request.get(`/api/v1/sheets/${sheet}/cells/B2/history`).then(response=>response.json())
  expect(history.items).toHaveLength(3)
  expect(history.items[0]).toMatchObject({after:{formula:'=125*3'},before:{value:250}})
  expect(history.items[1]).toMatchObject({actor_id:editor,before:{value:100},after:{value:250}})
  expect(history.items[2].before.empty).toBe(true)

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+48+107*1.5,box.y+30+27*1.5,{button:'right'})
  await page.getByRole('menuitem',{name:'편집 기록 표시'}).click()
  const dialog=page.getByRole('dialog',{name:'B2 편집 기록'})
  await expect(dialog.getByText(editor)).toBeVisible()
  await expect(dialog.getByText('=125*3')).toBeVisible()
  await expect(dialog.getByText('빈 셀')).toBeVisible()
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
