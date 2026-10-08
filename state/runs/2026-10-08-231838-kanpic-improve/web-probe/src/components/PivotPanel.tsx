import { CornerDownRight, Plus, RefreshCw, Settings2, Table2 } from 'lucide-react'
import { useState } from 'react'
import type { Pivot, Sheet } from '../types'
import './PivotPanel.css'

export function PivotPanel({pivots,sheets,onClose,onCreate,onEdit,onOpen,onRefresh,onNavigate}:{pivots:Pivot[];sheets:Sheet[];onClose:()=>void;onCreate:()=>void;onEdit:(pivot:Pivot)=>void;onOpen:(pivot:Pivot)=>void;onRefresh:(pivot:Pivot)=>Promise<void>;onNavigate:(pivot:Pivot)=>void}){
  const [refreshing,setRefreshing]=useState('')
  const refresh=async(pivot:Pivot)=>{setRefreshing(pivot.id);try{await onRefresh(pivot)}catch(error){alert(error instanceof Error?error.message:'피벗을 갱신하지 못했습니다.')}finally{setRefreshing('')}}
  return <aside className="pivot-panel"><header><span><Table2/> 피벗 테이블</span><button aria-label="피벗 패널 닫기" onClick={onClose}>×</button></header><div className="pivot-panel-intro"><p>원본 범위를 행·열로 그룹화하고 집계 결과에서 원본 행까지 드릴다운합니다.</p><button className="primary" onClick={onCreate}><Plus/> 새 피벗</button></div><div className="pivot-panel-list">{pivots.length===0?<div className="pivot-panel-empty"><Table2/><strong>이 시트에 피벗이 없습니다</strong><span>범위를 선택한 뒤 새 피벗을 만드세요.</span></div>:pivots.map(pivot=><article key={pivot.id}><span className="pivot-type"><Table2/></span><div><strong>{pivot.name}</strong><small>{sheets.find(sheet=>sheet.id===pivot.source_sheet_id)?.name??'삭제된 시트'}!{pivot.source_range} · {pivot.refresh_mode==='auto'?'자동':'수동'} · r{pivot.revision}</small></div><button title="결과 열기" onClick={()=>onOpen(pivot)}><Table2/></button><button title="원본 범위로 이동" onClick={()=>onNavigate(pivot)} disabled={!pivot.source_sheet_id||pivot.source_range==='#REF!'}><CornerDownRight/></button><button title="지금 갱신" disabled={refreshing===pivot.id||pivot.source_range==='#REF!'} onClick={()=>void refresh(pivot)}><RefreshCw className={refreshing===pivot.id?'spin':''}/></button><button title="피벗 설정" onClick={()=>onEdit(pivot)}><Settings2/></button></article>)}</div></aside>
}
