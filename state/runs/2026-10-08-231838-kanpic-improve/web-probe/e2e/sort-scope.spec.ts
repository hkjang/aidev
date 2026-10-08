import { expect, test } from '@playwright/test'

// 정렬은 행을 옮긴다. 옮기는 범위가 표보다 좁으면 값이 남의 행과 짝지어지고,
// 화면에 불러온 행까지만 옮기면 표의 절반만 정렬된 채 남는다.
test('sorting a column covers the whole table, not the loaded window', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`정렬 범위 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  let version=workbook.version
  let batch:Array<Record<string,unknown>>=[{row:1,column:1,value:'코드'},{row:1,column:2,value:'짝'}]
  for(let row=2;row<=201;row+=1){
    const value=202-row
    batch.push({row,column:1,value},{row,column:2,value:`짝-${value}`})
    if(batch.length>=200){
      const result=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:version,idempotency_key:`ss-${stamp}-${row}`,cells:batch}}).then(response=>response.json())
      version=result.server_version;batch=[]
    }
  }
  if(batch.length>0){
    const result=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:version,idempotency_key:`ss-${stamp}-last`,cells:batch}}).then(response=>response.json())
    version=result.server_version
  }

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+48+53,box.y+14,{button:'right'})
  await page.getByRole('menuitem',{name:/오름차순 정렬/}).click()
  const dialog=page.getByRole('dialog',{name:'정렬 범위 확인'})
  // 화면에 보이는 20여 행이 아니라 표 전체가 대상이어야 한다.
  await expect(dialog).toContainText('A1:B201')
  await expect(dialog).toContainText('200행이 정렬됩니다.')
  await dialog.getByRole('button',{name:'정렬'}).click()

  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A2:B201`).then(response=>response.json())
    const byRow=new Map<string,unknown>()
    for(const cell of range.items)byRow.set(`${cell.row}:${cell.column}`,cell.value)
    let ascending=true,misaligned=0
    for(let row=2;row<=201;row+=1){
      const code=byRow.get(`${row}:1`)
      if(row>2&&Number(code)<Number(byRow.get(`${row-1}:1`)))ascending=false
      if(byRow.get(`${row}:2`)!==`짝-${code}`)misaligned+=1
    }
    return {ascending,misaligned,first:byRow.get('2:1'),last:byRow.get('201:1')}
  },{timeout:20_000}).toEqual({ascending:true,misaligned:0,first:1,last:200})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})

test('sorting part of a table warns that the other columns stay put', async ({ page, request }) => {
  const stamp=Date.now()
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`정렬 선택 ${stamp}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['이름','부서','점수'],['최민','개발',80],['박지민','영업',90],['이서준','기획',75]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`sn-seed-${stamp}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  await page.getByRole('combobox',{name:'이름 상자'}).fill('A2:A4')
  await page.getByRole('combobox',{name:'이름 상자'}).press('Enter')
  await page.getByRole('menuitem',{name:'데이터'}).click()
  await page.getByRole('menuitem',{name:'정렬',exact:true}).click()
  await page.getByRole('menuitem',{name:'선택 열 기준 정렬 A → Z'}).click()
  const dialog=page.getByRole('dialog',{name:'정렬 범위 확인'})
  await expect(dialog.getByLabel('표 전체 정렬')).toBeChecked()
  await expect(dialog.getByText('다른 열은 그대로 있어 값의 짝이 어긋납니다.')).toBeVisible()
  await dialog.getByRole('button',{name:'정렬'}).click()

  // 기본값(표 전체)이므로 부서·점수가 이름을 따라 움직인다.
  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A2:C4`).then(response=>response.json())
    const at=(row:number,column:number)=>range.items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)?.value
    return [at(2,1),at(2,2),at(2,3),at(4,1),at(4,2),at(4,3)]
  },{timeout:15_000}).toEqual(['박지민','영업',90,'최민','개발',80])
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
