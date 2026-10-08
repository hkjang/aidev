import { expect, test } from '@playwright/test'

// 열 순서를 바꾸는 일은 잘라내기·삽입으로 돌려 하기에는 너무 잦다. 머리글을
// 잡아 끌어 옮기되 수식이 따라와야 쓸 수 있는 기능이 된다.
test('dragging a selected column header moves the column and its formulas', async ({ page, request }) => {
  const workbook=await request.post('/api/v1/workbooks',{data:{title:`열 이동 ${Date.now()}`}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id
  const rows=[['이름','수량','단가'],['연필',3,500],['공책',2,1200]]
  const cells=rows.flatMap((row,rowIndex)=>row.map((value,column)=>({row:rowIndex+1,column:column+1,value})))
  cells.push({row:2,column:4,formula:'=B2*C2'} as never)
  cells.push({row:3,column:4,formula:'=B3*C3'} as never)
  await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:workbook.version,idempotency_key:`move-seed-${workbook.id}`,cells}})

  await page.goto(`/workbooks/${workbook.id}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const canvas=page.locator('.grid-canvas')
  const box=(await canvas.boundingBox())!
  // 머리글 좌표: 행 머리글 너비 뒤로 열이 이어진다.
  const headerY=box.y+14
  const columnCentre=(index:number)=>box.x+48+120*(index-0.5)
  await page.mouse.click(columnCentre(2),headerY)
  await page.mouse.move(columnCentre(2),headerY)
  await page.mouse.down()
  await page.mouse.move(columnCentre(3)+30,headerY,{steps:8})
  await page.mouse.up()

  await expect.poll(async()=>{
    const range=await request.get(`/api/v1/sheets/${sheet}/ranges/A1:D3`).then(response=>response.json())
    const at=(row:number,column:number)=>range.items.find((cell:{row:number;column:number})=>cell.row===row&&cell.column===column)
    return{header:at(1,2)?.value,formula:at(2,4)?.formula,total:at(2,4)?.value}
  },{timeout:15_000}).toEqual({header:'단가',formula:'=C2*B2',total:1500})
  await request.delete(`/api/v1/workbooks/${workbook.id}`)
})
