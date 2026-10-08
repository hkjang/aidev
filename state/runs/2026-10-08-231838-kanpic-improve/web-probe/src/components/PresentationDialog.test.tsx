import { cleanup,fireEvent,render,screen,waitFor } from '@testing-library/react'
import { afterEach,describe,expect,it,vi } from 'vitest'
import { PresentationDialog,type PresentationAnalysis,type PresentationDeck } from './PresentationDialog'

const deck:PresentationDeck={title:'부서별 매출',subtitle:'영업2가 목표에 못 미칩니다.',slides:[
  {kind:'cover',title:'부서별 매출',lead:'영업2가 목표에 못 미칩니다.'},
  {kind:'content',title:'핵심 지표',component:{kind:'kpi',caption:'주요 지표',rows:[{label:'목표 미달',fields:['91%','영업2']}]}},
]}
const analysis:PresentationAnalysis={shape:'categories',chart:'bars',row_count:3,has_header:true,headline:'영업2가 목표에 못 미칩니다.',
  columns:[{name:'부서',kind:'text',role:'dimension'},{name:'매출',kind:'number',role:'measure'}]}

const range={startRow:1,startColumn:1,endRow:4,endColumn:2}

afterEach(()=>{cleanup();vi.unstubAllGlobals()})

describe('PresentationDialog',()=>{
  it('shows what kanpic read before anything is made',async()=>{
    const preview=vi.fn().mockResolvedValue({deck,analysis})
    const create=vi.fn()
    render(<PresentationDialog range={range} onClose={vi.fn()} onPreview={preview} onCreate={create} onLoadTemplates={async()=>[]} onDownload={vi.fn()}/>)
    await waitFor(()=>expect(preview).toHaveBeenCalled())
    expect(preview.mock.calls[0][0]).toMatchObject({range:'A1:B4',preview:true,include_table:true})
    await screen.findByText('항목별 비교 · 막대 차트 · 3행')
    // 열이 무슨 노릇을 하는지 보여야 사람이 잘못 읽었는지 알아차린다.
    expect(screen.getByText('부서')).toBeTruthy()
    expect(screen.getAllByText('항목').length).toBeGreaterThan(0)
    expect(await screen.findByText('핵심 지표')).toBeTruthy()
    // 미리보기만으로는 아무것도 만들어지지 않는다.
    expect(create).not.toHaveBeenCalled()
  })

  it('makes the deck and then offers the file',async()=>{
    const create=vi.fn().mockResolvedValue({presentation:{id:'deck-1',title:'부서별 매출',status:'completed',slide_count:5,template:'기본',edit_url:'http://presentation.invalid/d/1',warnings:['마지막 장의 글이 줄었습니다']}})
    const download=vi.fn().mockResolvedValue(undefined)
    render(<PresentationDialog range={range} onClose={vi.fn()} onPreview={async()=>({deck,analysis})} onCreate={create} onLoadTemplates={async()=>[{id:'t1',name:'푸른 표지'}]} onDownload={download}/>)
    await screen.findByText('핵심 지표')
    fireEvent.change(await screen.findByLabelText('프레젠테이션 템플릿'),{target:{value:'t1'}})
    fireEvent.click(screen.getByRole('button',{name:'프레젠테이션 만들기'}))
    await waitFor(()=>expect(create).toHaveBeenCalled())
    expect(create.mock.calls[0][0]).toMatchObject({range:'A1:B4',template_id:'t1'})
    // 서비스가 무엇을 바꿨는지 그대로 보여 준다. 조용히 삼키면 사람은 슬라이드를
    // 열어 보고 나서야 알게 된다.
    expect(await screen.findByText('마지막 장의 글이 줄었습니다')).toBeTruthy()
    fireEvent.click(screen.getByRole('button',{name:/PowerPoint 내려받기/}))
    await waitFor(()=>expect(download).toHaveBeenCalledWith('deck-1'))
  })

  it('says so when the range cannot be read',async()=>{
    render(<PresentationDialog range={range} onClose={vi.fn()} onPreview={async()=>{throw new Error('범위가 너무 넓습니다.')}} onCreate={vi.fn()} onLoadTemplates={async()=>[]} onDownload={vi.fn()}/>)
    expect(await screen.findByRole('alert')).toHaveTextContent('범위가 너무 넓습니다.')
  })
})
