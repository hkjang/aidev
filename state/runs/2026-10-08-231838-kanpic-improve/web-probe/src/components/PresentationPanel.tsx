import { Download,ExternalLink,Plus,Presentation,RefreshCw } from 'lucide-react'
import { useState } from 'react'
import './PresentationPanel.css'

export type PresentationRecord={
  id:string; provider:string; workbook_id:string; sheet_id:string; range:string
  source_version:number; title:string; template?:string; slide_count:number
  edit_url?:string; created_by:string; created_at:string; updated_at:string; stale:boolean
}

const when=(value:string)=>{
  const at=new Date(value)
  return Number.isNaN(at.getTime())?'':at.toLocaleString('ko-KR',{month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit'})
}

export function PresentationPanel({items,sheetNames,onClose,onCreate,onRefresh,onDownload}:{
  items:PresentationRecord[]
  sheetNames:Map<string,string>
  onClose:()=>void
  onCreate:()=>void
  onRefresh:(record:PresentationRecord)=>Promise<void>
  onDownload:(record:PresentationRecord)=>Promise<void>
}){
  const [busy,setBusy]=useState<string>()
  const [failed,setFailed]=useState<{id:string;message:string}>()
  const run=async(record:PresentationRecord,action:(record:PresentationRecord)=>Promise<void>)=>{
    setBusy(record.id);setFailed(undefined)
    try{await action(record)}
    catch(problem){setFailed({id:record.id,message:problem instanceof Error?problem.message:'요청을 처리하지 못했습니다.'})}
    finally{setBusy(undefined)}
  }
  return <aside className="presentation-panel">
    <header><span><Presentation/> 프레젠테이션</span><button aria-label="프레젠테이션 패널 닫기" onClick={onClose}>×</button></header>
    <div className="presentation-panel-intro">
      <p>이 워크북의 범위로 만든 덱입니다. 원본이 바뀌면 같은 덱을 그대로 다시 만들 수 있습니다.</p>
      <button className="primary" onClick={onCreate}><Plus/> 새 프레젠테이션</button>
    </div>
    <div className="presentation-panel-list">
      {items.length===0
        ?<div className="presentation-panel-empty"><Presentation/><strong>만든 프레젠테이션이 없습니다</strong><span>범위를 선택한 뒤 새로 만드세요.</span></div>
        :items.map(record=><article key={record.id} className={record.stale?'stale':''}>
          <div className="presentation-panel-head">
            <strong>{record.title||record.range}</strong>
            {record.stale&&<em title="덱을 만든 뒤 워크북이 바뀌었습니다">원본 변경됨</em>}
          </div>
          <small>{sheetNames.get(record.sheet_id)??'삭제된 시트'}!{record.range} · {record.slide_count}장{record.template?` · ${record.template}`:''}</small>
          <small className="presentation-panel-when">{when(record.updated_at)} · 워크북 v{record.source_version} 기준</small>
          {failed?.id===record.id&&<p className="presentation-panel-error" role="alert">{failed.message}</p>}
          <div className="presentation-panel-actions">
            <button disabled={busy===record.id} onClick={()=>void run(record,onRefresh)} title="지금 값으로 다시 만들기"><RefreshCw/> {busy===record.id?'다시 만드는 중…':'다시 만들기'}</button>
            <button disabled={busy===record.id} onClick={()=>void run(record,onDownload)} title="PowerPoint 내려받기"><Download/> PPTX</button>
            {record.edit_url&&<a href={record.edit_url} target="_blank" rel="noreferrer" title="편집기에서 열기"><ExternalLink/></a>}
          </div>
        </article>)}
    </div>
  </aside>
}
