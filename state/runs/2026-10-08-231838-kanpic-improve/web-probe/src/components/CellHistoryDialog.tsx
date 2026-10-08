import { useQuery } from '@tanstack/react-query'
import { History } from 'lucide-react'
import { api } from '../lib/api'
import { useDialog } from '../lib/useDialog'
import type { CellHistory, CellHistorySnapshot } from '../types'
import './CellHistoryDialog.css'

const stamp=(value:string)=>new Date(value).toLocaleString('ko-KR',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'})
const operationLabel=(type:string)=>type.startsWith('structure.')?'행·열 구조 변경':type==='cells.undo'?'실행 취소':type==='cells.redo'?'다시 실행':'셀 편집'

// A snapshot shows the formula when there was one, because the formula is what
// somebody actually typed and the value is only what it produced.
function Snapshot({snapshot}:{snapshot:CellHistorySnapshot}){
  if(snapshot.empty)return <em className="cell-history-empty">빈 셀</em>
  if(snapshot.formula)return <code>{snapshot.formula}</code>
  return <span>{snapshot.value===null||snapshot.value===undefined?'':String(snapshot.value)}</span>
}

/**
 * One cell's story. The version list already tells the workbook's; this answers
 * the question people actually ask, which is who changed this number and what
 * it said before.
 */
export function CellHistoryDialog({sheetId,address,version,onClose}:{sheetId:string;address:string;version:number;onClose:()=>void}){
  const dialog=useDialog<HTMLDivElement>(onClose)
  const history=useQuery({
    queryKey:['cell-history',sheetId,address,version],
    queryFn:()=>api<CellHistory>(`/api/v1/sheets/${sheetId}/cells/${address}/history`),
  })
  const items=history.data?.items??[]
  return <div className="modal-backdrop"><div className="modal cell-history-modal" role="dialog" ref={dialog} aria-modal="true" aria-label={`${address} 편집 기록`}>
    <header><div><History/><div><h2>{address} 편집 기록</h2><p>이 셀을 바꾼 작업만 최신순으로 보여 줍니다.</p></div></div><button aria-label="편집 기록 닫기" onClick={onClose}>×</button></header>
    <div className="cell-history-body">
      {history.isPending?<p className="cell-history-note">기록을 불러오는 중…</p>
        :items.length===0?<p className="cell-history-note">이 셀에는 기록된 편집이 없습니다. 서버에 저장된 작업 기록에서 읽으므로, 이 워크북을 가져오기로 만들었다면 그 이후 편집부터 보입니다.</p>
        :<ol className="cell-history-list">
          {items.map(item=><li key={item.operation_id}>
            <div className="cell-history-head">
              <strong>{item.actor_name||item.actor_id||'알 수 없는 사용자'}</strong>
              <span>{stamp(item.created_at)}</span>
              <em>{operationLabel(item.operation_type)} · v{item.server_version}</em>
            </div>
            <div className="cell-history-change">
              <Snapshot snapshot={item.before}/>
              <span aria-hidden="true">→</span>
              <Snapshot snapshot={item.after}/>
            </div>
          </li>)}
        </ol>}
    </div>
  </div></div>
}
