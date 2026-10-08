import { expect, test } from '@playwright/test'

// 부분합은 그룹마다 소계 행을 끼워 넣고 아래 행을 밀어낸다. 끼워 넣은 자리에
// 원래 있던 값이 남거나, 전체 합계가 소계를 다시 더하면 조용히 틀린 표가 된다.
test('subtotals total each group, fold it away, and do not double count', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`부분합 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','제품','매출'],['부산','공책',80],['부산','연필',95],['서울','연필',120],['서울','공책',150],['서울','지우개',40]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`sb-seed-${stamp}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A2')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'부분합…'}).click()

  const dialog=page.getByRole('dialog',{name:'부분합'})
  await expect(dialog.locator('.subtotal-summary')).toContainText('2개 그룹 · 3개 행이 추가됩니다.')
  await dialog.getByRole('button',{name:'부분합 넣기'}).click()

  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A1:C9`).then(response=>response.json())
    const at=(row:number,column:number)=>range.items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)
    return {
      busanLabel:at(4,1)?.value,busanTotal:at(4,3)?.value,
      // 소계 행이 앉은 자리에 이전 값이 남으면 안 된다.
      busanStale:at(4,2)?.value,
      seoulTotal:at(8,3)?.value,
      grand:at(9,3)?.value,grandFormula:at(9,3)?.formula,
    }
  },{timeout:20_000}).toEqual({
    busanLabel:'부산 합계',busanTotal:175,busanStale:undefined,
    seoulTotal:310,grand:485,grandFormula:'=SUBTOTAL(109,C2:C3,C5:C7)',
  })

  // 각 그룹은 소계 행 뒤로 접을 수 있도록 그룹으로 묶인다.
  // 그룹은 소계 값을 쓴 뒤에 하나씩 만들어지므로 값 확인만으로는 아직 이르다.
  await expect.poll(async()=>{
    const latest=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
    const groups=latest.sheets[0].layout.row_groups??[]
    return groups.map((group:{start:number;end:number})=>`${group.start}-${group.end}`).sort()
  },{timeout:20_000}).toEqual(['2-3','5-7'])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

// 넣을 수만 있고 지울 수 없으면 표는 부분합이 박힌 채로 남는다. 되돌리면
// 원래 표와 글자 하나까지 같아야 한다.
test('removing subtotals restores the table and drops the folds', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`부분합 제거 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['지역','제품','매출'],['부산','공책',80],['부산','연필',95],['서울','연필',120],['서울','공책',150]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`rb-seed-${stamp}`,cells}})
  const original=await request.get(`/api/v1/sheets/${sheet}/ranges/A1:C5`).then(response=>response.json())
  const snapshot=(items:Array<{row:number;column:number;value:unknown}>)=>
    items.filter(cell=>cell.value!==undefined).sort((first,second)=>first.row-second.row||first.column-second.column)
      .map(cell=>`${cell.row}:${cell.column}:${String(cell.value)}`)

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A2')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'부분합…'}).click()
  await page.getByRole('dialog',{name:'부분합'}).getByRole('button',{name:'부분합 넣기'}).click()
  await expect.poll(async()=>{
    const latest=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
    return (latest.sheets[0].layout.row_groups??[]).length
  },{timeout:20_000}).toBe(2)

  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'데이터 정리'}).click()
  await page.getByRole('menuitem',{name:'부분합 제거…'}).click()
  const dialog=page.getByRole('dialog',{name:'부분합 제거'})
  await expect(dialog.locator('.cleanup-summary')).toContainText('부분합 3개 행을 지우고 그룹을 풉니다.')
  await dialog.getByRole('button',{name:'삭제'}).click()

  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A1:C9`).then(response=>response.json())
    const latest=await request.get(`/api/v1/workbooks/${workbook.id}`).then(response=>response.json())
    return {cells:snapshot(range.items),groups:(latest.sheets[0].layout.row_groups??[]).length}
  },{timeout:20_000}).toEqual({cells:snapshot(original.items),groups:0})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
