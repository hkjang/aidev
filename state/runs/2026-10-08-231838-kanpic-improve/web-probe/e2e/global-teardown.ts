import { request } from '@playwright/test'

/**
 * 대부분의 시나리오는 워크북을 만들고 지우지 않는다. 한 번 돌 때마다 수십
 * 개가 남고, 오늘 하루 반복해 돌린 끝에 6,004개가 쌓였다. 홈 화면 목록을
 * 다루는 시나리오는 그만큼 느려지고 흔들린다. 실제로 정리한 뒤 전체 실행
 * 시간이 4분 30초에서 3분 12초로 줄었다.
 *
 * 이 실행이 만든 것만 지운다. 시작 시각 이후에 생긴 워크북이 대상이므로,
 * 같은 서버에 있던 다른 데이터는 건드리지 않는다.
 */
export default async function globalTeardown(){
  const startedAt=process.env.KANPIC_E2E_STARTED_AT
  if(!startedAt)return
  const context=await request.newContext({baseURL:process.env.KANPIC_E2E_BASE_URL||'http://localhost:8080'})
  try{
    const since=new Date(startedAt).getTime()
    // 삭제하면 목록이 줄어드므로, 지울 것이 없을 때까지 반복해 읽는다.
    for(let pass=0;pass<50;pass+=1){
      const response=await context.get('/api/v1/workbooks')
      if(!response.ok())return
      const items=((await response.json()).items??[]) as Array<{id:string;created_at?:string}>
      const mine=items.filter(item=>item.created_at&&new Date(item.created_at).getTime()>=since)
      if(mine.length===0)break
      for(const item of mine)await context.delete(`/api/v1/workbooks/${item.id}`)
    }
    // 삭제는 휴지통으로 옮길 뿐이다. 시나리오가 스스로 지운 것까지 포함해
    // 이 실행이 남긴 휴지통을 비운다. 그러지 않으면 휴지통을 다루는
    // 시나리오가 자기 항목을 수백 개 사이에서 찾지 못한다.
    for(let pass=0;pass<50;pass+=1){
      const response=await context.get('/api/v1/workbooks/trash')
      if(!response.ok())return
      const items=((await response.json()).items??[]) as Array<{id:string;created_at?:string}>
      const mine=items.filter(item=>item.created_at&&new Date(item.created_at).getTime()>=since)
      if(mine.length===0)return
      for(const item of mine)await context.delete(`/api/v1/workbooks/${item.id}/purge`)
    }
  }catch{
    // 정리는 거들 뿐이다. 실패해도 시나리오 결과를 뒤집지 않는다.
  }finally{
    await context.dispose()
  }
}
