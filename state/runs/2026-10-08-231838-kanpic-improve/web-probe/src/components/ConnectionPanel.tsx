import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AlertTriangle, Link2, RefreshCw, X } from 'lucide-react'
import { api } from '../lib/api'
import type { WorkbookConnections } from '../types'
import './ConnectionPanel.css'

const timeLabel=(value:string)=>new Date(value).toLocaleTimeString('ko-KR',{hour:'2-digit',minute:'2-digit',second:'2-digit'})

/**
 * IMPORTRANGE pulls data that lives somewhere else, so the two questions a
 * reader always has are "is this current" and "may I still see it". The panel
 * answers both: every connection with its live state, and one button that
 * re-reads all of them.
 */
export function ConnectionPanel({workbookId,version,readOnly,onClose,onRefreshed}:{
  workbookId:string
  version:number
  readOnly:boolean
  onClose:()=>void
  onRefreshed:(result:WorkbookConnections)=>void
}){
  const [refreshing,setRefreshing]=useState(false)
  const [refreshedAt,setRefreshedAt]=useState<string>()
  const connections=useQuery({
    queryKey:['connections',workbookId,version],
    queryFn:()=>api<WorkbookConnections>(`/api/v1/workbooks/${workbookId}/connections`),
  })
  const items=connections.data?.items??[]
  const broken=items.filter(item=>item.status!=='ok').length
  const refresh=async()=>{
    setRefreshing(true)
    try{
      const result=await api<WorkbookConnections>(`/api/v1/workbooks/${workbookId}/connections:refresh`,{method:'POST',body:'{}'})
      setRefreshedAt(result.refreshed_at??result.checked_at)
      onRefreshed(result)
    }catch(error){
      alert(error instanceof Error?error.message:'연결을 새로 고치지 못했습니다.')
    }finally{setRefreshing(false)}
  }
  return <section className="connection-panel" aria-label="데이터 연결">
    <header>
      <strong><Link2/> 데이터 연결</strong>
      <span>
        <button aria-label="연결 새로 고침" title="원본을 다시 읽습니다" disabled={readOnly||refreshing||items.length===0} onClick={()=>void refresh()}><RefreshCw className={refreshing?'spinning':''}/></button>
        <button aria-label="데이터 연결 패널 닫기" onClick={onClose}><X/></button>
      </span>
    </header>
    <div className="connection-body">
      {connections.isPending?<p className="connection-note">연결을 확인하는 중…</p>
        :items.length===0?<p className="connection-note">이 워크북에는 다른 워크북에서 가져오는 데이터가 없습니다. <code>=IMPORTRANGE("워크북 주소","시트!범위")</code> 로 만들 수 있습니다.</p>
        :<ul className="connection-list">
          {items.map(item=><li key={`${item.source}|${item.range}`} className={item.status==='ok'?'':'broken'}>
            <div className="connection-title">
              {item.status!=='ok'&&<AlertTriangle/>}
              <strong>{item.title||item.workbook_id||item.source}</strong>
              <code>{item.range}</code>
            </div>
            {item.status==='ok'
              ? <p className="connection-meta">{item.rows?.toLocaleString()}행 × {item.columns?.toLocaleString()}열 · {item.cells.length}개 셀에서 사용 중</p>
              : <p className="connection-error">{item.message}</p>}
            {item.cells.length>0&&<p className="connection-cells">{item.cells.slice(0,6).join(', ')}{item.cells.length>6?` 외 ${item.cells.length-6}곳`:''}</p>}
          </li>)}
        </ul>}
    </div>
    <footer>
      {broken>0?<span className="connection-warning">{broken}개 연결에 문제가 있습니다</span>
        :items.length>0?<span>{items.length}개 연결이 정상입니다</span>
        :<span/>}
      {refreshedAt&&<span className="connection-stamp">{timeLabel(refreshedAt)} 갱신</span>}
      {!refreshedAt&&connections.data&&<span className="connection-stamp">{timeLabel(connections.data.checked_at)} 확인</span>}
    </footer>
  </section>
}
