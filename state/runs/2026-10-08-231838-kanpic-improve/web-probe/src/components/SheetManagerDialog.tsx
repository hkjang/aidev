import { useQuery } from '@tanstack/react-query'
import { PromptDialog, type PromptRequest } from './PromptDialog'
import { useState } from 'react'
import { ArrowDown, ArrowUp, Eye, EyeOff, Sigma, SquareArrowOutUpRight, Table2, Trash2, X } from 'lucide-react'
import { api } from '../lib/api'
import type { Sheet, SheetStats, Workbook } from '../types'
import './SheetManagerDialog.css'
import { useDialog } from '../lib/useDialog'

function cellAddress(row:number,column:number){
  if(row<1||column<1)return '없음'
  let value=column,letters=''
  while(value>0){value-=1;letters=String.fromCharCode(65+(value%26))+letters;value=Math.floor(value/26)}
  return `${letters}${row}`
}

/**
 * One place to see and manage every sheet in a workbook: how much data each one
 * holds, where that data ends, whether it is hidden, and the reordering,
 * visibility, copy and delete actions.
 */
export function SheetManagerDialog({workbook,sheets,activeSheetId,readOnly=false,onClose,onSelect,onRename,onMove,onHidden,onDelete,onCopyTo}:{
  workbook:Workbook
  sheets:Sheet[]
  activeSheetId:string
  readOnly?:boolean
  onClose:()=>void
  onSelect:(sheet:Sheet)=>void
  onRename:(sheet:Sheet,name:string)=>Promise<void>
  onMove:(sheet:Sheet,position:number)=>Promise<void>
  onHidden:(sheet:Sheet,hidden:boolean)=>Promise<void>
  onDelete:(sheet:Sheet)=>Promise<void>
  onCopyTo:(sheet:Sheet)=>void
}){
  const [pending,setPending]=useState(false),[error,setError]=useState(''),[prompt,setPrompt]=useState<PromptRequest>()
  const stats=useQuery({queryKey:['sheet-stats',workbook.id,workbook.version],queryFn:()=>api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbook.id}/sheet-stats`)})
  const byID=new Map((stats.data?.items??[]).map(item=>[item.sheet_id,item]))
  const ordered=[...sheets].sort((left,right)=>left.position-right.position)
  const visibleCount=ordered.filter(sheet=>!sheet.hidden).length
  const run=async(action:()=>Promise<void>)=>{
    setPending(true);setError('')
    try{await action();await stats.refetch()}
    catch(reason){setError(reason instanceof Error?reason.message:'시트를 변경하지 못했습니다.')}
    finally{setPending(false)}
  }
  const totals=(stats.data?.items??[]).reduce((sum,item)=>({cells:sum.cells+item.non_empty_cells,formulas:sum.formulas+item.formula_cells}),{cells:0,formulas:0})
  const dialog=useDialog<HTMLElement>(onClose)
  if(prompt)return <PromptDialog request={prompt} onClose={()=>setPrompt(undefined)}/>
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal sheet-manager" role="dialog" ref={dialog as React.RefObject<any>} aria-modal="true" aria-label="시트 관리">
      <header><div><h2>시트 관리</h2><p>{ordered.length}개 시트 · 데이터 {totals.cells.toLocaleString()}셀 · 수식 {totals.formulas.toLocaleString()}개</p></div><button aria-label="시트 관리 닫기" onClick={onClose}><X/></button></header>
      <div className="sheet-manager-table" role="table">
        <div className="sheet-manager-head" role="row"><span>시트</span><span>데이터</span><span>사용 범위</span><span>마지막 변경</span><span>작업</span></div>
        {ordered.map((sheet,index)=>{
          const item=byID.get(sheet.id)
          return <div className={`sheet-manager-row${sheet.id===activeSheetId?' active':''}${sheet.hidden?' hidden-sheet':''}`} role="row" key={sheet.id}>
            <span className="sheet-manager-name">
              <i className={sheet.color?'':'empty'} style={sheet.color?{background:sheet.color}:undefined}/>
              <button className="sheet-manager-open" onClick={()=>{if(!sheet.hidden){onSelect(sheet);onClose()}}} disabled={sheet.hidden} title={sheet.hidden?'숨긴 시트는 표시한 뒤 열 수 있습니다':'이 시트로 이동'}>{sheet.name}</button>
              {sheet.hidden&&<em>숨김</em>}
              {!readOnly&&<button className="sheet-manager-rename" aria-label={`${sheet.name} 이름 변경`} onClick={()=>setPrompt({
                title:'시트 이름 변경',label:'시트 이름',value:sheet.name,confirmLabel:'이름 바꾸기',
                validate:value=>value.trim()===''?'이름을 입력하세요.':undefined,
                onSubmit:value=>{if(value.trim()!==sheet.name)void run(()=>onRename(sheet,value.trim()))},
              })}>이름</button>}
            </span>
            <span>{item?`${item.non_empty_cells.toLocaleString()}셀`:'—'}{item&&item.formula_cells>0?<small><Sigma/> {item.formula_cells.toLocaleString()}</small>:null}</span>
            <span>{item&&item.max_row>0?`A1:${cellAddress(item.max_row,item.max_column)}`:'비어 있음'}</span>
            <span>{item?.updated_at?new Date(item.updated_at).toLocaleString('ko-KR'):'—'}</span>
            <span className="sheet-manager-actions">
              <button aria-label={`${sheet.name} 왼쪽으로 이동`} disabled={pending||readOnly||index===0} onClick={()=>void run(()=>onMove(sheet,sheet.position-1))}><ArrowUp/></button>
              <button aria-label={`${sheet.name} 오른쪽으로 이동`} disabled={pending||readOnly||index===ordered.length-1} onClick={()=>void run(()=>onMove(sheet,sheet.position+1))}><ArrowDown/></button>
              <button aria-label={sheet.hidden?`${sheet.name} 표시`:`${sheet.name} 숨기기`} disabled={pending||readOnly||(!sheet.hidden&&visibleCount===1)} onClick={()=>void run(()=>onHidden(sheet,!sheet.hidden))}>{sheet.hidden?<Eye/>:<EyeOff/>}</button>
              <button aria-label={`${sheet.name} 다른 워크북으로 복사`} disabled={pending} onClick={()=>onCopyTo(sheet)}><SquareArrowOutUpRight/></button>
              <button className="danger" aria-label={`${sheet.name} 삭제`} disabled={pending||readOnly||ordered.length===1} onClick={()=>{
                if(window.confirm(`'${sheet.name}' 시트와 모든 셀을 삭제할까요?`))void run(()=>onDelete(sheet))
              }}><Trash2/></button>
            </span>
          </div>
        })}
      </div>
      {stats.isLoading&&<div className="sheet-manager-loading">시트 통계를 불러오는 중…</div>}
      {error&&<div className="sheet-manager-error" role="alert">{error}</div>}
      <footer><span><Table2/> 숨긴 시트는 탭에 표시되지 않지만 수식과 참조는 그대로 계산됩니다.</span><button className="primary" onClick={onClose}>닫기</button></footer>
    </section>
  </div>
}

/** Copies one sheet into another workbook the user can edit. */
export function CopySheetDialog({sheet,workbookId,onClose,onCopied}:{sheet:Sheet;workbookId:string;onClose:()=>void;onCopied:(target:Workbook)=>void}){
  const [target,setTarget]=useState(''),[name,setName]=useState(''),[pending,setPending]=useState(false),[error,setError]=useState('')
  const workbooks=useQuery({queryKey:['workbooks'],queryFn:()=>api<{items:Workbook[]}>('/api/v1/workbooks')})
  const candidates=(workbooks.data?.items??[]).filter(item=>item.access_role==='owner'||item.access_role==='editor')
  const copyDialog=useDialog<HTMLElement>(onClose)
  const copy=async()=>{
    const chosen=candidates.find(item=>item.id===target)
    if(!chosen)return
    setPending(true);setError('')
    try{
      await api<Sheet>(`/api/v1/sheets/${sheet.id}/copy`,{method:'POST',body:JSON.stringify({target_workbook_id:chosen.id,name:name.trim()||undefined})})
      onCopied(chosen)
    }catch(reason){setError(reason instanceof Error?reason.message:'시트를 복사하지 못했습니다.')}
    finally{setPending(false)}
  }
  return <div className="modal-backdrop" role="presentation" onMouseDown={event=>{if(event.target===event.currentTarget)onClose()}}>
    <section className="modal copy-sheet-modal" ref={copyDialog as React.RefObject<any>} role="dialog" aria-modal="true" aria-label="다른 워크북으로 복사">
      <h2>다른 워크북으로 복사</h2>
      <p>‘{sheet.name}’ 시트를 셀·서식·레이아웃과 함께 복사합니다. 편집 권한이 있는 워크북만 선택할 수 있습니다.</p>
      <label>대상 워크북
        <select aria-label="대상 워크북" value={target} onChange={event=>setTarget(event.target.value)}>
          <option value="">워크북을 선택하세요</option>
          {candidates.map(item=><option key={item.id} value={item.id}>{item.title}{item.id===workbookId?' (현재 워크북)':''}</option>)}
        </select>
      </label>
      <label>새 시트 이름 (선택)
        <input aria-label="새 시트 이름" maxLength={100} placeholder={sheet.name} value={name} onChange={event=>setName(event.target.value)}/>
      </label>
      {candidates.length===0&&<div className="copy-sheet-empty">편집 권한이 있는 다른 워크북이 없습니다.</div>}
      {error&&<div className="sheet-manager-error" role="alert">{error}</div>}
      <div className="modal-actions"><button className="secondary" onClick={onClose}>취소</button><button className="primary" disabled={pending||!target} onClick={()=>void copy()}>{pending?'복사 중…':'복사'}</button></div>
    </section>
  </div>
}
