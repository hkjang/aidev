import { useQuery } from '@tanstack/react-query'
import { BarChart3 } from 'lucide-react'
import { api, address } from '../lib/api'
import { columnStats, looksLikeHeader, statNumber } from '../lib/columnStats'
import type { Cell, SheetStats } from '../types'
import './ColumnStatsPanel.css'

const columnName=(column:number)=>address(1,column).replace(/\d+$/,'')

/**
 * What a data review starts with: how much of a column is filled, how many
 * distinct values it holds, which values repeat and where the numbers sit.
 * The rows come from the server rather than the loaded tiles, so the summary
 * covers the whole column even when only part of it has been scrolled to.
 */
export function ColumnStatsPanel({workbookId,workbookVersion,sheetId,column,onClose}:{
  workbookId:string
  workbookVersion:number
  sheetId:string
  column:number
  onClose:()=>void
}){
  const sheets=useQuery({queryKey:['sheet-stats',workbookId,workbookVersion],queryFn:()=>api<{items:SheetStats[]}>(`/api/v1/workbooks/${workbookId}/sheet-stats`)})
  const used=sheets.data?.items.find(item=>item.sheet_id===sheetId)
  const lastRow=Math.max(1,Math.min(used?.max_row??1,20_000))
  const range=`${address(1,column)}:${address(lastRow,column)}`
  const cells=useQuery({
    queryKey:['column-stats',sheetId,column,lastRow,workbookVersion],
    queryFn:()=>api<{items:Cell[]}>(`/api/v1/sheets/${sheetId}/ranges/${range}`),
    enabled:Boolean(used),
  })
  const rows=cells.data?.items??[]
  // A label on top of numbers is not one of the values being summarised.
  const header=looksLikeHeader(rows,column,1,lastRow)
  const firstRow=header?2:1
  const stats=columnStats(rows,column,firstRow,lastRow)
  const busy=sheets.isLoading||cells.isLoading
  const widest=Math.max(1,...stats.frequent.map(item=>item.count))
  const tallest=Math.max(1,...stats.buckets.map(bucket=>bucket.count))
  return <section className="side-panel stats-panel" aria-label="열 통계 패널">
    <header className="side-panel-head">
      <span><BarChart3/> {columnName(column)}열 통계</span>
      <button aria-label="열 통계 닫기" onClick={onClose}>×</button>
    </header>
    <div className="side-panel-body">
      {busy&&<p className="empty-hint">열을 읽는 중…</p>}
      {!busy&&<>
        <p className="stats-scope">{firstRow}행부터 {lastRow.toLocaleString()}행까지 검사했습니다.{header?' 1행은 머리글로 보고 제외했습니다.':''}</p>
        <div className="stats-grid">
          <div><small>값 있음</small><strong>{stats.filled.toLocaleString()}</strong></div>
          <div><small>빈 셀</small><strong>{stats.empty.toLocaleString()}</strong></div>
          <div><small>고유값</small><strong>{stats.unique.toLocaleString()}</strong></div>
          <div><small>숫자</small><strong>{stats.numbers.toLocaleString()}</strong></div>
        </div>
        {stats.numbers>0&&<div className="stats-grid">
          <div><small>합계</small><strong>{statNumber(stats.sum)}</strong></div>
          <div><small>평균</small><strong>{statNumber(stats.average)}</strong></div>
          <div><small>중앙값</small><strong>{statNumber(stats.median)}</strong></div>
          <div><small>표준편차</small><strong>{statNumber(stats.deviation)}</strong></div>
          <div><small>최소</small><strong>{statNumber(stats.min)}</strong></div>
          <div><small>최대</small><strong>{statNumber(stats.max)}</strong></div>
        </div>}
        {stats.buckets.length>1&&<section className="stats-section">
          <h3>분포</h3>
          <div className="stats-histogram" role="img" aria-label={`${statNumber(stats.min)}부터 ${statNumber(stats.max)}까지의 분포`}>
            {stats.buckets.map((bucket,index)=><i key={index} style={{height:`${Math.round(bucket.count/tallest*100)}%`}}
              title={`${statNumber(bucket.from)} ~ ${statNumber(bucket.to)} · ${bucket.count.toLocaleString()}개`}/>)}
          </div>
          <div className="stats-axis"><span>{statNumber(stats.min)}</span><span>{statNumber(stats.max)}</span></div>
        </section>}
        {stats.frequent.length>0&&<section className="stats-section">
          <h3>자주 나오는 값</h3>
          {stats.frequent.map(item=><div className="stats-row" key={item.value}>
            <span title={item.value}>{item.value}</span>
            <i style={{width:`${Math.round(item.count/widest*100)}%`}}/>
            <em>{item.count.toLocaleString()} · {Math.round(item.share*100)}%</em>
          </div>)}
        </section>}
        {stats.filled===0&&<p className="empty-hint">이 열에는 값이 없습니다.</p>}
      </>}
    </div>
  </section>
}
