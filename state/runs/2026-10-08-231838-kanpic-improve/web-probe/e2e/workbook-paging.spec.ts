import { expect, test } from '@playwright/test'

// 홈 화면은 열 수 있는 워크북을 전부 받아 브라우저에서 걸러 냈다. 수천 개가
// 되면 몇 MB를 받아 카드를 전부 그린다. 이제 한 페이지씩 받고, 검색과 필터도
// 서버가 한다 — 페이지를 나누면서 검색이 없으면 첫 페이지 밖의 워크북은
// 찾을 길이 없어진다.
test('the workbook list arrives a page at a time and search reaches past it', async ({ page, request }) => {
  const stamp=Date.now()
  const made:string[]=[]
  for(let index=0;index<3;index+=1){
    const workbook=await request.post('/api/v1/workbooks',{data:{title:`쪽나눔 ${stamp} 번호 ${index}`,workspace_id:'default'}}).then(r=>r.json())
    made.push(workbook.id)
  }
  try{
    await page.goto('/')
    await expect(page.locator('article.workbook-card').first()).toBeVisible()
    const label=page.locator('.home-search span')
    await expect(label).toContainText('중')
    const shown=await page.locator('article.workbook-card').count()
    // 한 페이지는 예순 개다. 그보다 많으면 나머지는 "더 보기" 뒤에 있다.
    expect(shown).toBeLessThanOrEqual(60)

    // 이름 일부로 찾으면 첫 페이지에 없더라도 나온다.
    await page.getByLabel('워크북 검색').fill(`쪽나눔 ${stamp} 번호 2`)
    await expect.poll(()=>page.locator('article.workbook-card').count(),{timeout:10000}).toBe(1)
    await expect(page.locator('article.workbook-card strong')).toHaveText(`쪽나눔 ${stamp} 번호 2`)

    // 맞는 것이 없으면 그렇게 말한다.
    await page.getByLabel('워크북 검색').fill(`없는 이름 ${stamp}`)
    await expect(page.getByText(/와 맞는 워크북이 없습니다/)).toBeVisible()

    // 검색을 지우면 목록이 돌아온다.
    await page.getByLabel('워크북 검색').fill('')
    await expect.poll(()=>page.locator('article.workbook-card').count(),{timeout:10000}).toBe(shown)
  }finally{
    for(const id of made)await request.delete(`/api/v1/workbooks/${id}`)
  }
})
