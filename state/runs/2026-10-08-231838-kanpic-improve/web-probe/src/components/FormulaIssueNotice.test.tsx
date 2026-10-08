import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { FormulaIssueNotice } from './FormulaIssueNotice'

const issue={row:1,column:3,code:'#REF!',message:'formula contains #REF!'}

afterEach(()=>cleanup())

describe('FormulaIssueNotice', () => {
  it('says nothing when the edit broke nothing', () => {
    const {container}=render(<FormulaIssueNotice issues={[]} onOpen={()=>{}} onClose={()=>{}}/>)
    expect(container).toBeEmptyDOMElement()
  })

  // 몇 곳인지만 알려 주고 어디인지 말하지 않으면 찾아다녀야 한다.
  it('names the first broken cell and what happened to it', () => {
    render(<FormulaIssueNotice issues={[issue,{...issue,row:2}]} onOpen={()=>{}} onClose={()=>{}}/>)
    expect(screen.getByRole('status')).toHaveTextContent('수식 2곳이 오류가 되었습니다')
    expect(screen.getByRole('status')).toHaveTextContent('C1 #REF!')
    expect(screen.getByRole('status')).toHaveTextContent('가리키던 셀이 사라졌습니다')
  })

  it('leads to the cell it named', () => {
    const onOpen=vi.fn()
    render(<FormulaIssueNotice issues={[issue]} onOpen={onOpen} onClose={()=>{}}/>)
    fireEvent.click(screen.getByRole('button',{name:'보기'}))
    expect(onOpen).toHaveBeenCalledWith(issue)
  })

  // 행·열 삭제는 실행 취소로 되살릴 수 없다. 회수 경로를 알려 주지 않으면
  // 사용자는 지워진 줄을 손으로 다시 입력한다.
  it('offers to revert a deletion even when no formula broke', () => {
    const onRevert=vi.fn()
    const backup={versionId:'v-1',summary:'행 3'}
    render(<FormulaIssueNotice issues={[]} backup={backup} onOpen={()=>{}} onRevert={onRevert} onClose={()=>{}}/>)
    expect(screen.getByRole('status')).toHaveTextContent('행 3을(를) 삭제했습니다')
    expect(screen.getByRole('status')).toHaveTextContent('실행 취소로는 되돌릴 수 없습니다')
    fireEvent.click(screen.getByRole('button',{name:'되돌리기'}))
    expect(onRevert).toHaveBeenCalledWith(backup)
  })

  it('offers both the broken cell and the way back when a deletion did both', () => {
    render(<FormulaIssueNotice issues={[issue]} backup={{versionId:'v-1',summary:'행 3'}} onOpen={()=>{}} onRevert={()=>{}} onClose={()=>{}}/>)
    expect(screen.getByRole('button',{name:'보기'})).toBeInTheDocument()
    expect(screen.getByRole('button',{name:'되돌리기'})).toBeInTheDocument()
  })

  // 셀 변경 트리거가 실패해도 편집한 사람은 아무것도 보지 못했다. 편집은
  // 저장되었으므로 실패를 알리되 편집이 취소된 것처럼 읽히면 안 된다.
  it('names the automation this edit set off and failed', () => {
    render(<FormulaIssueNotice issues={[]} automations={[{automation_id:'a-1',run_id:'r-1',message:'automation exceeds the 1 cell limit'}]} onOpen={()=>{}} onClose={()=>{}}/>)
    expect(screen.getByRole('status')).toHaveTextContent('자동화가 실패했습니다')
    expect(screen.getByRole('status')).toHaveTextContent('1 cell limit')
    expect(screen.getByRole('status')).toHaveTextContent('이 편집은 저장되었습니다')
  })
})
