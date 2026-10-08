import { cleanup,fireEvent,render,screen,waitFor } from '@testing-library/react'
import { afterEach,describe,expect,it,vi } from 'vitest'
import type { ConditionalFormat } from '../types'
import { ConditionalFormatDialog } from './ConditionalFormatDialog'

const rule:ConditionalFormat={id:'format-1',workbook_id:'book-1',workbook_version:3,sheet_id:'sheet-1',name:'중복 강조',range:'A1:A3',rule_type:'duplicate',operator:'duplicate',style:{background:'#fef3c7',color:'#92400e',bold:true},priority:2,stop_if_true:true,revision:1,created_by:'alice',updated_by:'alice',created_at:'2026-08-02T00:00:00Z',updated_at:'2026-08-02T00:00:00Z'}

afterEach(()=>{cleanup();vi.unstubAllGlobals()})

describe('ConditionalFormatDialog',()=>{
  it('creates a color scale for the selected range',async()=>{
    const create=vi.fn().mockResolvedValue({...rule,id:'format-2',rule_type:'color_scale',range:'B2:D8'})
    render(<ConditionalFormatDialog range={{startRow:2,startColumn:2,endRow:8,endColumn:4}} rules={[]} onClose={vi.fn()} onCreate={create} onUpdate={vi.fn()} onDelete={vi.fn()}/>)
    fireEvent.change(screen.getByLabelText('조건부 서식 이름'),{target:{value:'점수 분포'}})
    fireEvent.change(screen.getByLabelText('조건부 서식 유형'),{target:{value:'color_scale'}})
    fireEvent.click(screen.getByLabelText('중간 색상 사용'))
    fireEvent.change(screen.getByLabelText('최솟값 색상'),{target:{value:'#dcfce7'}})
    fireEvent.change(screen.getByLabelText('중간값 색상'),{target:{value:'#fef3c7'}})
    fireEvent.change(screen.getByLabelText('최댓값 색상'),{target:{value:'#ef4444'}})
    fireEvent.click(screen.getByText('규칙 저장'))
    await waitFor(()=>expect(create).toHaveBeenCalledTimes(1))
    expect(create.mock.calls[0][0]).toMatchObject({name:'점수 분포',range:'B2:D8',rule_type:'color_scale',min_color:'#dcfce7',mid_color:'#fef3c7',max_color:'#ef4444',priority:1,stop_if_true:false})
  })

  // 아이콘 규칙은 색이 아니라 종류와 방향을 보낸다. 막대 색이 함께 가면
  // 서버가 규칙을 데이터 막대로 오해할 여지가 생긴다.
  it('creates an icon set without the settings other rule types use',async()=>{
    const create=vi.fn().mockResolvedValue({...rule,id:'format-3',rule_type:'icon_set',icon_style:'5Arrows'})
    render(<ConditionalFormatDialog range={{startRow:1,startColumn:3,endRow:20,endColumn:3}} rules={[]} onClose={vi.fn()} onCreate={create} onUpdate={vi.fn()} onDelete={vi.fn()}/>)
    fireEvent.change(screen.getByLabelText('조건부 서식 유형'),{target:{value:'icon_set'}})
    fireEvent.change(screen.getByLabelText('아이콘 종류'),{target:{value:'5Arrows'}})
    fireEvent.click(screen.getByLabelText('아이콘 순서 뒤집기'))
    expect(screen.getByLabelText('화살표 5개 미리보기')).toBeTruthy()
    fireEvent.click(screen.getByText('규칙 저장'))
    await waitFor(()=>expect(create).toHaveBeenCalledTimes(1))
    expect(create.mock.calls[0][0]).toMatchObject({range:'C1:C20',rule_type:'icon_set',icon_style:'5Arrows',icon_reverse:true,stop_if_true:false})
    expect(create.mock.calls[0][0].bar_color).toBeUndefined()
    expect(create.mock.calls[0][0].style).toBeUndefined()
  })

  it('updates a duplicate rule with its revision and can delete it',async()=>{
    const update=vi.fn().mockResolvedValue({...rule,name:'반복 값',revision:2})
    const remove=vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('confirm',vi.fn(()=>true))
    render(<ConditionalFormatDialog range={{startRow:1,startColumn:1,endRow:1,endColumn:1}} rules={[rule]} onClose={vi.fn()} onCreate={vi.fn()} onUpdate={update} onDelete={remove}/>)
    fireEvent.click(screen.getByText('중복 강조'))
    fireEvent.change(screen.getByLabelText('조건부 서식 이름'),{target:{value:'반복 값'}})
    fireEvent.change(screen.getByLabelText('중복 값 조건'),{target:{value:'unique'}})
    fireEvent.click(screen.getByText('규칙 저장'))
    await waitFor(()=>expect(update).toHaveBeenCalledTimes(1))
    expect(update.mock.calls[0][0]).toBe(rule.id)
    expect(update.mock.calls[0][1]).toMatchObject({name:'반복 값',operator:'unique',rule_type:'duplicate',expected_revision:1})
    fireEvent.click(screen.getByText('중복 강조'))
    fireEvent.click(screen.getByText('삭제'))
    await waitFor(()=>expect(remove).toHaveBeenCalledWith(rule))
  })
})
