import { cleanup,fireEvent,render,screen,waitFor } from '@testing-library/react'
import { afterEach,describe,expect,it,vi } from 'vitest'
import { PresentationPanel,type PresentationRecord } from './PresentationPanel'

const record=(overrides:Partial<PresentationRecord>={}):PresentationRecord=>({
  id:'deck-1',provider:'ptium',workbook_id:'wb-1',sheet_id:'sheet-1',range:'A1:D4',
  source_version:7,title:'2026년 영업실적',template:'기본',slide_count:5,
  edit_url:'http://presentation.invalid/d/1',created_by:'alice',
  created_at:'2026-08-23T00:00:00Z',updated_at:'2026-08-23T00:00:00Z',stale:false,...overrides,
})
const sheets=new Map([['sheet-1','실적']])

afterEach(cleanup)

describe('PresentationPanel',()=>{
  it('marks only the decks whose source has moved on',()=>{
    render(<PresentationPanel items={[record(),record({id:'deck-2',title:'요약',stale:true})]} sheetNames={sheets}
      onClose={vi.fn()} onCreate={vi.fn()} onRefresh={vi.fn()} onDownload={vi.fn()}/>)
    expect(screen.getAllByText('원본 변경됨')).toHaveLength(1)
    expect(screen.getAllByText('실적!A1:D4 · 5장 · 기본')).toHaveLength(2)
    // 어느 워크북 버전을 기준으로 만든 것인지 보여야 무엇과 비교된 것인지 안다.
    expect(screen.getAllByText(/워크북 v7 기준/).length).toBe(2)
  })

  it('refreshes a deck and says so while it works',async()=>{
    let release=()=>{}
    const refresh=vi.fn().mockImplementation(()=>new Promise<void>(resolve=>{release=()=>resolve()}))
    render(<PresentationPanel items={[record({stale:true})]} sheetNames={sheets}
      onClose={vi.fn()} onCreate={vi.fn()} onRefresh={refresh} onDownload={vi.fn()}/>)
    fireEvent.click(screen.getByRole('button',{name:/다시 만들기/}))
    await screen.findByText('다시 만드는 중…')
    expect(refresh).toHaveBeenCalledWith(expect.objectContaining({id:'deck-1'}))
    release()
    await waitFor(()=>expect(screen.queryByText('다시 만드는 중…')).toBeNull())
  })

  // 실패를 조용히 삼키면 사람은 다시 만들어진 줄 알고 옛 덱을 보낸다.
  it('shows why a refresh failed',async()=>{
    render(<PresentationPanel items={[record({stale:true})]} sheetNames={sheets}
      onClose={vi.fn()} onCreate={vi.fn()} onRefresh={async()=>{throw new Error('원본 시트를 찾을 수 없습니다.')}} onDownload={vi.fn()}/>)
    fireEvent.click(screen.getByRole('button',{name:/다시 만들기/}))
    expect(await screen.findByRole('alert')).toHaveTextContent('원본 시트를 찾을 수 없습니다.')
    // 실패했으면 표시는 그대로 남아야 한다.
    expect(screen.getByText('원본 변경됨')).toBeTruthy()
  })

  it('names a sheet that is no longer there',()=>{
    render(<PresentationPanel items={[record({sheet_id:'gone'})]} sheetNames={sheets}
      onClose={vi.fn()} onCreate={vi.fn()} onRefresh={vi.fn()} onDownload={vi.fn()}/>)
    expect(screen.getByText(/삭제된 시트!A1:D4/)).toBeTruthy()
  })
})
