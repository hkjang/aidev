import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ArrowDownAZ, ArrowUpAZ, Check, Search, X } from 'lucide-react'
import { useDialog } from '../lib/useDialog'
import { api, address } from '../lib/api'
import { parseFilterRange } from '../lib/filter'
import { columnValues, withColumnValues, type FilterValue } from '../lib/columnFilter'
import type { Cell, FilterCriterion, FilterView } from '../types'
import './ColumnFilterMenu.css'

/**
 * The value list behind a column's filter button. It is what people reach for
 * far more often than the filter dialog: tick the values to keep, or sort the
 * column, without leaving the sheet.
 */
export function ColumnFilterMenu({view,sheetId,version,column,label,x,y,onClose,onApply,onSort}:{
  view:FilterView
  sheetId:string
  version:number
  column:number
  label:string
  x:number
  y:number
  onClose:()=>void
  onApply:(criteria:FilterCriterion[])=>Promise<void>
  onSort:(direction:'asc'|'desc')=>void
}){
  // The grid holds only the rows on screen, so the value list is read from the
  // server: filtering by what happens to be loaded would silently drop values.
  const range=useMemo(()=>{
    const bounds=parseFilterRange(view.range)
    return bounds?`${address(bounds.startRow,column)}:${address(bounds.endRow,column)}`:undefined
  },[view.range,column])
  const loaded=useQuery({
    queryKey:['column-filter-values',sheetId,range,version],
    queryFn:()=>api<{items:Cell[]}>(`/api/v1/sheets/${sheetId}/ranges/${range}`),
    enabled:Boolean(range),
  })
  const initial=useMemo(()=>columnValues(loaded.data?.items??[],view,column),[loaded.data,view,column])
  const [values,setValues]=useState<FilterValue[]>([])
  const [ready,setReady]=useState(false)
  if(!ready&&loaded.data){setReady(true);setValues(initial)}
  const [query,setQuery]=useState('')
  const [saving,setSaving]=useState(false)
  const menu=useDialog<HTMLDivElement>(onClose)
  const needle=query.trim().toLowerCase()
  const shown=needle?values.filter(value=>value.label.toLowerCase().includes(needle)):values
  const toggle=(target:FilterValue)=>setValues(current=>current.map(value=>value.label===target.label?{...value,checked:!value.checked}:value))
  // Select all and clear act on what the search is showing, which is how a
  // long list is narrowed down to a handful.
  const setShown=(checked:boolean)=>setValues(current=>current.map(value=>shown.some(item=>item.label===value.label)?{...value,checked}:value))
  const apply=async()=>{
    setSaving(true)
    try{await onApply(withColumnValues(view,column,values));onClose()}finally{setSaving(false)}
  }
  const keptCount=values.filter(value=>value.checked).length
  return <div className="column-filter" ref={menu} role="dialog" aria-label={`${label}열 필터`} style={{left:x,top:y}}>
    <div className="column-filter-sort">
      <button onClick={()=>{onSort('asc');onClose()}}><ArrowUpAZ/> 오름차순 정렬</button>
      <button onClick={()=>{onSort('desc');onClose()}}><ArrowDownAZ/> 내림차순 정렬</button>
    </div>
    <div className="column-filter-search">
      <Search/><input autoFocus aria-label="값 검색" placeholder="값 검색" value={query} onChange={event=>setQuery(event.target.value)}/>
      {query&&<button aria-label="검색 지우기" onClick={()=>setQuery('')}><X/></button>}
    </div>
    <div className="column-filter-bulk">
      <button onClick={()=>setShown(true)}>전체 선택</button>
      <button onClick={()=>setShown(false)}>선택 해제</button>
      <span>{keptCount.toLocaleString()} / {values.length.toLocaleString()}</span>
    </div>
    <div className="column-filter-values" role="group" aria-label="필터 값 목록">
      {!ready&&<p className="empty-hint">값을 읽는 중…</p>}
      {ready&&shown.length===0&&<p className="empty-hint">검색 결과가 없습니다.</p>}
      {shown.map(value=><button key={value.label} role="checkbox" aria-checked={value.checked} onClick={()=>toggle(value)}>
        <i className={value.checked?'checked':undefined}>{value.checked&&<Check/>}</i>
        <span title={value.label}>{value.label}</span>
        <em>{value.count.toLocaleString()}</em>
      </button>)}
    </div>
    <div className="column-filter-actions">
      <button onClick={onClose}>취소</button>
      <button className="primary" disabled={saving||!ready||keptCount===0} onClick={()=>void apply()}>적용</button>
    </div>
    {ready&&keptCount===0&&<p className="column-filter-warning">값을 하나 이상 남겨야 적용할 수 있습니다.</p>}
  </div>
}
