import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

// 병합된 칸에 붙여넣으면 병합이 어긋났다. 붙여넣은 칸만 병합을 잃고 나머지는
// 그대로 기억해서, 값은 병합 아래 숨어 보이지 않고 다음 편집은 "stored merge
// metadata is invalid" 로 거절됐다. 이제 서버가 그 병합 전체를 같은 작업에서
// 풀고, 결과에 무엇을 풀었는지 실어 격자가 말한다.
const mergeOf=(cell:{style?:Record<string,unknown>})=>cell.style?.merge as {start_row:number}|undefined

async function seed(request:APIRequestContext,title:string){
  const workbook=await request.post('/api/v1/workbooks',{data:{title}}).then(response=>response.json())
  const sheet=workbook.sheets[0].id as string
  await request.patch(`/api/v1/sheets/${sheet}/ranges:merge`,{data:{base_version:workbook.version,idempotency_key:`merge-${workbook.id}`,range:'B2:C3'}})
  return {workbook,sheet}
}

async function cellsOf(request:APIRequestContext,sheet:string,range:string){
  return (await request.get(`/api/v1/sheets/${sheet}/ranges/${range}`).then(r=>r.json())).items as Array<{row:number;column:number;value?:unknown;style?:Record<string,unknown>}>
}

async function openAt(page:Page,workbookId:string,x:number,y:number){
  await page.goto(`/workbooks/${workbookId}`)
  await expect(page.locator('.grid-canvas')).toBeVisible()
  const box=(await page.locator('.grid-canvas').boundingBox())!
  await page.mouse.click(box.x+x,box.y+y)
}

test('pasting into one cell of a merge dissolves the whole merge and says so', async ({ page, request }) => {
  const {workbook,sheet}=await seed(request,`병합 붙여넣기 ${Date.now()}`)
  // C3 (세 번째 열, 세 번째 행) — 병합의 덮인 칸.
  await openAt(page,workbook.id,80+100*2,42+20*2)
  await page.evaluate(()=>{
    const data=new DataTransfer();data.setData('text/plain','숨은 값')
    document.querySelector('.grid-viewport')?.dispatchEvent(new ClipboardEvent('paste',{clipboardData:data,bubbles:true,cancelable:true}))
  })
  await expect(page.locator('.formula-issue')).toContainText('B2:C3 병합을 풀었습니다')
  const cells=await cellsOf(request,sheet,'B2:C3')
  expect(cells.filter(mergeOf)).toEqual([])
  // 격자가 병합의 첫 칸으로 선택을 옮기므로 값은 B2 에 들어간다. 어디든 보여야 한다.
  expect(cells.some(cell=>cell.value==='숨은 값')).toBe(true)

  // 되돌리면 병합이 돌아온다. 풀린 칸 모두가 같은 작업에 들어 있기 때문이다.
  await page.keyboard.press('Control+z')
  await expect.poll(async()=>(await cellsOf(request,sheet,'B2:C3')).filter(mergeOf).length,{timeout:10_000}).toBe(4)
})

test('a fill that runs through a merge leaves no cell remembering it', async ({ page, request }) => {
  const {workbook,sheet}=await seed(request,`병합 채우기 ${Date.now()}`)
  const version=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(r=>r.json())).version
  // 자동 채우기와 같은 모양의 쓰기: B1..B4 를 쓰며 병합 B2:C3 의 앵커와 A열 쪽 칸을 지난다.
  const result=await request.patch(`/api/v1/sheets/${sheet}/cells:fill`,{data:{base_version:version,idempotency_key:`fill-${workbook.id}`,
    cells:[1,2,3,4].map(row=>({row,column:2,value:row}))}}).then(r=>r.json())
  expect(result.unmerged_ranges).toEqual(['B2:C3'])
  const cells=await cellsOf(request,sheet,'A1:D4')
  expect(cells.filter(mergeOf)).toEqual([])
  // 예전에는 C2·C3 가 사라진 병합을 기억해 다음 편집이 거절됐다. 이제 편집된다.
  const again=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(r=>r.json())).version
  const edit=await request.patch(`/api/v1/sheets/${sheet}/cells:batch`,{data:{base_version:again,idempotency_key:`edit-${workbook.id}`,cells:[{row:3,column:3,value:'편집됨'}]}})
  expect(edit.ok()).toBe(true)
})

// 저장이 끊겼다 이어져 같은 쓰기를 다시 보내면 서버는 첫 응답을 되풀이한다. 그
// 되풀이가 풀린 병합을 빠뜨리면, 되풀이만 본 격자는 병합이 사라진 것을 모른다.
test('a replayed write still says which merge it dissolved', async ({ request }) => {
  const {workbook,sheet}=await seed(request,`병합 되풀이 ${Date.now()}`)
  const version=(await request.get(`/api/v1/workbooks/${workbook.id}`).then(r=>r.json())).version
  const body={base_version:version,idempotency_key:`replay-${workbook.id}`,cells:[{row:2,column:2,value:'v'}]}
  const first=await request.patch(`/api/v1/sheets/${sheet}/cells:paste`,{data:body}).then(r=>r.json())
  const replay=await request.patch(`/api/v1/sheets/${sheet}/cells:paste`,{data:body}).then(r=>r.json())
  expect(first.unmerged_ranges).toEqual(['B2:C3'])
  expect(replay.duplicate).toBe(true)
  expect(replay.unmerged_ranges).toEqual(['B2:C3'])
})
